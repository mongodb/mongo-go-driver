package driverx

import (
	"context"
	"errors"

	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
	"go.mongodb.org/mongo-driver/x/network/wiremessagex"
)

//go:generate drivergen FindOperation find.generated.go

type FindOperation struct {
	// Filter determines what results are returned from find.
	filter bson.Raw

	// Sort specifies the order in which to return results.
	sort bson.Raw

	// Project limits the fields returned for all documents.
	projection bson.Raw

	// Hint specifies the index to use.
	hint bson.RawValue

	// Skip specifies the number of documents to skip before returning.
	skip *int64

	// Limit sets a limit on the number of documents to return.
	limit *int64

	// BatchSize specifies the number of documents to return in every batch.
	batchSize *int64

	// SingleBatch specifies whether the results should be returned in a single batch.
	singleBatch *bool

	// Comment sets a string to help trace an operation.
	comment *string

	// MaxTimeMS specifies the maximum amount of time to allow the query to run.
	maxTimeMS *int64

	// ReadConcern specifies the read concern for this operation.
	readConcern *readconcern.ReadConcern `drivergen:",pointerExempt"`

	// Max sets an exclusive upper bound for a specific index.
	max bson.Raw

	// Min sets an inclusive lower bound for a specific index.
	min bson.Raw

	// ReturnKey when true returns index keys for all result documents.
	returnKey *bool

	// ShowRecordID when true adds a $recordId field with the record identifier to returned documents.
	showRecordID *bool

	// OplogReplay when true replays a replica set's oplog.
	oplogReplay *bool

	// NoCursorTimeout when true prevents cursor from timing out after an inactivity period.
	noCursorTimeout *bool

	// Tailable keeps a cursor open and resumable after the last data has been retrieved.
	tailable *bool

	// AwaitData when true makes a cursor block before returning when no data is available.
	awaitData *bool

	// AllowPartialResults when true allows partial results to be returned if some shards are down.
	allowPartialResults *bool

	// Collation specifies a collation to be used.
	collation bson.Raw

	// Namespace sets the database and collection to run this operation against.
	ns Namespace `drivergen:"Namespace"`

	// Deployment sets the deployment to use for this operation.
	d Deployment `drivergen:"Deployment"`

	// ServerSelector sets the selector used to retrieve a server.
	serverSelector description.ServerSelector

	// ReadPreference set the read prefernce used with this operation.
	readPref *readpref.ReadPref `drivergen:"ReadPreference,pointerExempt"`

	// Session sets the session for this operation.
	client *session.Client `drivergen:"Session,pointerExempt"`

	// ClusterClock sets the cluster clock for this operation.
	clock *session.ClusterClock `drivergen:"ClusterClock,pointerExempt"`
}

// Find creates and returns a FindOperation.
func Find(filter bson.Raw) FindOperation { return FindOperation{filter: filter} }

// Select retrieves a server to be used when executing an operation.
func (fo FindOperation) Select(ctx context.Context) (Server, error) {
	if fo.d == nil {
		return nil, errors.New("FindOperation must have a Deployment set before Select can be called.")
	}
	return fo.d.SelectServer(ctx, createReadPrefSelector(fo.serverSelector, fo.readPref))
}

func (fo FindOperation) SelectAndExecute(ctx context.Context) (*BatchCursor, error) {
	srvr, err := fo.Select(ctx)
	if err != nil {
		return nil, err
	}
	return fo.Execute(ctx, srvr)
}

func (fo FindOperation) Execute(ctx context.Context, srvr Server) (*BatchCursor, error) {
	if fo.d == nil {
		return nil, errors.New("FindOperation must have a Deployment set before Execute can be called.")
	}
	conn, err := srvr.Connection(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	desc := description.SelectedServer{Server: conn.Description(), Kind: fo.d.Description().Kind}

	if desc.WireVersion == nil || desc.WireVersion.Max < 4 {
		return nil, errors.New("legacy not currently supported")
	}

	wm, err := fo.createWireMessage(nil, desc)
	if err != nil {
		return nil, err
	}

	res, err := roundTripDecode(ctx, conn, wm)
	if err != nil {
		return nil, err
	}

	spew.Dump(res)
	return nil, nil
}

func (fo FindOperation) createWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	if fo.ns.Collection == "" || fo.ns.DB == "" {
		return nil, errors.New("Collection and DB must be of non-zero length")
	}
	rp := createReadPref(fo.readPref, desc.Server.Kind, desc.Kind, true)
	if desc.WireVersion == nil || desc.WireVersion.Max < wiremessage.OpmsgWireVersion {
		return createQueryWireMessage(dst, desc, fo.ns.DB, rp, fo.queryAppendCommand)
	}
	return fo.createMsgWireMessage(dst, desc)
}

