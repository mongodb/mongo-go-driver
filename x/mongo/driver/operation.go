package driver

import (
	"bytes"
	"compress/zlib"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/golang/snappy"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	wiremessagex "go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
	"go.mongodb.org/mongo-driver/x/mongo/driverlegacy/session"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
)

var dollarCmd = [...]byte{'.', '$', 'c', 'm', 'd'}

var (
	// ErrNoDocCommandResponse occurs when the server indicated a response existed, but none was found.
	ErrNoDocCommandResponse = errors.New("command returned no documents")
	// ErrMultiDocCommandResponse occurs when the server sent multiple documents in response to a command.
	ErrMultiDocCommandResponse = errors.New("command returned multiple documents")
)

// OperationContext is used to execute an operation. It contains all of the common code required to
// select a server, transform an operation into a command, write the command to a connection from
// the selected server, read a response from that connection, process the response, and potentially
// retry.
//
// The required fields are Database, CommandFn, and either Deployment or ServerSelector and
// TopologyKind. All other fields are optional.
type OperationContext struct {
	CommandFn func(dst []byte, desc description.SelectedServer) ([]byte, error)
	Database  string

	Deployment   Deployment
	Server       Server
	TopologyKind description.TopologyKind

	ProcessResponseFn func(response bsoncore.Document, srvr Server) error
	RetryableFn       func(description.Server) RetryType

	Selector       description.ServerSelector
	ReadPreference *readpref.ReadPref
	ReadConcern    *readconcern.ReadConcern
	WriteConcern   *writeconcern.WriteConcern

	Client *session.Client
	Clock  *session.ClusterClock

	RetryMode *RetryMode
	Batches   *Batches

	RetryType RetryType
}

