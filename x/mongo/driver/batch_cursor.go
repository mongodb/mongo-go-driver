package driver

import (
	"context"
	"errors"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"github.com/mongodb/mongo-go-driver/x/bsonx/bsoncore"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver/session"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver/topology"
	"github.com/mongodb/mongo-go-driver/x/network/command"
	"github.com/mongodb/mongo-go-driver/x/network/wiremessage"
)

// BatchCursor is a batch implementation of a cursor. It returns documents in entire batches instead
// of one at a time. An individual document cursor can be built on top of this batch cursor.
type BatchCursor struct {
	clientSession *session.Client
	clock         *session.ClusterClock
	namespace     command.Namespace
	id            int64
	err           error
	server        *topology.Server
	opts          []bsonx.Elem
	currentBatch  []byte
	firstBatch    bool
	batchNumber   int

	// legacy server (< 3.2) fields
	batchSize   int32
	limit       int32
	numReturned int32 // number of docs returned by server
}

// NewBatchCursor creates a new BatchCursor from the provided parameters.
func NewBatchCursor(result bsoncore.Document, clientSession *session.Client, clock *session.ClusterClock, server *topology.Server, opts ...bsonx.Elem) (*BatchCursor, error) {
	cur, err := result.LookupErr("cursor")
	if err != nil {
		return nil, err
	}
	if cur.Type != bson.TypeEmbeddedDocument {
		return nil, fmt.Errorf("cursor should be an embedded document but it is a BSON %s", cur.Type)
	}

	elems, err := cur.Document().Elements()
	if err != nil {
		return nil, err
	}
	bc := &BatchCursor{
		clientSession: clientSession,
		clock:         clock,
		server:        server,
		opts:          opts,
		firstBatch:    true,
	}

	var ok bool
	for _, elem := range elems {
		switch elem.Key() {
		case "firstBatch":
			arr, ok := elem.Value().ArrayOK()
			if !ok {
				return nil, fmt.Errorf("firstBatch should be an array but it is a BSON %s", elem.Value().Type)
			}
			vals, err := arr.Values()
			if err != nil {
				return nil, err
			}

			for _, val := range vals {
				if val.Type != bsontype.EmbeddedDocument {
					return nil, fmt.Errorf("element of cursor batch is not a document, but at %s", val.Type)
				}
				bc.currentBatch = append(bc.currentBatch, val.Data...)
			}
		case "ns":
			if elem.Value().Type != bson.TypeString {
				return nil, fmt.Errorf("namespace should be a string but it is a BSON %s", elem.Value().Type)
			}
			namespace := command.ParseNamespace(elem.Value().StringValue())
			err = namespace.Validate()
			if err != nil {
				return nil, err
			}
			bc.namespace = namespace
		case "id":
			bc.id, ok = elem.Value().Int64OK()
			if !ok {
				return nil, fmt.Errorf("id should be an int64 but it is a BSON %s", elem.Value().Type)
			}
		}
	}

	// close session if everything fits in first batch
	if bc.id == 0 {
		bc.closeImplicitSession()
	}
	return bc, nil
}

// NewEmptyBatchCursor returns a batch cursor that is empty.
func NewEmptyBatchCursor() *BatchCursor {
	return &BatchCursor{}
}

// NewLegacyBatchCursor creates a new BatchCursor for server versions 3.0 and below from the
// provided parameters.
//
// TODO(GODRIVER-617): The batch parameter here should be []bsoncore.Document. Change it to this
// once we have the new wiremessage package that uses bsoncore instead of bson.
func NewLegacyBatchCursor(ns command.Namespace, cursorID int64, batch []bson.Raw, limit int32, batchSize int32, server *topology.Server) (*BatchCursor, error) {
	bc := &BatchCursor{
		id:          cursorID,
		server:      server,
		namespace:   ns,
		limit:       limit,
		batchSize:   batchSize,
		numReturned: int32(len(batch)),
		firstBatch:  true,
	}

	// take as many documents from the batch as needed
	firstBatchSize := int32(len(batch))
	if limit != 0 && limit < firstBatchSize {
		firstBatchSize = limit
	}
	batch = batch[:firstBatchSize]
	for _, doc := range batch {
		bc.currentBatch = append(bc.currentBatch, doc...)
	}

	return bc, nil
}