func (fo FindOperation) createMsgWireMessage(dst []byte, desc description.SelectedServer) ([]byte, error) {
	// TODO(GODRIVER-617): How do we allow users to supply flags? Perhaps we don't and we add
	// functions to allow users to set them themselves.
	var flags wiremessage.MsgFlag
	var wmindex int32
	wmindex, dst = wiremessagex.AppendHeaderStart(dst, wiremessage.NextRequestID(), 0, wiremessage.OpMsg)
	dst = wiremessagex.AppendMsgFlags(dst, flags)
	// Body
	dst = wiremessagex.AppendMsgSectionType(dst, wiremessage.SingleDocument)

	idx, dst := bsoncore.AppendDocumentStart(dst)

	dst = bsoncore.AppendStringElement(dst, "find", fo.ns.Collection)

	if fo.filter != nil {
		dst = bsoncore.AppendDocumentElement(dst, "filter", fo.filter)
	}
	if fo.sort != nil {
		dst = bsoncore.AppendDocumentElement(dst, "sort", fo.sort)
	}
	if fo.projection != nil {
		dst = bsoncore.AppendDocumentElement(dst, "projection", fo.projection)
	}
	if fo.hint.Type != bsontype.Type(0) {
		dst = bsoncore.AppendHeader(dst, fo.hint.Type, "hint")
		dst = append(dst, fo.hint.Value...)
	}
	if fo.skip != nil {
		dst = bsoncore.AppendInt64Element(dst, "skip", *fo.skip)
	}
	if fo.limit != nil {
		dst = bsoncore.AppendInt64Element(dst, "limit", *fo.limit)
	}
	if fo.batchSize != nil {
		dst = bsoncore.AppendInt64Element(dst, "batchSize", *fo.batchSize)
	}
	if fo.singleBatch != nil {
		dst = bsoncore.AppendBooleanElement(dst, "singleBatch", *fo.singleBatch)
	}
	if fo.comment != nil {
		dst = bsoncore.AppendStringElement(dst, "comment", *fo.comment)
	}
	if fo.maxTimeMS != nil {
		dst = bsoncore.AppendInt64Element(dst, "maxTimeMS", *fo.maxTimeMS)
	}
	if fo.max != nil {
		dst = bsoncore.AppendDocumentElement(dst, "max", fo.max)
	}
	if fo.min != nil {
		dst = bsoncore.AppendDocumentElement(dst, "min", fo.min)
	}
	if fo.returnKey != nil {
		dst = bsoncore.AppendBooleanElement(dst, "returnKey", *fo.returnKey)
	}
	if fo.showRecordID != nil {
		dst = bsoncore.AppendBooleanElement(dst, "showRecordID", *fo.showRecordID)
	}
	if fo.tailable != nil {
		dst = bsoncore.AppendBooleanElement(dst, "tailable", *fo.tailable)
	}
	if fo.oplogReplay != nil {
		dst = bsoncore.AppendBooleanElement(dst, "oplogReplay", *fo.oplogReplay)
	}
	if fo.noCursorTimeout != nil {
		dst = bsoncore.AppendBooleanElement(dst, "noCursorTimeout", *fo.noCursorTimeout)
	}
	if fo.awaitData != nil {
		dst = bsoncore.AppendBooleanElement(dst, "awaitData", *fo.awaitData)
	}
	if fo.allowPartialResults != nil {
		dst = bsoncore.AppendBooleanElement(dst, "allowPartialResults", *fo.allowPartialResults)
	}
	if fo.collation != nil {
		dst = bsoncore.AppendDocumentElement(dst, "collation", fo.collation)
	}
	dst, err := addReadConcern(dst, fo.readConcern, fo.client, desc)
	if err != nil {
		return dst, err
	}

	dst, err = addSession(dst, fo.client, desc)
	if err != nil {
		return dst, err
	}

	dst = addClusterTime(dst, fo.client, fo.clock, desc)

	dst = bsoncore.AppendStringElement(dst, "$db", fo.ns.DB)
	rp := createReadPref(fo.readPref, desc.Server.Kind, desc.Kind, false)
	if len(rp) > 0 {
		dst = bsoncore.AppendDocumentElement(dst, "$readPreference", rp)
	}

	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)

	return bsoncore.UpdateLength(dst, wmindex, int32(len(dst[wmindex:]))), nil
}

