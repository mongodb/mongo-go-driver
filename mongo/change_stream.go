package mongo

import (
	"context"

	"bytes"
	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/dispatch"
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/mongo/changestreamopt"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

const errorInterrupted int32 = 11601
const errorCappedPositionLost int32 = 136
const errorCursorKilled int32 = 237

// ErrMissingResumeToken indicates that a change stream notification from the server did not
// contain a resume token.
var ErrMissingResumeToken = errors.New("cannot provide resume functionality when the resume token is missing")

type changeStream struct {
	cmd         *bson.Document // aggregate command to run to create stream and rebuild cursor
	pipeline    *bson.Array
	options     []option.ChangeStreamOptioner
	coll        *Collection
	db          *Database
	ns          command.Namespace
	cursor      Cursor
	resumeToken *bson.Document
	err         error
	streamType  StreamType
	client      *Client
	sess        *Session
	readPref    *readpref.ReadPref
	readConcern *readconcern.ReadConcern
}

func (cs *changeStream) replaceOptions(desc description.SelectedServer) {
	resumeTokenIndex := -1
	optimeIndex := -1

	for i, opt := range cs.options {
		if _, ok := opt.(option.OptResumeAfter); ok {
			resumeTokenIndex = i
		} else if _, ok := opt.(option.OptStartAtOperationTime); ok {
			optimeIndex = i
		}
	}

	// if cs has not received any changes and resumeAfter not specified and max wire version >= 7, run known agg cmd
	// with startAtOperationTime set to startAtOperationTime provided by user or saved from initial agg
	// must not send resumeAfter key

	// else: run known agg cmd with resumeAfter set to last known resumeToken
	// must not set startAtOperationTime (remove if originally in cmd)

	if resumeTokenIndex == -1 && desc.WireVersion.Max >= 7 && cs.resumeToken == nil {
		// use last known operation time
		startTimeOpt := changestreamopt.StartAtOperationTime(cs.sess.OperationTime).ConvertChangeStreamOption()
		if optimeIndex != -1 {
			cs.options[optimeIndex] = startTimeOpt
		} else {
			cs.options = append(cs.options, startTimeOpt)
		}

		// remove resume token
		if resumeTokenIndex != -1 {
			cs.options = append(cs.options[:resumeTokenIndex], cs.options[resumeTokenIndex+1:]...)
		}
	} else {
		if cs.resumeToken == nil {
			return // restart stream without the resume token
		}

		resumeOpt := changestreamopt.ResumeAfter(cs.resumeToken).ConvertChangeStreamOption()
		if resumeTokenIndex != -1 {
			cs.options[resumeTokenIndex] = resumeOpt
		} else {
			cs.options = append(cs.options, resumeOpt)
		}

		// remove startAtOperationTime
		if optimeIndex != -1 {
			cs.options = append(cs.options[:optimeIndex], cs.options[optimeIndex+1:]...)
		}
	}
}

func createOptionsDoc(csType StreamType, opts ...option.ChangeStreamOptioner) (*bson.Document, error) {
	doc := bson.NewDocument()
	if csType == ClientStream {
		doc.Append(bson.EC.Boolean("allChangesForCluster", true))
	}

	for _, opt := range opts {
		err := opt.Option(doc)
		if err != nil {
			return nil, err
		}
	}

	return doc, nil
}

func parseOptions(client *Client, csType StreamType,
	opts ...changestreamopt.ChangeStream) ([]option.ChangeStreamOptioner, *bson.Document, *Session, error) {

	var foundFullDoc bool
	for _, opt := range opts {
		if _, ok := opt.(changestreamopt.OptFullDocument); ok {
			foundFullDoc = true
		}
	}

	if !foundFullDoc {
		opts = append(opts, changestreamopt.FullDocument(mongoopt.Default))
	}

	csOpts, sess, err := changestreamopt.BundleChangeStream(opts...).Unbundle(true)
	if err != nil {
		return nil, nil, nil, err
	}

	if err = client.ValidSession(sess); err != nil {
		return nil, nil, nil, err
	}

	optionsDoc, err := createOptionsDoc(csType, csOpts...)
	if err != nil {
		return nil, nil, nil, err
	}

	var mongoSess *Session
	if sess != nil {
		mongoSess = &Session{
			Client: sess,
		}
	} else {
		// create implicit session because it will be needed
		newSess, err := session.NewClientSession(client.topology.SessionPool, client.id, session.Implicit)
		if err != nil {
			return nil, nil, nil, err
		}

		mongoSess = &Session{
			Client: newSess,
		}
	}

	return csOpts, optionsDoc, mongoSess, nil
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
		optionsDoc, err := createOptionsDoc(cs.streamType, cs.options...)
		if err != nil {
			return err
		}

		cs.pipeline.Set(0, bson.VC.Document(
			bson.NewDocument(
				bson.EC.SubDocument("$changeStream", optionsDoc),
			),
		))
		cs.cmd.Set(bson.EC.Array("pipeline", cs.pipeline))
	}

	readCmd := command.Read{
		DB:          cs.db.name,
		Command:     cs.cmd,
		Session:     cs.sess.Client,
		Clock:       cs.client.clock,
		ReadPref:    cs.readPref,
		ReadConcern: cs.readConcern,
	}

	rdr, err := readCmd.RoundTrip(ctx, desc, conn)
	if err != nil {
		cs.sess.EndSession()
		return err
	}

	cursor, err := ss.BuildCursor(rdr, readCmd.Session, readCmd.Clock)
	if err != nil {
		cs.sess.EndSession()
		return err
	}
	cs.cursor = cursor

	// can get resume token from initial aggregate command if non-empty batch
	// operationTime from aggregate saved in the session
	cursorElem, err := rdr.Lookup("cursor")
	if err != nil {
		return err
	}
	cursorDoc := cursorElem.Value().MutableDocument()

	cs.ns = command.ParseNamespace(cursorDoc.Lookup("ns").StringValue())

	batchVal := cursorDoc.Lookup("firstBatch")
	if err != nil {
		return err
	}

	batch := batchVal.MutableArray()
	if batch.Len() == 0 {
		return nil // no resume token
	}

	firstVal, err := batch.Lookup(0)
	if err != nil {
		return err
	}

	cs.resumeToken = firstVal.MutableDocument().Lookup("_id").MutableDocument()
	return nil
}