// ID returns the cursor ID for this batch cursor.
func (bc *BatchCursor) ID() int64 {
	return bc.id
}

// Next indicates if there is another batch available. Returning false does not necessarily indicate
// that the cursor is closed. This method will return false when an empty batch is returned.
//
// If Next returns true, there is a valid batch of documents available. If Next returns false, there
// is not a valid batch of documents available.
func (bc *BatchCursor) Next(ctx context.Context) bool {
	if ctx == nil {
		ctx = context.Background()
	}

	if bc.firstBatch {
		bc.firstBatch = false
		return true
	}

	if bc.id == 0 || bc.server == nil {
		return false
	}

	if bc.legacy() {
		bc.legacyGetMore(ctx)
	} else {
		bc.getMore(ctx)
	}

	return len(bc.currentBatch) > 0
}

// Batch will append the current batch of documents to dst. RequiredBytes can be called to determine
// the length of the current batch of documents.
//
// If there is no batch available, this method does nothing.
func (bc *BatchCursor) Batch(dst []byte) []byte { return append(dst, bc.currentBatch...) }

// RequiredBytes returns the number of bytes required for the current batch.
func (bc *BatchCursor) RequiredBytes() int { return len(bc.currentBatch) }

// Err returns the latest error encountered.
func (bc *BatchCursor) Err() error { return bc.err }

// Close closes this batch cursor.
func (bc *BatchCursor) Close(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if bc.server == nil {
		return nil
	}

	if bc.legacy() {
		return bc.legacyKillCursor(ctx)
	}

	defer bc.closeImplicitSession()
	conn, err := bc.server.Connection(ctx)
	if err != nil {
		return err
	}

	_, err = (&command.KillCursors{
		Clock: bc.clock,
		NS:    bc.namespace,
		IDs:   []int64{bc.id},
	}).RoundTrip(ctx, bc.server.SelectedDescription(), conn)
	if err != nil {
		_ = conn.Close() // The command response error is more important here
		return err
	}

	bc.id = 0
	return conn.Close()
}

func (bc *BatchCursor) closeImplicitSession() {
	if bc.clientSession != nil && bc.clientSession.SessionType == session.Implicit {
		bc.clientSession.EndSession()
	}
}

func (bc *BatchCursor) clearBatch() {
	bc.currentBatch = bc.currentBatch[:0]
}

func (bc *BatchCursor) getMore(ctx context.Context) {
	bc.clearBatch()
	if bc.id == 0 {
		return
	}

	conn, err := bc.server.Connection(ctx)
	if err != nil {
		bc.err = err
		return
	}

	response, err := (&command.GetMore{
		Clock:   bc.clock,
		ID:      bc.id,
		NS:      bc.namespace,
		Opts:    bc.opts,
		Session: bc.clientSession,
	}).RoundTrip(ctx, bc.server.SelectedDescription(), conn)
	if err != nil {
		_ = conn.Close() // The command response error is more important here
		bc.err = err
		return
	}

	err = conn.Close()
	if err != nil {
		bc.err = err
		return
	}

	id, err := response.LookupErr("cursor", "id")
	if err != nil {
		bc.err = err
		return
	}
	var ok bool
	bc.id, ok = id.Int64OK()
	if !ok {
		bc.err = fmt.Errorf("BSON Type %s is not %s", id.Type, bson.TypeInt64)
		return
	}

	// if this is the last getMore, close the session
	if bc.id == 0 {
		bc.closeImplicitSession()
	}

	batch, err := response.LookupErr("cursor", "nextBatch")
	if err != nil {
		bc.err = err
		return
	}
	var arr bson.Raw
	arr, ok = batch.ArrayOK()
	if !ok {
		bc.err = fmt.Errorf("BSON Type %s is not %s", batch.Type, bson.TypeArray)
		return
	}
	vals, err := arr.Values()
	if err != nil {
		bc.err = err
		return
	}

	for _, val := range vals {
		if val.Type != bsontype.EmbeddedDocument {
			bc.err = fmt.Errorf("element of cursor batch is not a document, but at %s", val.Type)
			bc.currentBatch = bc.currentBatch[:0] // don't return a batch on error
			return
		}
		bc.currentBatch = append(bc.currentBatch, val.Value...)
	}

	return
}