func (oc OperationContext) selectServer(ctx context.Context) (Server, error) {
	if err := oc.Validate(); err != nil {
		return nil, err
	}

	if oc.Server != nil {
		return oc.Server, nil
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	rp := oc.ReadPreference
	if rp == nil {
		rp = readpref.Primary()
	}

	return oc.Deployment.SelectServer(ctx, createReadPrefSelector(rp, oc.Selector))
}

// Validate validates this operation, ensuring the fields are set properly.
func (oc OperationContext) Validate() error {
	if oc.CommandFn == nil {
		return errors.New("the CommandFn field must be set on OperationContext")
	}
	if (oc.Deployment == nil) && (oc.Server == nil && oc.TopologyKind == description.TopologyKind(0)) {
		return errors.New("the Deployment field or the Server and TopologyKind fields must be set on OperationContext")
	}
	if oc.Database == "" {
		return errors.New("the Database field must be non-empty on OperationContext")
	}
	return nil
}

// Execute runs this operation.
func (oc OperationContext) Execute(ctx context.Context) error {
	err := oc.Validate()
	if err != nil {
		return err
	}

	srvr, err := oc.selectServer(ctx)
	if err != nil {
		return err
	}

	conn, err := srvr.Connection(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	kind := oc.TopologyKind
	if oc.Deployment != nil {
		kind = oc.Deployment.Kind()
	}
	desc := description.SelectedServer{Server: conn.Description(), Kind: kind}

	var retryable RetryType
	if oc.RetryableFn != nil {
		retryable = oc.RetryableFn(desc.Server)
	}
	if retryable == RetryWrite && oc.Client != nil {
		oc.Client.RetryWrite = true
		oc.Client.IncrementTxnNumber()
	}

	var res bsoncore.Document
	var operationErr WriteCommandError
	var original error
	var retries int
	// TODO(GODRIVER-617): Add support for retryable reads.
	if retryable == RetryWrite && oc.RetryMode != nil {
		switch *oc.RetryMode {
		case RetryOnce, RetryOncePerCommand:
			retries = 1
		case RetryContext:
			retries = -1
		}
	}
	batching := oc.Batches.Valid()
	for {
		if batching {
			err = oc.Batches.AdvanceBatch(int(desc.MaxBatchCount), int(desc.MaxDocumentSize))
			if err != nil {
				return err
			}
		}

		// convert to wire message
		wm, err := oc.createWireMessage(nil, desc)
		if err != nil {
			return err
		}

		// compress wiremessage if allowed
		if compressor, ok := conn.(Compressor); ok && oc.canCompress("") {
			wm, err = compressor.CompressWireMessage(wm, nil)
			if err != nil {
				return err
			}
		}

		// roundtrip
		res, err = oc.roundTrip(ctx, conn, wm)

		// Pull out $clusterTime and operationTime and update session and clock. We handle this before
		// handling the error to ensure we are properly gossiping the cluster time.
		_ = updateClusterTimes(oc.Client, oc.Clock, res)
		_ = updateOperationTime(oc.Client, res)
		if err != nil {
			return err
		}

		var perr error
		if oc.ProcessResponseFn != nil {
			perr = oc.ProcessResponseFn(res, srvr)
		}
		switch tt := err.(type) {
		case WriteCommandError:
			if retryable == RetryWrite && tt.Retryable() && retries != 0 {
				retries--
				original, err = err, nil
				conn.Close() // Avoid leaking the connection.
				srvr, err = oc.selectServer(ctx)
				if err != nil {
					return original
				}
				conn, err := srvr.Connection(ctx)
				// We know that oc.RetryableFn is not nil because retryable is a valid retryable
				// value.
				if err != nil || conn == nil || oc.RetryableFn(conn.Description()) == RetryWrite {
					return original
				}
				defer conn.Close() // Avoid leaking the new connection.
				continue
			}
			if batching && oc.Batches.Ordered != nil && *oc.Batches.Ordered == true && len(tt.WriteErrors) > 0 {
				return tt
			}
			operationErr.WriteConcernError = tt.WriteConcernError
			operationErr.WriteErrors = append(operationErr.WriteErrors, tt.WriteErrors...)
		case Error:
			if retryable == RetryWrite && tt.Retryable() && retries != 0 {
				retries--
				original, err = err, nil
				conn.Close() // Avoid leaking the connection.
				srvr, err = oc.selectServer(ctx)
				if err != nil {
					return original
				}
				conn, err := srvr.Connection(ctx)
				// We know that oc.RetryableFn is not nil because retryable is a valid retryable
				// value.
				if err != nil || conn == nil || oc.RetryableFn(conn.Description()) == RetryWrite {
					return original
				}
				defer conn.Close() // Avoid leaking the new connection.
				continue
			}
			return err
		case nil:
			if perr != nil {
				return perr
			}
		default:
			return err
		}

		if batching && len(oc.Batches.Documents) > 0 {
			if retryable == RetryWrite && oc.Client != nil {
				oc.Client.IncrementTxnNumber()
				if oc.RetryMode != nil && *oc.RetryMode == RetryOncePerCommand {
					retries = 1
				}
			}
			oc.Batches.ClearBatch()
			continue
		}
		break
	}
	return nil
}

// roundTrip writes a wiremessage to the connection, reads a wiremessage, and then decodes the
// response into a result or an error. The wm parameter is reused when reading the wiremessage.
func (oc OperationContext) roundTrip(ctx context.Context, conn Connection, wm []byte) (bsoncore.Document, error) {
	err := conn.WriteWireMessage(ctx, wm)
	if err != nil {
		return nil, Error{Message: err.Error(), Labels: []string{TransientTransactionError, NetworkError}}
	}

	res, err := conn.ReadWireMessage(ctx, wm[:0])
	if err != nil {
		err = Error{Message: err.Error(), Labels: []string{TransientTransactionError, NetworkError}}
	}
	res, err = oc.decompressWireMessage(res)
	if err != nil {
		return nil, err
	}
	return decodeResult(res)
}

// decompressWireMessage handles decompressing a wiremessage. If the wiremessage
// is not compressed, this method will return the wiremessage.
func (oc OperationContext) decompressWireMessage(wm []byte) ([]byte, error) {
	// read the header and ensure this is a compressed wire message
	length, reqid, respto, opcode, rem, ok := wiremessagex.ReadHeader(wm)
	if !ok || len(wm) < int(length) {
		return nil, errors.New("malformed wire message: insufficient bytes")
	}
	if opcode != wiremessage.OpCompressed {
		return wm, nil
	}
	// get the original opcode and uncompressed size
	opcode, rem, ok = wiremessagex.ReadCompressedOriginalOpCode(rem)
	if !ok {
		return nil, errors.New("malformed OP_COMPRESSED: missing original opcode")
	}
	uncompressedSize, rem, ok := wiremessagex.ReadCompressedUncompressedSize(rem)
	if !ok {
		return nil, errors.New("malformed OP_COMPRESSED: missing uncompressed size")
	}
	// get the compressor ID and decompress the message
	compressorID, rem, ok := wiremessagex.ReadCompressedCompressorID(rem)
	if !ok {
		return nil, errors.New("malformed OP_COMPRESSED: missing compressor ID")
	}
	compressedSize := length - 25 // header (16) + original opcode (4) + uncompressed size (4) + compressor ID (1)
	// return the original wiremessage
	msg, rem, ok := wiremessagex.ReadCompressedCompressedMessage(rem, compressedSize)
	if !ok {
		return nil, errors.New("malformed OP_COMPRESSED: insufficient bytes for compressed wiremessage")
	}

	uncompressed := make([]byte, 0, uncompressedSize+16)
	uncompressed = wiremessagex.AppendHeader(uncompressed, uncompressedSize+16, reqid, respto, opcode)
	switch compressorID {
	case wiremessage.CompressorSnappy:
		var err error
		uncompressed, err = snappy.Decode(uncompressed, msg)
		if err != nil {
			return nil, err
		}
	case wiremessage.CompressorZLib:
		decompressor, err := zlib.NewReader(bytes.NewReader(msg))
		if err != nil {
			return nil, err
		}
		uncompressed = uncompressed[:uncompressedSize+16]
		_, err = io.ReadFull(decompressor, uncompressed[16:])
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown compressorID %d", compressorID)
	}
	return uncompressed, nil
}

func (oc OperationContext) createWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	if desc.WireVersion == nil || desc.WireVersion.Max < wiremessage.OpmsgWireVersion {
		return oc.createQueryWireMessage(dst, desc)
	}
	return oc.createMsgWireMessage(dst, desc)
}

func (oc OperationContext) createQueryWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	flags := slaveOK(desc, nil)
	var wmindex int32
	wmindex, dst = wiremessagex.AppendHeaderStart(dst, wiremessage.NextRequestID(), 0, wiremessage.OpQuery)
	dst = wiremessagex.AppendQueryFlags(dst, flags)
	// FullCollectionName
	dst = append(dst, oc.Database...)
	dst = append(dst, dollarCmd[:]...)
	dst = append(dst, 0x00)
	dst = wiremessagex.AppendQueryNumberToSkip(dst, 0)
	dst = wiremessagex.AppendQueryNumberToReturn(dst, -1)

	wrapper := int32(-1)
	rp := createReadPref(oc.ReadPreference, desc.Server.Kind, desc.Kind, true)
	if len(rp) > 0 {
		wrapper, dst = bsoncore.AppendDocumentStart(dst)
		dst = bsoncore.AppendHeader(dst, bsontype.EmbeddedDocument, "$query")
	}
	idx, dst := bsoncore.AppendDocumentStart(dst)
	dst, err := oc.CommandFn(dst, desc)
	if err != nil {
		return dst, err
	}

	if oc.Batches != nil && len(oc.Batches.Current) > 0 {
		aidx, dst := bsoncore.AppendArrayElementStart(dst, oc.Batches.Identifier)
		for i, doc := range oc.Batches.Current {
			dst = bsoncore.AppendDocumentElement(dst, strconv.Itoa(i), doc)
		}
		dst, _ = bsoncore.AppendArrayEnd(dst, aidx)
	}

	dst, err = addReadConcern(dst, oc.ReadConcern, oc.Client, desc)
	if err != nil {
		return dst, err
	}

	dst, err = addWriteConcern(dst, oc.WriteConcern)
	if err != nil {
		return dst, err
	}

	dst, err = addSession(dst, oc.Client, desc)
	if err != nil {
		return dst, err
	}

	// TODO(GODRIVER-617): This should likely be part of addSession, but we need to ensure that we
	// either turn off RetryWrite when we are doing a retryable read or that we pass in RetryType to
	// addSession. We should also only be adding this if the connection supports sessions, but I
	// think that's a given if we've set RetryWrite to true.
	if oc.RetryType == RetryWrite && oc.Client != nil && oc.Client.RetryWrite {
		dst = bsoncore.AppendInt64Element(dst, "txnNumber", oc.Client.TxnNumber)
	}

	dst = addClusterTime(dst, oc.Client, oc.Clock, desc)

	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)

	if len(rp) > 0 {
		var err error
		dst = bsoncore.AppendDocumentElement(dst, "$readPreference", rp)
		dst, err = bsoncore.AppendDocumentEnd(dst, wrapper)
		if err != nil {
			return dst, err
		}
	}

	return bsoncore.UpdateLength(dst, wmindex, int32(len(dst[wmindex:]))), nil
}

