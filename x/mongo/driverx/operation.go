package driverx

import (
	"context"
	"errors"
	"strconv"

	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
	"go.mongodb.org/mongo-driver/x/network/wiremessagex"
)

// OperationContext is used to execute an operation. It contains all of the common code required to
// select a server, transform an operation into a command, write the command to a connection from
// the selected server, read a response from that connection, process the response, and potentially
// retry. The only fields that are required are Deployment, Database, and CommandFn. All other
// fields are optional.
type OperationContext struct {
	CommandFn  func(dst []byte, desc description.SelectedServer) ([]byte, error)
	Deployment Deployment
	Database   string

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

func (oc OperationContext) SelectServer(ctx context.Context) (Server, error) {
	if err := oc.Validate(); err != nil {
		return nil, err
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

func (oc OperationContext) Validate() error {
	if oc.CommandFn == nil {
		return errors.New("the CommandFn field must be set on OperationContext")
	}
	if oc.Deployment == nil {
		return errors.New("the Deployment field must be set on OperationContext")
	}
	if oc.Database == "" {
		return errors.New("the Database field must be non-empty on OperationContext")
	}
	return nil
}

func (oc OperationContext) Execute(ctx context.Context) error {
	err := oc.Validate()
	if err != nil {
		return err
	}

	srvr, err := oc.SelectServer(ctx)
	if err != nil {
		return err
	}

	conn, err := srvr.Connection(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	desc := description.SelectedServer{Server: conn.Description(), Kind: oc.Deployment.Description().Kind}

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
				retries -= 1
				original, err = err, nil
				conn.Close() // Avoid leaking the connection.
				srvr, err = oc.SelectServer(ctx)
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
				retries -= 1
				original, err = err, nil
				conn.Close() // Avoid leaking the connection.
				srvr, err = oc.SelectServer(ctx)
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
func (OperationContext) roundTrip(ctx context.Context, conn Connection, wm []byte) (bsoncore.Document, error) {
	err := conn.WriteWireMessage(ctx, wm)
	if err != nil {
		return nil, Error{Message: err.Error(), Labels: []string{TransientTransactionError, NetworkError}}
	}

	res, err := conn.ReadWireMessage(ctx, wm[:0])
	if err != nil {
		err = Error{Message: err.Error(), Labels: []string{TransientTransactionError, NetworkError}}
	}
	return decodeResult(res)
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
