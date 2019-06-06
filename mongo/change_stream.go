package mongo

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"reflect"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
)

const errorInterrupted int32 = 11601
const errorCappedPositionLost int32 = 136
const errorCursorKilled int32 = 237

// ErrMissingResumeToken indicates that a change stream notification from the server did not
// contain a resume token.
var ErrMissingResumeToken = errors.New("cannot provide resume functionality when the resume token is missing")

// ErrNilCursor indicates that the cursor for the change stream is nil.
var ErrNilCursor = errors.New("cursor is nil")

// ChangeStream instances iterate a stream of change documents. Each document can be decoded via the
// Decode method. Resume tokens should be retrieved via the ResumeToken method and can be stored to
// resume the change stream at a specific point in time.
//
// A typical usage of the ChangeStream type would be:
type ChangeStream struct {
	Current        bson.Raw
	aggregate      *operation.Aggregate
	pipelineSlice  []bsoncore.Document
	cursor         pbrtBatchCursor
	cursorOptions  driver.CursorOptions
	batch          []bsoncore.Document
	resumeToken    bson.Raw
	err            error
	sess           *session.Client // needed for operation
	operationTime  *primitive.Timestamp
	readConcern    *readconcern.ReadConcern
	readPreference *readpref.ReadPref
	client         *Client
	registry       *bsoncodec.Registry
	streamType     StreamType
	options        *options.ChangeStreamOptions
}

type changeStreamConfig struct {
	readConcern    *readconcern.ReadConcern
	readPreference *readpref.ReadPref
	client         *Client
	registry       *bsoncodec.Registry
	streamType     StreamType
	collectionName     string
	databaseName       string
}

func newChangeStream(ctx context.Context, config changeStreamConfig, pipeline interface{},
	opts ...*options.ChangeStreamOptions) (*ChangeStream, error) {
	cs := &ChangeStream{
		err:            nil,
		readConcern:    config.readConcern,
		readPreference: config.readPreference,
		client:         config.client,
		registry:       config.registry,
		streamType:     config.streamType,
		options:        options.MergeChangeStreamOptions(opts...),
	}

	cs.aggregate = operation.NewAggregate(nil).
		ReadPreference(cs.readPreference).ReadConcern(cs.readConcern).
		Deployment(cs.client.topology).ClusterClock(cs.client.clock)

	if cs.options.BatchSize != nil {
		cs.cursorOptions.BatchSize = *cs.options.BatchSize
	}
	if cs.options.MaxAwaitTime != nil {
		cs.cursorOptions.MaxTimeMS = int64(time.Duration(*cs.options.MaxAwaitTime) / time.Millisecond)
	}
	cs.cursorOptions.CommandMonitor = cs.client.monitor

	switch cs.streamType {
	case ClientStream:
		cs.aggregate.Database("admin")
	case DatabaseStream:
		cs.aggregate.Database(config.databaseName)
	case CollectionStream:
		cs.aggregate.Collection(config.collectionName).Database(config.databaseName)
	default:
		return nil, fmt.Errorf("must supply a valid StreamType in config, instead of %v", cs.streamType)
	}

	if cs.buildPipelineSlice(pipeline); cs.err != nil {
		return nil, cs.err
	}

	resTok := cs.options.StartAfter
	var marshaledToken bson.Raw

	if resTok == nil {
		resTok = cs.options.ResumeAfter
	} else {
		marshaledToken, cs.err = bson.Marshal(resTok)
		if cs.err != nil {
			return nil, cs.err
		}
	}

	cs.resumeToken = marshaledToken
	cs.sess = sessionFromContext(ctx)
	if cs.err = cs.client.validSession(cs.sess); cs.err != nil {
		return nil, cs.err
	}

	var pipelineArr bsoncore.Document
	pipelineArr, cs.err = cs.buildPipelineArray()
	cs.aggregate.Pipeline(pipelineArr)

	cs.executeOperation(ctx, false)
	if cs.err != nil {
		return nil, cs.err
	}

	return cs, cs.err
}

func (cs *ChangeStream) executeOperation(ctx context.Context, replaceOptions bool) {
	if replaceOptions {
		cs.replaceOptions(ctx)
		var plArr bsoncore.Document
		plArr, cs.err = cs.buildPipelineArray()
		if cs.err != nil {
			return
		}
		cs.aggregate = cs.aggregate.Pipeline(plArr)
	}

	cs.err = replaceErrors(cs.aggregate.Execute(ctx))
	if cs.err != nil {
		return
	}

	cs.cursor, cs.err = cs.aggregate.Result(cs.cursorOptions)
	cs.err = replaceErrors(cs.err)
}