func (bc *BatchCursor) legacy() bool {
	return bc.server.Description().WireVersion == nil || bc.server.Description().WireVersion.Max < 4
}

func (bc *BatchCursor) legacyKillCursor(ctx context.Context) error {
	conn, err := bc.server.Connection(ctx)
	if err != nil {
		return err
	}

	kc := wiremessage.KillCursors{
		NumberOfCursorIDs: 1,
		CursorIDs:         []int64{bc.id},
		CollectionName:    bc.namespace.Collection,
		DatabaseName:      bc.namespace.DB,
	}

	err = conn.WriteWireMessage(ctx, kc)
	if err != nil {
		_ = conn.Close()
		return err
	}

	err = conn.Close() // no reply from OP_KILL_CURSORS
	if err != nil {
		return err
	}

	bc.id = 0
	bc.clearBatch()
	return nil
}

func (bc *BatchCursor) legacyGetMore(ctx context.Context) {
	bc.clearBatch()
	if bc.id == 0 {
		return
	}

	conn, err := bc.server.Connection(ctx)
	if err != nil {
		bc.err = err
		return
	}

	numToReturn := bc.batchSize
	if bc.limit != 0 && bc.numReturned+bc.batchSize > bc.limit {
		numToReturn = bc.limit - bc.numReturned
	}
	gm := wiremessage.GetMore{
		FullCollectionName: bc.namespace.DB + "." + bc.namespace.Collection,
		CursorID:           bc.id,
		NumberToReturn:     numToReturn,
	}

	err = conn.WriteWireMessage(ctx, gm)
	if err != nil {
		_ = conn.Close()
		bc.err = err
		return
	}

	response, err := conn.ReadWireMessage(ctx)
	if err != nil {
		_ = conn.Close()
		bc.err = err
		return
	}

	err = conn.Close()
	if err != nil {
		bc.err = err
		return
	}

	reply, ok := response.(wiremessage.Reply)
	if !ok {
		bc.err = errors.New("did not receive OP_REPLY response")
		return
	}

	err = validateGetMoreReply(reply)
	if err != nil {
		bc.err = err
		return
	}

	bc.id = reply.CursorID
	bc.numReturned += reply.NumberReturned
	if bc.limit != 0 && bc.numReturned >= bc.limit {
		err = bc.Close(ctx)
		if err != nil {
			bc.err = err
			return
		}
	}

	for _, doc := range reply.Documents {
		bc.currentBatch = append(bc.currentBatch, doc...)
	}
}

func validateGetMoreReply(reply wiremessage.Reply) error {
	if int(reply.NumberReturned) != len(reply.Documents) {
		return command.NewCommandResponseError("malformed OP_REPLY: NumberReturned does not match number of returned documents", nil)
	}

	if reply.ResponseFlags&wiremessage.CursorNotFound == wiremessage.CursorNotFound {
		return command.QueryFailureError{
			Message: "query failure - cursor not found",
		}
	}
	if reply.ResponseFlags&wiremessage.QueryFailure == wiremessage.QueryFailure {
		return command.QueryFailureError{
			Message:  "query failure",
			Response: reply.Documents[0],
		}
	}

	return nil
}