func (oc OperationContext) createMsgWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	// TODO(GODRIVER-617): We need to figure out how to include the writeconcern here so that we can
	// set the moreToCome bit.
	var flags wiremessage.MsgFlag
	var wmindex int32
	wmindex, dst = wiremessagex.AppendHeaderStart(dst, wiremessage.NextRequestID(), 0, wiremessage.OpMsg)
	dst = wiremessagex.AppendMsgFlags(dst, flags)
	// Body
	dst = wiremessagex.AppendMsgSectionType(dst, wiremessage.SingleDocument)

	idx, dst := bsoncore.AppendDocumentStart(dst)

	dst, err := oc.CommandFn(dst, desc)
	if err != nil {
		return dst, err
	}
	dst, err = addReadConcern(dst, oc.ReadConcern, oc.Client, desc)
	if err != nil {
		return dst, err
	}
	dst, err = addWriteConcern(dst, oc.WriteConcern)
	if err != nil {
		return dst, err
	}

	dst, err = addSession(dst, oc.Client, desc)
	if err != nil {
		return dst, err
	}

	// TODO(GODRIVER-617): This should likely be part of addSession, but we need to ensure that we
	// either turn off RetryWrite when we are doing a retryable read or that we pass in RetryType to
	// addSession. We should also only be adding this if the connection supports sessions, but I
	// think that's a given if we've set RetryWrite to true.
	if oc.RetryType == RetryWrite && oc.Client != nil && oc.Client.RetryWrite {
		dst = bsoncore.AppendInt64Element(dst, "txnNumber", oc.Client.TxnNumber)
	}

	dst = addClusterTime(dst, oc.Client, oc.Clock, desc)

	dst = bsoncore.AppendStringElement(dst, "$db", oc.Database)
	rp := createReadPref(oc.ReadPreference, desc.Server.Kind, desc.Kind, false)
	if len(rp) > 0 {
		dst = bsoncore.AppendDocumentElement(dst, "$readPreference", rp)
	}

	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)

	if oc.Batches != nil && len(oc.Batches.Current) > 0 {
		dst = wiremessagex.AppendMsgSectionType(dst, wiremessage.DocumentSequence)
		idx, dst = bsoncore.ReserveLength(dst)

		dst = append(dst, oc.Batches.Identifier...)
		dst = append(dst, 0x00)

		for _, doc := range oc.Batches.Current {
			dst = append(dst, doc...)
		}

		dst = bsoncore.UpdateLength(dst, idx, int32(len(dst[idx:])))
	}

	return bsoncore.UpdateLength(dst, wmindex, int32(len(dst[wmindex:]))), nil
}