// ID returns the cursor ID for this change stream.
func (cs *ChangeStream) ID() int64 {
	if cs.cursor == nil {
		return 0
	}
	return cs.cursor.ID()
}

// Next gets the next result from this change stream. Returns true if there were no errors and the next
// result is available for decoding.
func (cs *ChangeStream) Next(ctx context.Context) bool {
	if ctx == nil {
		ctx = context.Background()
	}

	if len(cs.batch) == 0 {
		cs.loopNext(ctx)
		if cs.err != nil || len(cs.batch) == 0 {
			return false
		}
	}

	cs.Current = bson.Raw(cs.batch[0])
	cs.batch = cs.batch[1:]
	err := cs.storeResumeToken()
	if err != nil {
		cs.err = err
		return false
	}
	return true
}

// Decode will decode the current document into val.
func (cs *ChangeStream) Decode(out interface{}) error {
	if cs.cursor == nil {
		return ErrNilCursor
	}

	return bson.UnmarshalWithRegistry(cs.registry, cs.Current, out)
}

// Err returns the current error.
func (cs *ChangeStream) Err() error {
	if cs.err != nil {
		return replaceErrors(cs.err)
	}
	if cs.cursor == nil {
		return nil
	}

	return cs.cursor.Err()
}

// Close closes this cursor.
func (cs *ChangeStream) Close(ctx context.Context) error {
	if cs.cursor == nil {
		return nil // cursor is already closed
	}

	return replaceErrors(cs.cursor.Close(ctx))
}

// ResumeToken returns the last cached resume token for this change stream.
func (cs *ChangeStream) ResumeToken() bson.Raw {
	return cs.resumeToken
}

func (cs *ChangeStream) loopNext(ctx context.Context) {
	for {
		if cs.cursor == nil {
			return
		}

		if cs.cursor.Next(ctx) {
			// If this is the first batch, the batch cursor will return true, but the batch could be empty.
			cs.batch, cs.err = cs.cursor.Batch().Documents()
			if cs.err != nil || len(cs.batch) > 0 {
				return
			}

			// no error but empty batch
			if pbrt := cs.cursor.PostBatchResumeToken(); cs.emptyBatch() && pbrt != nil {
				cs.resumeToken = bson.Raw(pbrt)
			}
			continue
		}

		cs.err = replaceErrors(cs.cursor.Err())
		if cs.err == nil {
			// If a getMore was done but the batch was empty, the batch cursor will return false with no error
			if len(cs.batch) == 0 {
				continue
			}

			return
		}

		switch t := cs.err.(type) {
		case CommandError:
			if t.Code == errorInterrupted || t.Code == errorCappedPositionLost || t.Code == errorCursorKilled || t.HasErrorLabel("NonResumableChangeStreamError") {
				return
			}
		}

		if cs.err = cs.cursor.Close(ctx); cs.err != nil {
			return
		}

		if cs.executeOperation(ctx, true); cs.err != nil {
			return
		}
	}
}

func (cs *ChangeStream) storeResumeToken() error {
	// If cs.Current is the last document in the batch and a pbrt is included, cache the pbrt
	// Otherwise, cache the _id of the document

	var tokenDoc bson.Raw
	if len(cs.batch) == 0 {
		if pbrt := cs.cursor.PostBatchResumeToken(); pbrt != nil {
			tokenDoc = bson.Raw(pbrt)
		}
	}

	if tokenDoc == nil {
		var ok bool
		tokenDoc, ok = cs.Current.Lookup("_id").DocumentOK()
		if !ok {
			_ = cs.Close(context.Background())
			return ErrMissingResumeToken
		}
	}

	cs.resumeToken = tokenDoc
	return nil
}

func (cs *ChangeStream) buildPipelineSlice(pipeline interface{}) {
	val := reflect.ValueOf(pipeline)
	if !val.IsValid() || !(val.Kind() == reflect.Slice) {
		cs.err = fmt.Errorf("unable to parse pipeline: %v", pipeline)
		return
	}

	cs.pipelineSlice = make([]bsoncore.Document, val.Len()+1, val.Len()+1)

	var elem []byte
	for i := 0; i < val.Len(); i++ {
		elem, cs.err = transformBsoncoreDocument(cs.registry, val.Index(i).Interface())
		if cs.err != nil {
			return
		}

		cs.pipelineSlice[i+1] = elem
	}

	csIdx, csDoc := bsoncore.AppendDocumentStart(nil)
	csDocTemp := cs.createPipelineOptionsDoc()
	if cs.err != nil {
		return
	}
	csDoc = bsoncore.AppendDocumentElement(csDoc, "$changeStream", csDocTemp)
	csDoc, cs.err = bsoncore.AppendDocumentEnd(csDoc, csIdx)
	if cs.err != nil {
		return
	}
	cs.pipelineSlice[0] = csDoc
}

