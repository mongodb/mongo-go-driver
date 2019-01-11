package driverx

import (
	"context"
	"errors"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/result"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
	"go.mongodb.org/mongo-driver/x/network/wiremessagex"
)

//go:generate drivergen InsertOperation insert.generated.go

var msgInsert = [...]byte{'d', 'o', 'c', 'u', 'm', 'e', 'n', 't', 's', 0x00}

type InsertOperation struct {
	_ struct{} `drivergen:"insert,commandName"`

	// Documents adds documents to this operation that will be inserted when this operation is
	// executed.
	documents []bsoncore.Document `drivergen:",variadic"`

	// Ordered sets ordered. If true, when a write fails, the operation will return the error, when
	// false write failures do not stop execution of the operation.
	ordered *bool

	// WriteConcern sets the write concern for this operation.
	writeConcern *writeconcern.WriteConcern `drivergen:",pointerExempt"`

	// BypassDocumentValidation allows the operation to opt-out of document level validation. Valid
	// for server versions >= 3.2. For servers < 3.2, this setting is ignored.
	bypassDocumentValidation *bool

	// Retry enables retryable writes for this operation. Retries are not handled automatically,
	// instead a boolean is returned from Execute and SelectAndExecute that indicates if the
	// operation can be retried. Retrying is handled by calling RetryExecute.
	retry *bool

	// Namespace sets the database and collection to run this operation against.
	ns Namespace `drivergen:"Namespace"`

	// Deployment sets the deployment to use for this operation.
	d Deployment `drivergen:"Deployment"`

	// ServerSelector sets the selector used to retrieve a server.
	serverSelector description.ServerSelector

	// Session sets the session for this operation.
	client *session.Client `drivergen:"Session,pointerExempt"`

	// ClusterClock sets the cluster clock for this operation.
	clock *session.ClusterClock `drivergen:"ClusterClock,pointerExempt"`

	batch []bsoncore.Document `drivergen:"-"` // The current batch for this operation.
}

func Insert(documents ...bsoncore.Document) InsertOperation {
	return InsertOperation{documents: documents}
}

// Select retrieves a server to be used when executing an operation.
func (io InsertOperation) Select(ctx context.Context) (Server, error) {
	if io.d == nil {
		return nil, errors.New("FindOperation must have a Deployment set before Select can be called.")
	}
	return io.d.SelectServer(ctx, createReadPrefSelector(io.serverSelector, readpref.Primary()))
}

// SelectAndExecute selects a server and runs this operation against it. This method returns the
// result, a boolean indicating if this operation is retryable, the operation to use when retrying,
// and an error.
func (io InsertOperation) SelectAndExecute(ctx context.Context) (result.Insert, bool, InsertOperation, error) {
	srvr, err := io.Select(ctx)
	if err != nil {
		return result.Insert{}, false, io, err
	}

	return io.Execute(ctx, srvr)
}

func (io InsertOperation) isRetryable(res bsoncore.Document, err error) bool {
	cerr, ok := err.(Error)
	return (io.retry != nil && *io.retry == true) &&
		((ok && cerr.Retryable()) || (writeConcernErrorRetryable(res.Lookup("writeConcernError"))))
}

func (io InsertOperation) RetryExecute(ctx context.Context, original error) (result.Insert, bool, InsertOperation, error) {
	if io.d == nil {
		return result.Insert{}, false, io, errors.New("InsertOperation must have a Deployment set before RetryExecute can be called.")
	}
	srvr, err := io.Select(ctx)

	// Return original error if server selection fails.
	if err != nil {
		return result.Insert{}, false, InsertOperation{}, original
	}

	conn, err := srvr.Connection(ctx)
	// Return original error if connection retrieval fails or new server does not support retryable writes.
	if err != nil || conn == nil || !retrySupported(io.d.Description(), conn.Description(), io.client, io.writeConcern) {
		return result.Insert{}, false, InsertOperation{}, original
	}
	defer conn.Close()

	return io.execute(ctx, conn)
}

func (io InsertOperation) Execute(ctx context.Context, srvr Server) (result.Insert, bool, InsertOperation, error) {
	if io.d == nil {
		return result.Insert{}, false, io, errors.New("InsertOperation must have a Deployment set before Execute can be called.")
	}
	conn, err := srvr.Connection(ctx)
	if err != nil {
		return result.Insert{}, false, io, err
	}

	desc := conn.Description()

	if io.client != nil {
		io.client.RetryWrite = false // explicitly set to false to prevent encoding transaction number
	}

	// execute unacknowledged writes early since they can't be retryable.
	if !writeconcern.AckWrite(io.writeConcern) {
		go func() {
			defer func() { _ = recover() }()
			defer conn.Close()

			_, _, _, _ = io.execute(ctx, conn)
		}()

		return result.Insert{}, false, io, errors.New("unacknowledged write")
	}
	defer conn.Close()

	retryable := (io.retry != nil && *io.retry == true) && retrySupported(io.d.Description(), desc, io.client, io.writeConcern)
	if retryable {
		// io.client must not be nil or retrySupported would have returned false
		io.client.RetryWrite = true
		io.client.IncrementTxnNumber()
	}
	return io.execute(ctx, conn)
}

