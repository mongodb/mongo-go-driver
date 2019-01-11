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

//go:generate drivergen -use-pointers InsertOperation insert.generated.go

var msgInsert = [...]byte{'d', 'o', 'c', 'u', 'm', 'e', 'n', 't', 's', 0x00}

type InsertOperation struct {
	insert struct{} `drivergen:",commandName"`

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

	batch []bsoncore.Document `drivergen:"documents,msgDocSeq"` // The current batch for this operation.

	result result.Insert `drivergen:"-"`
}

func Insert(documents ...bsoncore.Document) *InsertOperation {
	return &InsertOperation{documents: documents}
}

// Result returns the result from executing this operation. This should only be called after Execute
// and any retries.
func (io *InsertOperation) Result() result.Insert {
	if io == nil {
		return result.Insert{}
	}
	return io.result
}

// Select retrieves a server to be used when executing an operation.
func (io *InsertOperation) Select(ctx context.Context) (Server, error) {
	if io.d == nil {
		return nil, errors.New("FindOperation must have a Deployment set before Select can be called.")
	}
	return io.d.SelectServer(ctx, createReadPrefSelector(readpref.Primary(), io.serverSelector))
}

// SelectAndExecute selects a server and runs this operation against it.
func (io *InsertOperation) SelectAndExecute(ctx context.Context) error {
	srvr, err := io.Select(ctx)
	if err != nil {
		return err
	}

	return io.Execute(ctx, srvr)
}

// SelectAndExecute selects a server and retries this operation against it. This is a convenience
// method and should only be called after a retryable error is returned from SelectAndExecute or
// Execute.
func (io *InsertOperation) SelectAndRetryExecute(ctx context.Context, original error) error {
	if io.d == nil {
		return errors.New("InsertOperation must have a Deployment set before RetryExecute can be called.")
	}
	srvr, err := io.Select(ctx)

	// Return original error if server selection fails.
	if err != nil {
		return original
	}

	return io.RetryExecute(ctx, srvr, original)
}

// RetryExecute retries this operation against the provided server. This method should only be
// called after a retryable error is returned from either SelectAndExecute or Execute.
func (io *InsertOperation) RetryExecute(ctx context.Context, srvr Server, original error) error {
	conn, err := srvr.Connection(ctx)
	// Return original error if connection retrieval fails or new server does not support retryable writes.
	if err != nil || conn == nil || !retrySupported(io.d.Description(), conn.Description(), io.client, io.writeConcern) {
		return original
	}
	defer conn.Close()

	return io.execute(ctx, conn)
}

// Execute runs this operation against the provided server. If the error returned is retryable,
// either SelectAndRetryExecute or Select followed by RetryExecute can be called to retry this
// operation.
func (io *InsertOperation) Execute(ctx context.Context, srvr Server) error {
	if io.d == nil {
		return errors.New("InsertOperation must have a Deployment set before Execute can be called.")
	}
	conn, err := srvr.Connection(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	desc := conn.Description()

	if io.client != nil && !writeconcern.AckWrite(io.writeConcern) {
		return errors.New("session provided for an unacknowledged write")
	}

	retryable := (io.retry != nil && *io.retry == true) && retrySupported(io.d.Description(), desc, io.client, io.writeConcern)
	if retryable {
		// io.client must not be nil or retrySupported would have returned false
		io.client.RetryWrite = true
		io.client.IncrementTxnNumber()
	}
	return io.execute(ctx, conn)
}

func (io *InsertOperation) execute(ctx context.Context, conn Connection) error {
	var err error
	desc := description.SelectedServer{Server: conn.Description(), Kind: io.d.Description().Kind}
	var res bsoncore.Document
	// if we don't have a batch then we aren't retrying and this is the first execution of this
	// operation, so we need to increment the transaction number.
	retryable := (io.retry != nil && *io.retry == true) && retrySupported(io.d.Description(), desc.Server, io.client, io.writeConcern)
	for {
		if len(io.batch) == 0 {
			// encode batch
			io.batch, io.documents, err = splitBatches(io.documents, int(desc.MaxBatchCount), int(desc.MaxDocumentSize))
			if err != nil {
				return err
			}
		}
		// convert to wire message
		wm, err := io.createWireMessage(nil, desc)
		if err != nil {
			return err
		}
		// roundtrip
		res, err = roundTripDecode(ctx, conn, wm)
		if err != nil {
			return err
		}
		var r result.Insert
		err = bson.Unmarshal(res, &r)
		if err != nil {
			return err
		}
		io.result.WriteErrors = append(io.result.WriteErrors, r.WriteErrors...)
		io.result.WriteConcernError = r.WriteConcernError
		io.result.N += r.N

		if r.WriteConcernError != nil && retryable {
			// Construct a write concern error and return
			return Error{
				Code:    int32(r.WriteConcernError.Code),
				Message: r.WriteConcernError.ErrMsg,
			}
		}

		if (io.ordered != nil && *io.ordered == true) && len(r.WriteErrors) > 0 {
			return nil
		}

		if len(io.documents) > 0 {
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
	return nil
}

func (io *InsertOperation) createWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	if io.ns.Collection == "" || io.ns.DB == "" {
		return nil, errors.New("Collection and DB must be of non-zero length")
	}
	if desc.WireVersion == nil || desc.WireVersion.Max < wiremessage.OpmsgWireVersion {
		rp := createReadPref(readpref.Primary(), desc.Server.Kind, desc.Kind, true)
		return createQueryWireMessage(dst, desc, io.ns.DB, rp, io.queryAppendCommand)
	}
	return io.createMsgWireMessage(dst, desc)
}

func (io *InsertOperation) createMsgWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
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

func (io *InsertOperation) queryAppendCommand(dst []byte, desc description.SelectedServer) ([]byte, error) {
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

	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)
	return dst, nil
}