// Retryable writes are supported if the server supports sessions, the operation is not
// within a transaction, and the write is acknowledged
func retrySupported(
	tdesc description.Topology,
	desc description.Server,
	sess *session.Client,
	wc *writeconcern.WriteConcern,
) bool {
	return (tdesc.SessionTimeoutMinutes != 0 && tdesc.Kind != description.Single) &&
		description.SessionsSupported(desc.WireVersion) &&
		sess != nil &&
		!(sess.TransactionInProgress() || sess.TransactionStarting()) &&
		writeconcern.AckWrite(wc)
}

// createReadPrefSelector will either return the first non-nil selector or create a read preference
// selector with the provided read preference.
func createReadPrefSelector(rp *readpref.ReadPref, selectors ...description.ServerSelector) description.ServerSelector {
	for _, selector := range selectors {
		if selector != nil {
			return selector
		}
	}
	if rp == nil {
		rp = readpref.Primary()
	}
	return description.CompositeSelector([]description.ServerSelector{
		description.ReadPrefSelector(rp),
		description.LatencySelector(15 * time.Millisecond),
	})
}

func addReadConcern(dst []byte, rc *readconcern.ReadConcern, client *session.Client, desc description.SelectedServer) ([]byte, error) {
	// Starting transaction's read concern overrides all others
	if client != nil && client.TransactionStarting() && client.CurrentRc != nil {
		rc = client.CurrentRc
	}

	// start transaction must append afterclustertime IF causally consistent and operation time exists
	if rc == nil && client != nil && client.TransactionStarting() && client.Consistent && client.OperationTime != nil {
		rc = readconcern.New()
	}

	if rc == nil {
		return dst, nil
	}

	_, data, err := rc.MarshalBSONValue() // always returns a document
	if err != nil {
		return dst, err
	}

	if description.SessionsSupported(desc.WireVersion) && client != nil && client.Consistent && client.OperationTime != nil {
		data = data[:len(data)-1] // remove the null byte
		data = bsoncore.AppendTimestampElement(data, "afterClusterTime", client.OperationTime.T, client.OperationTime.I)
		data, _ = bsoncore.AppendDocumentEnd(data, 0)
	}

	return bsoncore.AppendDocumentElement(dst, "readConcern", data), nil
}

