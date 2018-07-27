package mongo

import (
	"context"

	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/dispatch"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/mongo/readconcern"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
	"github.com/mongodb/mongo-go-driver/options"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"time"
)

const errorInterrupted int32 = 11601
const errorCappedPositionLost int32 = 136
const errorCursorKilled int32 = 237

// ErrMissingResumeToken indicates that a change stream notification from the server did not
// contain a resume token.
var ErrMissingResumeToken = errors.New("cannot provide resume functionality when the resume token is missing")

type changeStream struct {
	cmd      bsonx.Doc // aggregate command to run to create stream and rebuild cursor
	pipeline bsonx.Arr
	options  *options.ChangeStreamOptions
	coll     *Collection
	db       *Database
	ns       command.Namespace
	cursor   Cursor

	resumeToken bsonx.Doc
	err         error
	streamType  StreamType
	client      *Client
	sess        Session
	readPref    *readpref.ReadPref
	readConcern *readconcern.ReadConcern
}

func (cs *changeStream) replaceOptions(desc description.SelectedServer) {
	// if cs has not received any changes and resumeAfter not specified and max wire version >= 7, run known agg cmd
	// with startAtOperationTime set to startAtOperationTime provided by user or saved from initial agg
	// must not send resumeAfter key

	// else: run known agg cmd with resumeAfter set to last known resumeToken
	// must not set startAtOperationTime (remove if originally in cmd)

	if cs.options.ResumeAfter == nil && desc.WireVersion.Max >= 7 && cs.resumeToken == nil {
		cs.options.SetStartAtOperationTime(cs.sess.OperationTime())
	} else {
		if cs.resumeToken == nil {
			return // restart stream without the resume token
		}

		cs.options.SetResumeAfter(cs.resumeToken)
		// remove startAtOperationTime
		cs.options.SetStartAtOperationTime(nil)
	}
}

func createOptionsDoc(csType StreamType, opts *options.ChangeStreamOptions) (bsonx.Doc, error) {
	doc := bsonx.Doc{}
	if csType == ClientStream {
		doc = doc.Append("allChangesForCluster", bsonx.Boolean(true))
	}

	if opts.BatchSize != nil {
		doc = doc.Append("batchSize", bsonx.Int32(*opts.BatchSize))
	}
	if opts.Collation != nil {
		doc = doc.Append("collation", bsonx.Document(opts.Collation.ToDocument()))
	}
	if opts.FullDocument != nil {
		doc = doc.Append("fullDocument", bsonx.String(string(*opts.FullDocument)))
	}
	if opts.MaxAwaitTime != nil {
		ms := int64(time.Duration(*opts.MaxAwaitTime) / time.Millisecond)
		doc = doc.Append("maxAwaitTimeMS", bsonx.Int64(ms))
	}
	if opts.ResumeAfter != nil {
		doc = doc.Append("resumeAfter", bsonx.Document(opts.ResumeAfter))
	}
	if opts.StartAtOperationTime != nil {
		doc = doc.Append("startAtOperationTime", bsonx.Timestamp(opts.StartAtOperationTime.T, opts.StartAtOperationTime.I))
	}

	return doc, nil
}

func parseOptions(ctx context.Context, client *Client, csType StreamType,
	opts *options.ChangeStreamOptions) (bsonx.Doc, Session, error) {

	if opts.FullDocument == nil {
		opts = opts.SetFullDocument(options.Default)
	}

	sess := sessionFromContext(ctx)
	if err := client.ValidSession(sess); err != nil {
		return nil, nil, err
	}

	optionsDoc, err := createOptionsDoc(csType, opts)
	if err != nil {
		return nil, nil, err
	}

	var mongoSess Session
	if sess != nil {
		mongoSess = &sessionImpl{
			Client: sess,
		}
	} else {
		// create implicit session because it will be needed
		newSess, err := session.NewClientSession(client.topology.SessionPool, client.id, session.Implicit)
		if err != nil {
			return nil, nil, err
		}

		mongoSess = &sessionImpl{
			Client: newSess,
		}
	}

	return optionsDoc, mongoSess, nil
}