func newChangeStream(ctx context.Context, coll *Collection, pipeline interface{},
	opts ...changestreamopt.ChangeStream) (*changeStream, error) {

	pipelineArr, err := transformAggregatePipeline(pipeline)
	if err != nil {
		return nil, err
	}

	coreOpts, optsDoc, sess, err := parseOptions(coll.client, CollectionStream, opts...)
	if err != nil {
		return nil, err
	}

	pipelineArr.Prepend(
		bson.VC.Document(
			bson.NewDocument(
				bson.EC.SubDocument("$changeStream", optsDoc),
			),
		),
	)

	cmd := bson.NewDocument(
		bson.EC.String("aggregate", coll.name),
		bson.EC.Array("pipeline", pipelineArr),
		bson.EC.SubDocument("cursor", bson.NewDocument()),
	)

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
		options:     coreOpts,
	}

	err = cs.runCommand(ctx, false)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func newDbChangeStream(ctx context.Context, db *Database, pipeline interface{},
	opts ...changestreamopt.ChangeStream) (*changeStream, error) {

	pipelineArr, err := transformAggregatePipeline(pipeline)
	if err != nil {
		return nil, err
	}

	coreOpts, optsDoc, sess, err := parseOptions(db.client, DatabaseStream, opts...)
	if err != nil {
		return nil, err
	}

	pipelineArr.Prepend(
		bson.VC.Document(
			bson.NewDocument(
				bson.EC.SubDocument("$changeStream", optsDoc),
			),
		),
	)

	cmd := bson.NewDocument(
		bson.EC.Int32("aggregate", 1),
		bson.EC.Array("pipeline", pipelineArr),
		bson.EC.SubDocument("cursor", bson.NewDocument()),
	)

	cs := &changeStream{
		client:      db.client,
		db:          db,
		sess:        sess,
		cmd:         cmd,
		pipeline:    pipelineArr,
		streamType:  DatabaseStream,
		readPref:    db.readPreference,
		readConcern: db.readConcern,
		options:     coreOpts,
	}

	err = cs.runCommand(ctx, false)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func newClientChangeStream(ctx context.Context, client *Client, pipeline interface{},
	opts ...changestreamopt.ChangeStream) (*changeStream, error) {

	pipelineArr, err := transformAggregatePipeline(pipeline)
	if err != nil {
		return nil, err
	}

	coreOpts, optsDoc, sess, err := parseOptions(client, ClientStream, opts...)
	if err != nil {
		return nil, err
	}

	pipelineArr.Prepend(
		bson.VC.Document(
			bson.NewDocument(
				bson.EC.SubDocument("$changeStream", optsDoc),
			),
		),
	)

	cmd := bson.NewDocument(
		bson.EC.Int32("aggregate", 1),
		bson.EC.Array("pipeline", pipelineArr),
		bson.EC.SubDocument("cursor", bson.NewDocument()),
	)

	cs := &changeStream{
		client:      client,
		db:          client.Database("admin"),
		sess:        sess,
		cmd:         cmd,
		pipeline:    pipelineArr,
		streamType:  ClientStream,
		readPref:    client.readPreference,
		readConcern: client.readConcern,
		options:     coreOpts,
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

	return bson.NewDecoder(bytes.NewReader(br)).Decode(out)
}

func (cs *changeStream) DecodeBytes() (bson.Reader, error) {
	br, err := cs.cursor.DecodeBytes()
	if err != nil {
		return nil, err
	}

	id, err := br.Lookup("_id")
	if err != nil {
		_ = cs.Close(context.Background())
		return nil, ErrMissingResumeToken
	}

	cs.resumeToken = id.Value().MutableDocument()
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