func addWriteConcern(dst []byte, wc *writeconcern.WriteConcern) ([]byte, error) {
	if wc == nil {
		return dst, nil
	}

	t, data, err := wc.MarshalBSONValue()
	if err == writeconcern.ErrEmptyWriteConcern {
		return dst, nil
	}
	if err != nil {
		return dst, err
	}

	return append(bsoncore.AppendHeader(dst, t, "writeConcern"), data...), nil
}

func addSession(dst []byte, client *session.Client, desc description.SelectedServer) ([]byte, error) {
	if client == nil || !description.SessionsSupported(desc.WireVersion) || desc.SessionTimeoutMinutes == 0 {
		return dst, nil
	}
	if client.Terminated {
		return dst, session.ErrSessionEnded
	}
	lsid, _ := client.SessionID.MarshalBSON()
	dst = bsoncore.AppendDocumentElement(dst, "lsid", lsid)

	if client.TransactionRunning() || client.RetryingCommit {
		dst = bsoncore.AppendInt64Element(dst, "txnNumber", client.TxnNumber)
		if client.TransactionStarting() {
			dst = bsoncore.AppendBooleanElement(dst, "startTransaction", true)
		}
		dst = bsoncore.AppendBooleanElement(dst, "autocommit", false)
	}

	client.ApplyCommand(desc.Server)

	return dst, nil
}

func addClusterTime(dst []byte, client *session.Client, clock *session.ClusterClock, desc description.SelectedServer) []byte {
	if (clock == nil && client == nil) || !description.SessionsSupported(desc.WireVersion) {
		return dst
	}
	clusterTime := clock.GetClusterTime()
	if client != nil {
		clusterTime = session.MaxClusterTime(clusterTime, client.ClusterTime)
	}
	if clusterTime == nil {
		return dst
	}
	val, err := clusterTime.LookupErr("$clusterTime")
	if err != nil {
		return dst
	}
	return append(bsoncore.AppendHeader(dst, val.Type, "$clusterTime"), val.Value...)
	// return bsoncore.AppendDocumentElement(dst, "$clusterTime", clusterTime)
}