func (cs *changeStream) runCommand(ctx context.Context, replaceOptions bool) error {
	ss, err := cs.client.topology.SelectServer(ctx, cs.db.writeSelector)
	if err != nil {
		return err
	}

	desc := ss.Description()
	conn, err := ss.Connection(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if replaceOptions {
		cs.replaceOptions(desc)
		optionsDoc, err := createOptionsDoc(cs.streamType, cs.options)
		if err != nil {
			return err
		}

		changeStreamDoc := bsonx.Doc{
			{"$changeStream", bsonx.Document(optionsDoc)},
		}
		cs.pipeline[0] = bsonx.Document(changeStreamDoc)
		cs.cmd.Set("pipeline", bsonx.Array(cs.pipeline))
	}

	readCmd := command.Read{
		DB:          cs.db.name,
		Command:     cs.cmd,
		Session:     cs.sess.(*sessionImpl).Client,
		Clock:       cs.client.clock,
		ReadPref:    cs.readPref,
		ReadConcern: cs.readConcern,
	}

	rdr, err := readCmd.RoundTrip(ctx, desc, conn)
	if err != nil {
		cs.sess.EndSession(ctx)
		return err
	}

	cursor, err := ss.BuildCursor(rdr, readCmd.Session, readCmd.Clock)
	if err != nil {
		cs.sess.EndSession(ctx)
		return err
	}
	cs.cursor = cursor

	// can get resume token from initial aggregate command if non-empty batch
	// operationTime from aggregate saved in the session
	cursorValue, err := rdr.LookupErr("cursor")
	if err != nil {
		return err
	}
	cursorDoc := cursorValue.Document()

	cs.ns = command.ParseNamespace(cursorDoc.Lookup("ns").StringValue())

	batchVal := cursorDoc.Lookup("firstBatch")
	if err != nil {
		return err
	}

	batch := batchVal.Array()
	elements, err := batch.Elements()
	if err != nil {
		return err
	}

	if len(elements) == 0 {
		return nil // no resume token
	}

	firstElem, err := batch.IndexErr(0)
	if err != nil {
		return err
	}

	tokenDoc, err := bsonx.ReadDoc(firstElem.Value().Document().Lookup("_id").Document())
	if err != nil {
		return err
	}

	cs.resumeToken = tokenDoc
	return nil
}

func newChangeStream(ctx context.Context, coll *Collection, pipeline interface{},
	opts ...*options.ChangeStreamOptions) (*changeStream, error) {

	pipelineArr, err := transformAggregatePipeline(coll.registry, pipeline)
	if err != nil {
		return nil, err
	}

	csOpts := options.MergeChangeStreamOptions(opts...)
	optsDoc, sess, err := parseOptions(ctx, coll.client, CollectionStream, csOpts)
	if err != nil {
		return nil, err
	}

	csDoc := bsonx.Document(bsonx.Doc{
		{"$changeStream", bsonx.Document(optsDoc)},
	})
	pipelineArr = append(bsonx.Arr{csDoc}, pipelineArr...)

	cmd := bsonx.Doc{
		{"aggregate", bsonx.String(coll.name)},
		{"pipeline", bsonx.Array(pipelineArr)},
		{"cursor", bsonx.Document(bsonx.Doc{})},
	}

	cs := &changeStream{
		client:      coll.client,
		sess:        sess,
		cmd:         cmd,
		pipeline:    pipelineArr,
		coll:        coll,
		db:          coll.db,
		streamType:  CollectionStream,
		readPref:    coll.readPreference,
		readConcern: coll.readConcern,
		options:     csOpts,
	}

	err = cs.runCommand(ctx, false)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func newDbChangeStream(ctx context.Context, db *Database, pipeline interface{},
	opts ...*options.ChangeStreamOptions) (*changeStream, error) {

	pipelineArr, err := transformAggregatePipeline(db.registry, pipeline)
	if err != nil {
		return nil, err
	}

	csOpts := options.MergeChangeStreamOptions(opts...)
	optsDoc, sess, err := parseOptions(ctx, db.client, DatabaseStream, csOpts)
	if err != nil {
		return nil, err
	}

	csDoc := bsonx.Document(bsonx.Doc{
		{"$changeStream", bsonx.Document(optsDoc)},
	})
	pipelineArr = append(bsonx.Arr{csDoc}, pipelineArr...)

	cmd := bsonx.Doc{
		{"aggregate", bsonx.Int32(1)},
		{"pipeline", bsonx.Array(pipelineArr)},
		{"cursor", bsonx.Document(bsonx.Doc{})},
	}

	cs := &changeStream{
		client:      db.client,
		db:          db,
		sess:        sess,
		cmd:         cmd,
		pipeline:    pipelineArr,
		streamType:  DatabaseStream,
		readPref:    db.readPreference,
		readConcern: db.readConcern,
		options:     csOpts,
	}

	err = cs.runCommand(ctx, false)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func newClientChangeStream(ctx context.Context, client *Client, pipeline interface{},
	opts ...*options.ChangeStreamOptions) (*changeStream, error) {

	pipelineArr, err := transformAggregatePipeline(client.registry, pipeline)
	if err != nil {
		return nil, err
	}

	csOpts := options.MergeChangeStreamOptions(opts...)
	optsDoc, sess, err := parseOptions(ctx, client, ClientStream, csOpts)
	if err != nil {
		return nil, err
	}

	csDoc := bsonx.Document(bsonx.Doc{
		{"$changeStream", bsonx.Document(optsDoc)},
	})
	pipelineArr = append(bsonx.Arr{csDoc}, pipelineArr...)

	cmd := bsonx.Doc{
		{"aggregate", bsonx.Int32(1)},
		{"pipeline", bsonx.Array(pipelineArr)},
		{"cursor", bsonx.Document(bsonx.Doc{})},
	}

	cs := &changeStream{
		client:      client,
		db:          client.Database("admin"),
		sess:        sess,
		cmd:         cmd,
		pipeline:    pipelineArr,
		streamType:  ClientStream,
		readPref:    client.readPreference,
		readConcern: client.readConcern,
		options:     csOpts,
	}

	err = cs.runCommand(ctx, false)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func (cs *changeStream) ID() int64 {
	return cs.cursor.ID()
}

func (cs *changeStream) Next(ctx context.Context) bool {
	if cs.cursor.Next(ctx) {
		return true
	}

	err := cs.cursor.Err()
	if err == nil {
		return false
	}

	switch t := err.(type) {
	case command.Error:
		if t.Code == errorInterrupted || t.Code == errorCappedPositionLost || t.Code == errorCursorKilled {
			return false
		}
	}

	killCursors := command.KillCursors{
		NS:  cs.ns,
		IDs: []int64{cs.ID()},
	}

	_, _ = dispatch.KillCursors(ctx, killCursors, cs.client.topology, cs.db.writeSelector)
	cs.err = cs.runCommand(ctx, true)
	if cs.err != nil {
		return false
	}

	return true
}

func (cs *changeStream) Decode(out interface{}) error {
	br, err := cs.DecodeBytes()
	if err != nil {
		return err
	}

	return bson.UnmarshalWithRegistry(cs.coll.registry, br, out)
}

func (cs *changeStream) DecodeBytes() (bson.Raw, error) {
	br, err := cs.cursor.DecodeBytes()
	if err != nil {
		return nil, err
	}

	idVal, err := br.LookupErr("_id")
	if err != nil {
		_ = cs.Close(context.Background())
		return nil, ErrMissingResumeToken
	}

	tokenDoc, err := bsonx.ReadDoc(idVal.Document())
	if err != nil {
		_ = cs.Close(context.Background())
		return nil, ErrMissingResumeToken
	}

	cs.resumeToken = tokenDoc
	return br, nil
}

func (cs *changeStream) Err() error {
	if cs.err != nil {
		return cs.err
	}

	return cs.cursor.Err()
}

func (cs *changeStream) Close(ctx context.Context) error {
	return cs.cursor.Close(ctx)
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
