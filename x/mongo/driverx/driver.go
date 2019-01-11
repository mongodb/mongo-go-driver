package driverx

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
	"go.mongodb.org/mongo-driver/x/network/wiremessagex"
)

var dollarCmd = [...]byte{'.', '$', 'c', 'm', 'd'}

// this is the amount of reserved buffer space in a message that the
// driver reserves for command overhead.
const reservedCommandBufferBytes = 16 * 10 * 10 * 10

var (
	// ErrNoDocCommandResponse occurs when the server indicated a response existed, but none was found.
	ErrNoDocCommandResponse = errors.New("command returned no documents")
	// ErrMultiDocCommandResponse occurs when the server sent multiple documents in response to a command.
	ErrMultiDocCommandResponse = errors.New("command returned multiple documents")
	// ErrDocumentTooLarge occurs when a document that is larger than the maximum size accepted by a
	// server is passed to an insert command.
	ErrDocumentTooLarge = errors.New("an inserted document is too large")
	// TransientTransactionError is an error label for transient errors with transactions.
	TransientTransactionError = "TransientTransactionError"
	// NetworkError is an error label for network errors.
	NetworkError = "NetworkError"
)

// Deployment is implemented by types that can select a server from a deployment.
type Deployment interface {
	SelectServer(context.Context, description.ServerSelector) (Server, error)
	Description() description.Topology
}

// Server represents a MongoDB server. Implementations should pool connections and handle the
// retrieving and returning of connections.
type Server interface {
	Connection(context.Context) (Connection, error)
}

// Connection represents a connection to a MongoDB server.
type Connection interface {
	WriteWireMessage(context.Context, []byte) error
	ReadWireMessage(ctx context.Context, dst []byte) ([]byte, error)
	Description() description.Server
	Close() error
	ID() string
}

// Namespace encapsulates a database and collection name, which together
// uniquely identifies a collection within a MongoDB cluster.
type Namespace struct {
	DB         string
	Collection string
}

// Collation allows users to specify language-specific rules for string comparison, such as
// rules for lettercase and accent marks.
type Collation struct {
	Locale          string `bson:",omitempty"` // The locale
	CaseLevel       bool   `bson:",omitempty"` // The case level
	CaseFirst       string `bson:",omitempty"` // The case ordering
	Strength        int    `bson:",omitempty"` // The number of comparision levels to use
	NumericOrdering bool   `bson:",omitempty"` // Whether to order numbers based on numerical order and not collation order
	Alternate       string `bson:",omitempty"` // Whether spaces and punctuation are considered base characters
	MaxVariable     string `bson:",omitempty"` // Which characters are affected by alternate: "shifted"
	Backwards       bool   `bson:",omitempty"` // Causes secondary differences to be considered in reverse order, as it is done in the French language
}

// Executor implementations handle running an operation using the provided Server.
type Executor interface {
	Execute(context.Context) error
}

// TODO(GODRUVER-617): We can likely use 1 type for both the RetryType and the RetryMode by using
// 2 bits for the mode and 1 bit for the type. Although in the practical sense, we might not want to
// do that since the type of retryability is tied to the operation itself and isn't going change,
// e.g. and insert operation will always be a write, however some operations are both reads and
// writes, for instance aggregate is a read but with a $out parameter it's a write.

// RetryType specifies whether a retry is a read, write, or disabled.
type RetryType uint

// THese are the availables types of retry.
const (
	_ RetryType = iota
	RetryWrite
	RetryRead
)

// RetryMode specifies the way that retries are handled for retryable operations.
type RetryMode uint

// These are the modes available for retrying.
const (
	// RetryNone disables retrying.
	RetryNone RetryMode = iota
	// RetryOnce will enable retrying the entire operation once.
	RetryOnce
	// RetryOncePerCommand will enable retrying each command associated with an operation. For
	// example, if an insert is batch split into 4 commands then each of those commands is eligible
	// for one retry.
	RetryOncePerCommand
	// RetryContext will enable retrying until the context.Context's deadline is exceeded or it is
	// cancelled.
	RetryContext
)

// Enabled returns if this RetryMode enables retrying.
func (rm RetryMode) Enabled() bool {
	return rm == RetryOnce || rm == RetryOncePerCommand || rm == RetryContext
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