func responseClusterTime(response bsoncore.Document) bsoncore.Document {
	clusterTime, err := response.LookupErr("$clusterTime")
	if err != nil {
		// $clusterTime not included by the server
		return nil
	}
	idx, doc := bsoncore.AppendDocumentStart(nil)
	doc = bsoncore.AppendHeader(doc, clusterTime.Type, "$clusterTime")
	doc = append(doc, clusterTime.Data...)
	doc, _ = bsoncore.AppendDocumentEnd(doc, idx)
	return doc
}

func updateClusterTimes(sess *session.Client, clock *session.ClusterClock, response bsoncore.Document) error {
	clusterTime := responseClusterTime(response)
	if clusterTime == nil {
		return nil
	}

	if sess != nil {
		err := sess.AdvanceClusterTime(bson.Raw(clusterTime))
		if err != nil {
			return err
		}
	}

	if clock != nil {
		clock.AdvanceClusterTime(bson.Raw(clusterTime))
	}

	return nil
}

func updateOperationTime(sess *session.Client, response bsoncore.Document) error {
	if sess == nil {
		return nil
	}

	opTimeElem, err := response.LookupErr("operationTime")
	if err != nil {
		// operationTime not included by the server
		return nil
	}

	t, i := opTimeElem.Timestamp()
	return sess.AdvanceOperationTime(&primitive.Timestamp{
		T: t,
		I: i,
	})
}

func createReadPref(rp *readpref.ReadPref, serverKind description.ServerKind, topologyKind description.TopologyKind, isOpQuery bool) bsoncore.Document {
	idx, doc := bsoncore.AppendDocumentStart(nil)

	if rp == nil {
		if topologyKind == description.Single && serverKind != description.Mongos {
			doc = bsoncore.AppendStringElement(doc, "mode", "primaryPreferred")
			doc, _ = bsoncore.AppendDocumentEnd(doc, idx)
			return doc
		}
		return nil
	}

	switch rp.Mode() {
	case readpref.PrimaryMode:
		if serverKind == description.Mongos {
			return nil
		}
		if topologyKind == description.Single {
			doc = bsoncore.AppendStringElement(doc, "mode", "primaryPreferred")
			doc, _ = bsoncore.AppendDocumentEnd(doc, idx)
			return doc
		}
		doc = bsoncore.AppendStringElement(doc, "mode", "primary")
	case readpref.PrimaryPreferredMode:
		doc = bsoncore.AppendStringElement(doc, "mode", "primaryPreferred")
	case readpref.SecondaryPreferredMode:
		_, ok := rp.MaxStaleness()
		if serverKind == description.Mongos && isOpQuery && !ok && len(rp.TagSets()) == 0 {
			return nil
		}
		doc = bsoncore.AppendStringElement(doc, "mode", "secondaryPreferred")
	case readpref.SecondaryMode:
		doc = bsoncore.AppendStringElement(doc, "mode", "secondary")
	case readpref.NearestMode:
		doc = bsoncore.AppendStringElement(doc, "mode", "nearest")
	}

	sets := make([]bsoncore.Document, 0, len(rp.TagSets()))
	for _, ts := range rp.TagSets() {
		if len(ts) == 0 {
			continue
		}
		i, set := bsoncore.AppendDocumentStart(nil)
		for _, t := range ts {
			set = bsoncore.AppendStringElement(set, t.Name, t.Value)
		}
		set, _ = bsoncore.AppendDocumentEnd(set, i)
		sets = append(sets, set)
	}
	if len(sets) > 0 {
		var aidx int32
		aidx, doc = bsoncore.AppendArrayElementStart(doc, "tags")
		for i, set := range sets {
			doc = bsoncore.AppendDocumentElement(doc, strconv.Itoa(i), set)
		}
		doc, _ = bsoncore.AppendArrayEnd(doc, aidx)
	}

	if d, ok := rp.MaxStaleness(); ok {
		doc = bsoncore.AppendInt32Element(doc, "maxStalenessSeconds", int32(d.Seconds()))
	}

	return doc
}

func slaveOK(desc description.SelectedServer, rp []byte) wiremessage.QueryFlag {
	if desc.Kind == description.Single && desc.Server.Kind != description.Mongos {
		return wiremessage.SlaveOK
	}

	if mode, ok := bsoncore.Document(rp).Lookup("mode").StringValueOK(); ok && mode != "primary" {
		return wiremessage.SlaveOK
	}

	return 0
}