func (cs *ChangeStream) createPipelineOptionsDoc() bsoncore.Document {
	plDocIdx, plDoc := bsoncore.AppendDocumentStart(nil)

	if cs.streamType == ClientStream {
		plDoc = bsoncore.AppendBooleanElement(plDoc, "allChangesForCluster", true)
	}

	if cs.options.FullDocument != nil {
		plDoc = bsoncore.AppendStringElement(plDoc, "fullDocument", string(*cs.options.FullDocument))
	}

	if cs.options.ResumeAfter != nil {
		var raDoc bsoncore.Document
		raDoc, cs.err = transformBsoncoreDocument(cs.registry, cs.options.ResumeAfter)
		if cs.err != nil {
			return nil
		}

		plDoc = bsoncore.AppendDocumentElement(plDoc, "resumeAfter", raDoc)
	}

	if cs.options.StartAfter != nil {
		var saDoc bsoncore.Document
		saDoc, cs.err = transformBsoncoreDocument(cs.registry, cs.options.StartAfter)
		if cs.err != nil {
			return nil
		}

		plDoc = bsoncore.AppendDocumentElement(plDoc, "startAfter", saDoc)
	}

	if cs.options.StartAtOperationTime != nil {
		plDoc = bsoncore.AppendTimestampElement(plDoc, "startAtOperationTime", cs.options.StartAtOperationTime.T, cs.options.StartAtOperationTime.I)
	}

	plDoc, cs.err = bsoncore.AppendDocumentEnd(plDoc, plDocIdx)
	if cs.err != nil {
		return nil
	}

	return plDoc
}

func (cs *ChangeStream) buildPipelineArray() (bsoncore.Document, error) {
	pipelineDocIdx, pipelineArr := bsoncore.AppendArrayStart(nil)
	for i, doc := range cs.pipelineSlice {
		pipelineArr = bsoncore.AppendDocumentElement(pipelineArr, strconv.Itoa(i), doc)
	}
	pipelineArr, cs.err = bsoncore.AppendArrayEnd(pipelineArr, pipelineDocIdx)
	if cs.err != nil {
		return nil, cs.err
	}
	return pipelineArr, cs.err
}

// Returns true if the underlying cursor's batch is empty
func (cs *ChangeStream) emptyBatch() bool {
	return len(cs.cursor.Batch().Data) == 5 // empty BSON array
}

func (cs *ChangeStream) replaceOptions(ctx context.Context) {
	// Cached resume token: use the resume token as the resumeAfter option and set no other resume options
	if cs.resumeToken != nil {
		cs.options.SetResumeAfter(cs.resumeToken)
		cs.options.SetStartAfter(nil)
		cs.options.SetStartAtOperationTime(nil)
		return
	}

	// No cached resume token but cached operation time: use the operation time as the startAtOperationTime option and
	// set no other resume options

	var conn driver.Connection
	conn, cs.err = cs.cursor.Server().Connection(ctx)
	if cs.err != nil {
		return
	}

	if (cs.operationTime != nil || cs.options.StartAtOperationTime != nil) && conn.Description().WireVersion.Max >= 7 {
		opTime := cs.options.StartAtOperationTime
		if cs.operationTime != nil {
			opTime = cs.operationTime
		}

		cs.options.SetStartAtOperationTime(opTime)
		cs.options.SetResumeAfter(nil)
		cs.options.SetStartAfter(nil)
		return
	}

	// No cached resume token or operation time: set none of the resume options
	cs.options.SetResumeAfter(nil)
	cs.options.SetStartAfter(nil)
	cs.options.SetStartAtOperationTime(nil)
}

// StreamType represents the type of a change stream.
type StreamType uint8

// These constants represent valid change stream types. A change stream can be initialized over a collection, all
// collections in a database, or over a whole client.
const (
	CollectionStream StreamType = iota
	DatabaseStream
	ClientStream
)
