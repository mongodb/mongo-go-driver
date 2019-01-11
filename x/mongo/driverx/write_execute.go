package driverx

import (
	"context"
	"strconv"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
	"go.mongodb.org/mongo-driver/x/network/wiremessagex"
)

type writeOperation interface {
	// processResponse handles processing the response document from a write command. It should
	// store any internal state that will be returned later as the result of this operation.
	processResponse(response bsoncore.Document) error

	// retryable determines if this operation is currently retryable using a connection from the
	// server described.
	retryable(description.Server) bool

	// selectSever chooses and retruns a server. This is used for both initial server selection and
	// subsequent server selection that occurs during retries.
	selectServer(context.Context) (Server, error)

	// command appends the wiremessage agnostic components of the command to dst. This includes the
	// command name and colleciton, and any options that are not $db nor the documents that would go
	// in an OP_MSG type 1 payload.
	//
	// The dst slice is returned with the command body.
	command(dst []byte, desc description.SelectedServer) (res []byte, err error)
}

// writeOperationContext holds the common elements of executing a write operation. The
// writeOperation is the write operation specific functionality. This type is designed for use with
// insert, delete, and update.
type writeOperationContext struct {
	writeOperation

	tkind      description.TopologyKind
	db         string // database
	identifier string // identifier for document batch or type 1 payload
	documents  []bsoncore.Document

	client  *session.Client
	mode    *RetryMode
	ordered *bool
}

func (wo writeOperationContext) execute(ctx context.Context) error {
	tkind := wo.tkind
	client := wo.client
	rm := wo.mode
	ordered := wo.ordered
	documents := wo.documents

	srvr, err := wo.selectServer(ctx)
	if err != nil {
		return err
	}

	conn, err := srvr.Connection(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	desc := description.SelectedServer{Server: conn.Description(), Kind: tkind}

	retryable := wo.retryable(desc.Server)
	if retryable && client != nil {
		client.RetryWrite = true
		client.IncrementTxnNumber()
	}

	var res bsoncore.Document
	var original error
	var batch []bsoncore.Document
	var operationErr WriteCommandError
	var retries int
	if retryable && rm != nil {
		switch *rm {
		case RetryOnce, RetryOncePerCommand:
			retries = 1
		case RetryContext:
			retries = -1
		}
	}
	for {
		if len(batch) == 0 {
			// encode batch
			batch, documents, err = splitBatches(documents, int(desc.MaxBatchCount), int(desc.MaxDocumentSize))
			if err != nil {
				return err
			}
		}
		// convert to wire message
		wm, err := wo.createWireMessage(nil, batch, desc)
		if err != nil {
			return err
		}
		// roundtrip
		res, err = roundTripDecode(ctx, conn, wm)
		perr := wo.processResponse(res)
		switch tt := err.(type) {
		case WriteCommandError:
			if retryable && tt.Retryable() && retries != 0 {
				retries -= 1
				original, err = err, nil
				conn.Close() // Avoid leaking the connection.
				srvr, err = wo.selectServer(ctx)
				if err != nil {
					return original
				}
				conn, err := srvr.Connection(ctx)
				if err != nil || conn == nil || !wo.retryable(conn.Description()) {
					return original
				}
				defer conn.Close() // Avoid leaking the new connection.
				continue
			}
			if ordered != nil && *ordered == true && len(tt.WriteErrors) > 0 {
				return tt
			}
			operationErr.WriteConcernError = tt.WriteConcernError
			operationErr.WriteErrors = append(operationErr.WriteErrors, tt.WriteErrors...)
		case Error:
			if retryable && tt.Retryable() && retries != 0 {
				retries -= 1
				original, err = err, nil
				conn.Close() // Avoid leaking the connection.
				srvr, err = wo.selectServer(ctx)
				if err != nil {
					return original
				}
				conn, err := srvr.Connection(ctx)
				if err != nil || conn == nil || !wo.retryable(conn.Description()) {
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

		if len(documents) > 0 {
			if retryable {
				// io.client must not be nil or retrySupported would have returned false
				client.IncrementTxnNumber()
				if rm != nil && *rm == RetryOncePerCommand {
					retries = 1
				}
			}
			batch = batch[:0]
			continue
		}
		break
	}
	return err
}

func (wo writeOperationContext) createWireMessage(dst []byte, batch []bsoncore.Document, desc description.SelectedServer) ([]byte, error) {
	if desc.WireVersion == nil || desc.WireVersion.Max < wiremessage.OpmsgWireVersion {
		return wo.createQueryWireMessage(dst, batch, desc)
	}
	return wo.createMsgWireMessage(dst, batch, desc)
}

func (wo writeOperationContext) createQueryWireMessage(dst []byte, batch []bsoncore.Document, desc description.SelectedServer) ([]byte, error) {
	flags := slaveOKbytes(desc, nil)
	var wmindex int32
	wmindex, dst = wiremessagex.AppendHeaderStart(dst, wiremessage.NextRequestID(), 0, wiremessage.OpQuery)
	dst = wiremessagex.AppendQueryFlags(dst, flags)
	// FullCollectionName
	dst = append(dst, wo.db...)
	dst = append(dst, dollarCmd[:]...)
	dst = append(dst, 0x00)
	dst = wiremessagex.AppendQueryNumberToSkip(dst, 0)
	dst = wiremessagex.AppendQueryNumberToReturn(dst, -1)
	// TODO(GODRIVER-617): It isn't clear to me whether we need to ever send a $readPreference with
	// a write command. I don't think so, but need to confirm.
	idx, dst := bsoncore.AppendDocumentStart(dst)
	dst, err := wo.command(dst, desc)
	if err != nil {
		return dst, err
	}

	aidx, dst := bsoncore.AppendArrayElementStart(dst, wo.identifier)
	for i, doc := range batch {
		dst = bsoncore.AppendDocumentElement(dst, strconv.Itoa(i), doc)
	}
	dst, _ = bsoncore.AppendArrayEnd(dst, aidx)

	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)

	return bsoncore.UpdateLength(dst, wmindex, int32(len(dst[wmindex:]))), nil
}

func (wo writeOperationContext) createMsgWireMessage(dst []byte, batch []bsoncore.Document, desc description.SelectedServer) ([]byte, error) {
	// TODO(GODRIVER-617): We need to figure out how to include the writeconcern here so that we can
	// set the moreToCome bit.
	var flags wiremessage.MsgFlag
	var wmindex int32
	wmindex, dst = wiremessagex.AppendHeaderStart(dst, wiremessage.NextRequestID(), 0, wiremessage.OpMsg)
	dst = wiremessagex.AppendMsgFlags(dst, flags)
	// Body
	dst = wiremessagex.AppendMsgSectionType(dst, wiremessage.SingleDocument)

	idx, dst := bsoncore.AppendDocumentStart(dst)

	dst, err := wo.command(dst, desc)
	if err != nil {
		return dst, err
	}

	dst = bsoncore.AppendStringElement(dst, "$db", wo.db)

	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)

	dst = wiremessagex.AppendMsgSectionType(dst, wiremessage.DocumentSequence)
	idx, dst = bsoncore.ReserveLength(dst)

	dst = append(dst, wo.identifier...)
	dst = append(dst, 0x00)

	for _, doc := range batch {
		dst = append(dst, doc...)
	}

	dst = bsoncore.UpdateLength(dst, idx, int32(len(dst[idx:])))

	return bsoncore.UpdateLength(dst, wmindex, int32(len(dst[wmindex:]))), nil
}