func (OperationContext) canCompress(cmd string) bool {
	if cmd == "isMaster" || cmd == "saslStart" || cmd == "saslContinue" || cmd == "getnonce" || cmd == "authenticate" ||
		cmd == "createUser" || cmd == "updateUser" || cmd == "copydbSaslStart" || cmd == "copydbgetnonce" || cmd == "copydb" {
		return false
	}
	return true
}

func decodeResult(wm []byte) (bsoncore.Document, error) {
	wmLength := len(wm)
	length, _, _, opcode, wm, ok := wiremessagex.ReadHeader(wm)
	if !ok || int(length) > wmLength {
		return nil, errors.New("malformed wire message: insufficient bytes")
	}

	wm = wm[:wmLength-16] // constrain to just this wiremessage, incase there are multiple in the slice

	switch opcode {
	case wiremessage.OpReply:
		var flags wiremessage.ReplyFlag
		flags, wm, ok = wiremessagex.ReadReplyFlags(wm)
		if !ok {
			return nil, errors.New("malformed OP_REPLY: missing flags")
		}
		_, wm, ok = wiremessagex.ReadReplyCursorID(wm)
		if !ok {
			return nil, errors.New("malformed OP_REPLY: missing cursorID")
		}
		_, wm, ok = wiremessagex.ReadReplyStartingFrom(wm)
		if !ok {
			return nil, errors.New("malformed OP_REPLY: missing startingFrom")
		}
		var numReturned int32
		numReturned, wm, ok = wiremessagex.ReadReplyNumberReturned(wm)
		if !ok {
			return nil, errors.New("malformed OP_REPLY: missing numberReturned")
		}
		if numReturned == 0 {
			return nil, ErrNoDocCommandResponse
		}
		if numReturned > 1 {
			return nil, ErrMultiDocCommandResponse
		}
		var rdr bsoncore.Document
		rdr, rem, ok := wiremessagex.ReadReplyDocument(wm)
		if !ok || len(rem) > 0 {
			return nil, NewCommandResponseError("malformed OP_REPLY: NumberReturned does not match number of documents returned", nil)
		}
		err := rdr.Validate()
		if err != nil {
			return nil, NewCommandResponseError("malformed OP_REPLY: invalid document", err)
		}
		if flags&wiremessage.QueryFailure == wiremessage.QueryFailure {
			return nil, QueryFailureError{
				Message:  "command failure",
				Response: rdr,
			}
		}

		return rdr, extractError(rdr)
	case wiremessage.OpMsg:
		_, wm, ok = wiremessagex.ReadMsgFlags(wm)
		if !ok {
			return nil, errors.New("malformed wire message: missing OP_MSG flags")
		}

		var res bsoncore.Document
		for len(wm) > 0 {
			var stype wiremessage.SectionType
			stype, wm, ok = wiremessagex.ReadMsgSectionType(wm)
			if !ok {
				return nil, errors.New("malformed wire message: insuffienct bytes to read section type")
			}

			switch stype {
			case wiremessage.SingleDocument:
				res, wm, ok = wiremessagex.ReadMsgSectionSingleDocument(wm)
				if !ok {
					return nil, errors.New("malformed wire message: insufficient bytes to read single document")
				}
			case wiremessage.DocumentSequence:
				// TODO(GODRIVER-617): Implement document sequence returns.
				_, _, wm, ok = wiremessagex.ReadMsgSectionDocumentSequence(wm)
				if !ok {
					return nil, errors.New("malformed wire message: insufficient bytes to read document sequence")
				}
			default:
				return nil, fmt.Errorf("malformed wire message: uknown section type %v", stype)
			}
		}

		err := res.Validate()
		if err != nil {
			return nil, NewCommandResponseError("malformed OP_MSG: invalid document", err)
		}

		return res, extractError(res)
	default:
		return nil, fmt.Errorf("cannot decode result from %s", opcode)
	}
}