func (fo FindOperation) queryAppendCommand(dst []byte, desc description.SelectedServer) ([]byte, error) {
	idx, dst := bsoncore.AppendDocumentStart(dst)

	dst = bsoncore.AppendStringElement(dst, "find", fo.ns.Collection)

	if fo.filter != nil {
		dst = bsoncore.AppendDocumentElement(dst, "filter", fo.filter)
	}
	if fo.sort != nil {
		dst = bsoncore.AppendDocumentElement(dst, "sort", fo.sort)
	}
	if fo.projection != nil {
		dst = bsoncore.AppendDocumentElement(dst, "projection", fo.projection)
	}
	if fo.hint.Type != bsontype.Type(0) {
		dst = bsoncore.AppendHeader(dst, fo.hint.Type, "hint")
		dst = append(dst, fo.hint.Value...)
	}
	if fo.skip != nil {
		dst = bsoncore.AppendInt64Element(dst, "skip", *fo.skip)
	}
	if fo.limit != nil {
		dst = bsoncore.AppendInt64Element(dst, "limit", *fo.limit)
	}
	if fo.batchSize != nil {
		dst = bsoncore.AppendInt64Element(dst, "batchSize", *fo.batchSize)
	}
	if fo.singleBatch != nil {
		dst = bsoncore.AppendBooleanElement(dst, "singleBatch", *fo.singleBatch)
	}
	if fo.comment != nil {
		dst = bsoncore.AppendStringElement(dst, "comment", *fo.comment)
	}
	if fo.maxTimeMS != nil {
		dst = bsoncore.AppendInt64Element(dst, "maxTimeMS", *fo.maxTimeMS)
	}
	if fo.max != nil {
		dst = bsoncore.AppendDocumentElement(dst, "max", fo.max)
	}
	if fo.min != nil {
		dst = bsoncore.AppendDocumentElement(dst, "min", fo.min)
	}
	if fo.returnKey != nil {
		dst = bsoncore.AppendBooleanElement(dst, "returnKey", *fo.returnKey)
	}
	if fo.showRecordID != nil {
		dst = bsoncore.AppendBooleanElement(dst, "showRecordID", *fo.showRecordID)
	}
	if fo.tailable != nil {
		dst = bsoncore.AppendBooleanElement(dst, "tailable", *fo.tailable)
	}
	if fo.oplogReplay != nil {
		dst = bsoncore.AppendBooleanElement(dst, "oplogReplay", *fo.oplogReplay)
	}
	if fo.noCursorTimeout != nil {
		dst = bsoncore.AppendBooleanElement(dst, "noCursorTimeout", *fo.noCursorTimeout)
	}
	if fo.awaitData != nil {
		dst = bsoncore.AppendBooleanElement(dst, "awaitData", *fo.awaitData)
	}
	if fo.allowPartialResults != nil {
		dst = bsoncore.AppendBooleanElement(dst, "allowPartialResults", *fo.allowPartialResults)
	}
	if fo.collation != nil {
		dst = bsoncore.AppendDocumentElement(dst, "collation", fo.collation)
	}
	dst, err := addReadConcern(dst, fo.readConcern, fo.client, desc)
	if err != nil {
		return dst, err
	}

	dst, err = addSession(dst, fo.client, desc)
	if err != nil {
		return dst, err
	}

	dst = addClusterTime(dst, fo.client, fo.clock, desc)

	dst, _ = bsoncore.AppendDocumentEnd(dst, idx)
	return dst, nil
}