func (io InsertOperation) execute(ctx context.Context, conn Connection) (result.Insert, bool, InsertOperation, error) {
	var err error
	desc := description.SelectedServer{Server: conn.Description(), Kind: io.d.Description().Kind}
	var res bsoncore.Document
	var ret result.Insert
	for {
		if len(io.batch) == 0 {
			// encode batch
			io.batch, io.documents, err = splitBatches(io.documents, int(desc.MaxBatchCount), int(desc.MaxDocumentSize))
			if err != nil {
				return ret, false, io, err
			}
		}
		// convert to wire message
		wm, err := io.createWireMessage(nil, desc)
		if err != nil {
			return ret, false, io, err
		}
		// roundtrip
		res, err = roundTripDecode(ctx, conn, wm)
		if err != nil {
			return ret, io.isRetryable(res, err), io, err
		}
		var r result.Insert
		err = bson.Unmarshal(res, &r)
		if err != nil {
			return ret, io.isRetryable(res, err), io, err
		}
		ret.WriteErrors = append(ret.WriteErrors, r.WriteErrors...)
		ret.WriteConcernError = r.WriteConcernError
		ret.N += r.N

		if (io.ordered != nil && *io.ordered == true) && len(r.WriteErrors) > 0 {
			return ret, io.isRetryable(res, err), io, err
		}
		// on failure determine retryability, create new InsertOperation with remaining documents, and
		// if there are more batches, continue
		// return
		if len(io.documents) > 0 {
			retryable := (io.retry != nil && *io.retry == true) && retrySupported(io.d.Description(), desc.Server, io.client, io.writeConcern)
			if retryable {
				// io.client must not be nil or retrySupported would have returned false
				io.client.IncrementTxnNumber()
			}
			io.batch = io.batch[:0]
			continue
		}
		break
	}
	// TODO(GODRIVER-617): We used to call ProcessWriteConcernError here, but instead we should have
	// the Connection implementation check for this. It should be fast since we're using bsoncore
	// (and we could potentially optimize with the bytes package).
	return ret, io.isRetryable(res, err), io, err
}

func (io InsertOperation) createWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	if io.ns.Collection == "" || io.ns.DB == "" {
		return nil, errors.New("Collection and DB must be of non-zero length")
	}
	if desc.WireVersion == nil || desc.WireVersion.Max < wiremessage.OpmsgWireVersion {
		rp := createReadPref(readpref.Primary(), desc.Server.Kind, desc.Kind, true)
		return createQueryWireMessage(dst, desc, io.ns.DB, rp, io.queryAppendCommand)
	}
	return io.createMsgWireMessage(dst, desc)
}

func (io InsertOperation) createMsgWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	// TODO(GODRIVER-617): How do we allow users to supply flags? Perhaps we don't and we add
	// functions to allow users to set them themselves.
	var flags wiremessage.MsgFlag
	var wmindex int32
	wmindex, dst = wiremessagex.AppendHeaderStart(dst, wiremessage.NextRequestID(), 0, wiremessage.OpMsg)
	dst = wiremessagex.AppendMsgFlags(dst, flags)
	// Body
	dst = wiremessagex.AppendMsgSectionType(dst, wiremessage.SingleDocument)

	idx, dst := bsoncore.AppendDocumentStart(dst)

	dst = bsoncore.AppendStringElement(dst, "insert", io.ns.Collection)

	if io.ordered != nil {
		dst = bsoncore.AppendBooleanElement(dst, "ordered", *io.ordered)
	}
	if io.bypassDocumentValidation != nil {
		dst = bsoncore.AppendBooleanElement(dst, "bypassDocumentValidation", *io.bypassDocumentValidation)
	}
	dst, err := addWriteConcern(dst, io.writeConcern)
	if err != nil {
		return dst, err
	}

	dst, err = addSession(dst, io.client, desc)
	if err != nil {
		return dst, err
	}

	if io.client.RetryWrite {
		dst = bsoncore.AppendInt64Element(dst, "txnNumber", io.client.TxnNumber)
	}

	dst = addClusterTime(dst, io.client, io.clock, desc)

	dst = bsoncore.AppendStringElement(dst, "$db", io.ns.DB)

	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)

	dst = wiremessagex.AppendMsgSectionType(dst, wiremessage.DocumentSequence)
	idx, dst = bsoncore.ReserveLength(dst)

	dst = append(dst, msgInsert[:]...)

	for _, doc := range io.batch {
		dst = append(dst, doc...)
	}

	dst = bsoncore.UpdateLength(dst, idx, int32(len(dst[idx:])))

	return bsoncore.UpdateLength(dst, wmindex, int32(len(dst[wmindex:]))), nil
}

func (io InsertOperation) queryAppendCommand(dst []byte, desc description.SelectedServer) ([]byte, error) {
	idx, dst := bsoncore.AppendDocumentStart(dst)
	dst = bsoncore.AppendStringElement(dst, "insert", io.ns.Collection)

	aidx, dst := bsoncore.AppendArrayElementStart(dst, "documents")
	for i, doc := range io.batch {
		dst = bsoncore.AppendDocumentElement(dst, strconv.Itoa(i), doc)
	}
	dst, _ = bsoncore.AppendArrayEnd(dst, aidx)

	if io.ordered != nil {
		dst = bsoncore.AppendBooleanElement(dst, "ordered", *io.ordered)
	}
	if io.bypassDocumentValidation != nil {
		dst = bsoncore.AppendBooleanElement(dst, "bypassDocumentValidation", *io.bypassDocumentValidation)
	}
	dst, err := addWriteConcern(dst, io.writeConcern)
	if err != nil {
		return dst, err
	}

	dst, err = addSession(dst, io.client, desc)
	if err != nil {
		return dst, err
	}

	if io.client.RetryWrite {
		dst = bsoncore.AppendInt64Element(dst, "txnNumber", io.client.TxnNumber)
	}

	dst = addClusterTime(dst, io.client, io.clock, desc)

	dst = bsoncore.AppendStringElement(dst, "$db", io.ns.DB)

	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)
	return dst, nil
}
