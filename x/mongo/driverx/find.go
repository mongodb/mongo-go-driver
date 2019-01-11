package driverx

import (
	"context"
	"errors"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"github.com/mongodb/mongo-go-driver/mongo/readconcern"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"github.com/mongodb/mongo-go-driver/x/bsonx/bsoncore"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver/session"
	"github.com/mongodb/mongo-go-driver/x/network/commandx"
	"github.com/mongodb/mongo-go-driver/x/network/description"
)

type FindOperation struct {
	filter              bson.Raw
	sort                bson.Raw
	projection          bson.Raw
	hint                bson.RawValue
	skip                *int64
	limit               *int64
	batchSize           *int64
	singleBatch         *bool
	comment             *string
	maxTimeMS           *int64
	readConcern         *readconcern.ReadConcern
	max                 bson.Raw
	min                 bson.Raw
	returnKey           *bool
	showRecordID        *bool
	tailable            *bool
	oplogReplay         *bool
	noCursorTimeout     *bool
	awaitData           *bool
	allowPartialResults *bool
	collation           bson.Raw

	serverSelector description.ServerSelector
	readPref       *readpref.ReadPref
	client         *session.Client
	clock          *session.ClusterClock
}

func Find(filter bson.Raw) FindOperation { return FindOperation{filter: filter} }

func (fo FindOperation) Filter(filter bson.Raw) FindOperation {
	fo.filter = filter
	return fo
}

func (fo FindOperation) Sort(sort bson.Raw) FindOperation {
	fo.sort = sort
	return fo
}

func (fo FindOperation) Projection(projection bson.Raw) FindOperation {
	fo.projection = projection
	return fo
}

func (fo FindOperation) Hint(hint bson.RawValue) FindOperation {
	fo.hint = hint
	return fo
}

func (fo FindOperation) Skip(skip int64) FindOperation {
	fo.skip = &skip
	return fo
}

func (fo FindOperation) Limit(limit int64) FindOperation {
	fo.limit = &limit
	return fo
}

func (fo FindOperation) BatchSize(batchSize int64) FindOperation {
	fo.batchSize = &batchSize
	return fo
}

func (fo FindOperation) SingleBatch(singleBatch bool) FindOperation {
	fo.singleBatch = &singleBatch
	return fo
}

func (fo FindOperation) Comment(comment string) FindOperation {
	fo.comment = &comment
	return fo
}

func (fo FindOperation) MaxTimeMS(maxTimeMS int64) FindOperation {
	fo.maxTimeMS = &maxTimeMS
	return fo
}

func (fo FindOperation) ReadConcern(rc *readconcern.ReadConcern) FindOperation {
	fo.readConcern = rc
	return fo
}

func (fo FindOperation) Max(max bson.Raw) FindOperation {
	fo.max = max
	return fo
}

func (fo FindOperation) Min(min bson.Raw) FindOperation {
	fo.min = min
	return fo
}

func (fo FindOperation) ReturnKey(returnKey bool) FindOperation {
	fo.returnKey = &returnKey
	return fo
}

func (fo FindOperation) ShowRecordID(showRecordID bool) FindOperation {
	fo.showRecordID = &showRecordID
	return fo
}

func (fo FindOperation) Tailable(tailable bool) FindOperation {
	fo.tailable = &tailable
	return fo
}

func (fo FindOperation) OplogReplay(oplogReplay bool) FindOperation {
	fo.oplogReplay = &oplogReplay
	return fo
}

func (fo FindOperation) NoCursorTimeout(noCursorTimeout bool) FindOperation {
	fo.noCursorTimeout = &noCursorTimeout
	return fo
}

func (fo FindOperation) AwaitData(awaitData bool) FindOperation {
	fo.awaitData = &awaitData
	return fo
}

func (fo FindOperation) AllowPartialResults(allowPartialResults bool) FindOperation {
	fo.allowPartialResults = &allowPartialResults
	return fo
}

func (fo FindOperation) Collation(collation bson.Raw) FindOperation {
	fo.collation = collation
	return fo
}

func (fo FindOperation) ServerSelector(ss description.ServerSelector) FindOperation {
	fo.serverSelector = ss
	return fo
}

func (fo FindOperation) Session(client *session.Client) FindOperation {
	fo.client = client
	return fo
}

func (fo FindOperation) ClusterClock(clock *session.ClusterClock) FindOperation {
	fo.clock = clock
	return fo
}

func (fo FindOperation) Execute(ctx context.Context, ns Namespace, d Deployment) (*BatchCursor, error) {
	selector := fo.serverSelector
	if selector == nil {
		rp := fo.readPref
		if rp == nil {
			rp = readpref.Primary()
		}
		selector = description.CompositeSelector([]description.ServerSelector{
			description.ReadPrefSelector(rp),
			description.LatencySelector(15 * time.Millisecond),
		})
	}

	srvr, err := d.SelectServer(ctx, selector)
	if err != nil {
		return nil, err
	}

	deploymentKind := d.Description().Kind

	conn, err := srvr.Connection(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	desc := conn.Description()

	if desc.WireVersion == nil || desc.WireVersion.Max < 4 {
		return nil, errors.New("legacy not currently supported")
	}

	cmd, err := fo.createCommand(ns, desc, deploymentKind)
	if err != nil {
		return nil, err
	}

	wm, err := cmd.MarshalWireMessage(nil, description.SelectedServer{Server: desc, Kind: deploymentKind})
	if err != nil {
		return nil, err
	}

	err = conn.WriteWireMessage(ctx, wm)
	if err != nil {
		return nil, err
	}

	wm, err = conn.ReadWireMessage(ctx, wm[:0])
	if err != nil {
		return nil, err
	}
	var cursor commandx.Cursor
	err = cursor.UnmarshalWireMessage(wm)
	if err != nil {
		return nil, err
	}

	spew.Dump(cursor)
	return nil, nil
}

func (fo FindOperation) createCommand(ns Namespace, desc description.Server, tKind description.TopologyKind) (commandx.Find, error) {
	cmd := commandx.Find{
		Collection: bsoncore.Value{Type: bsontype.String, Data: bsoncore.AppendString(nil, ns.Collection)},
		Database:   []byte(ns.DB),
	}

	// Add read preference
	opquery := (desc.WireVersion == nil || desc.WireVersion.Max < 4)
	rp := fo.createReadPref(desc.Kind, tKind, opquery)
	if rp != nil {
		cmd.ReadPref, _ = rp.MarshalBSON()
	}

	// Add read concern
	rc := fo.readConcern

	// Starting transaction's read concern overrides all others
	if fo.client != nil && fo.client.TransactionStarting() && fo.client.CurrentRc != nil {
		rc = fo.client.CurrentRc
	}

	// start transaction must append afterclustertime IF causally consistent and operation time exists
	if rc == nil && fo.client != nil && fo.client.TransactionStarting() && fo.client.Consistent && fo.client.OperationTime != nil {
		rc = readconcern.New()
	}

	if rc != nil {
		_, data, err := rc.MarshalBSONValue() // always returns a document
		if err != nil {
			return cmd, err
		}

		if description.SessionsSupported(desc.WireVersion) && fo.client != nil && fo.client.Consistent && fo.client.OperationTime != nil {
			data = data[:len(data)-1] // remove the null byte
			data = bsoncore.AppendTimestampElement(data, "afterClusterTime", fo.client.OperationTime.T, fo.client.OperationTime.I)
			data, _ = bsoncore.AppendDocumentEnd(data, 0)
		}

		cmd.ReadConcern = data
	}

	// Add session
	if fo.client != nil && description.SessionsSupported(desc.WireVersion) && desc.SessionTimeoutMinutes != 0 {
		if fo.client.Terminated {
			return cmd, session.ErrSessionEnded
		}
		cmd.LSID, _ = fo.client.SessionID.MarshalBSON()

		if fo.client.TransactionRunning() || fo.client.RetryingCommit {
			cmd.TxnNumber = bsoncore.Value{Type: bsontype.Int64, Data: bsoncore.AppendInt64(nil, fo.client.TxnNumber)}
			if fo.client.TransactionStarting() {
				cmd.StartTransaction = bsoncore.Value{Type: bsontype.Boolean, Data: bsoncore.AppendBoolean(nil, true)}
			}
			cmd.AutoCommit = bsoncore.Value{Type: bsontype.Boolean, Data: bsoncore.AppendBoolean(nil, false)}
		}

		fo.client.ApplyCommand()
	}

	// Add cluster time
	if (fo.clock != nil || fo.client != nil) && description.SessionsSupported(desc.WireVersion) {
		clusterTime := fo.clock.GetClusterTime()
		if fo.client != nil {
			clusterTime = session.MaxClusterTime(clusterTime, fo.client.ClusterTime)
		}
		if clusterTime != nil {
			cmd.ClusterTime = bsoncore.Document(clusterTime)
		}
	}

	if fo.filter != nil {
		cmd.Filter = bsoncore.Document(fo.filter)
	}
	if fo.sort != nil {
		cmd.Sort = bsoncore.Document(fo.sort)
	}
	if fo.projection != nil {
		cmd.Projection = bsoncore.Document(fo.projection)
	}
	if fo.hint.Type != bsontype.Type(0) {
		cmd.Hint = bsoncore.Value{Type: fo.hint.Type, Data: fo.hint.Value}
	}
	if fo.skip != nil {
		cmd.Skip = bsoncore.Value{Type: bsontype.Int64, Data: bsoncore.AppendInt64(nil, *fo.skip)}
	}
	if fo.limit != nil {
		cmd.Limit = bsoncore.Value{Type: bsontype.Int64, Data: bsoncore.AppendInt64(nil, *fo.limit)}
	}
	if fo.batchSize != nil {
		cmd.BatchSize = bsoncore.Value{Type: bsontype.Int64, Data: bsoncore.AppendInt64(nil, *fo.batchSize)}
	}
	if fo.singleBatch != nil {
		cmd.SingleBatch = bsoncore.Value{Type: bsontype.Boolean, Data: bsoncore.AppendBoolean(nil, *fo.singleBatch)}
	}
	if fo.comment != nil {
		cmd.Comment = bsoncore.Value{Type: bsontype.String, Data: bsoncore.AppendString(nil, *fo.comment)}
	}
	if fo.maxTimeMS != nil {
		cmd.MaxTimeMS = bsoncore.Value{Type: bsontype.Int64, Data: bsoncore.AppendInt64(nil, *fo.maxTimeMS)}
	}
	if fo.max != nil {
		cmd.Max = bsoncore.Document(fo.max)
	}
	if fo.min != nil {
		cmd.Min = bsoncore.Document(fo.min)
	}
	if fo.returnKey != nil {
		cmd.ReturnKey = bsoncore.Value{Type: bsontype.Boolean, Data: bsoncore.AppendBoolean(nil, *fo.returnKey)}
	}
	if fo.showRecordID != nil {
		cmd.ShowRecordID = bsoncore.Value{Type: bsontype.Boolean, Data: bsoncore.AppendBoolean(nil, *fo.showRecordID)}
	}
	if fo.tailable != nil {
		cmd.Tailable = bsoncore.Value{Type: bsontype.Boolean, Data: bsoncore.AppendBoolean(nil, *fo.tailable)}
	}
	if fo.oplogReplay != nil {
		cmd.OplogReplay = bsoncore.Value{Type: bsontype.Boolean, Data: bsoncore.AppendBoolean(nil, *fo.oplogReplay)}
	}
	if fo.noCursorTimeout != nil {
		cmd.NoCursorTimeout = bsoncore.Value{Type: bsontype.Boolean, Data: bsoncore.AppendBoolean(nil, *fo.noCursorTimeout)}
	}
	if fo.awaitData != nil {
		cmd.AwaitData = bsoncore.Value{Type: bsontype.Boolean, Data: bsoncore.AppendBoolean(nil, *fo.awaitData)}
	}
	if fo.allowPartialResults != nil {
		cmd.AllowPartialResults = bsoncore.Value{Type: bsontype.Boolean, Data: bsoncore.AppendBoolean(nil, *fo.allowPartialResults)}
	}
	if fo.collation != nil {
		cmd.Collation = bsoncore.Document(fo.collation)
	}

	return cmd, nil
}

// TODO(GODRIVER-617): Remove dependency on bsonx.
func (fo FindOperation) createReadPref(serverKind description.ServerKind, topologyKind description.TopologyKind, isOpQuery bool) bsonx.Doc {
	doc := bsonx.Doc{}
	rp := fo.readPref

	if rp == nil {
		if topologyKind == description.Single && serverKind != description.Mongos {
			return append(doc, bsonx.Elem{"mode", bsonx.String("primaryPreferred")})
		}
		return nil
	}

	switch rp.Mode() {
	case readpref.PrimaryMode:
		if serverKind == description.Mongos {
			return nil
		}
		if topologyKind == description.Single {
			return append(doc, bsonx.Elem{"mode", bsonx.String("primaryPreferred")})
		}
		doc = append(doc, bsonx.Elem{"mode", bsonx.String("primary")})
	case readpref.PrimaryPreferredMode:
		doc = append(doc, bsonx.Elem{"mode", bsonx.String("primaryPreferred")})
	case readpref.SecondaryPreferredMode:
		_, ok := rp.MaxStaleness()
		if serverKind == description.Mongos && isOpQuery && !ok && len(rp.TagSets()) == 0 {
			return nil
		}
		doc = append(doc, bsonx.Elem{"mode", bsonx.String("secondaryPreferred")})
	case readpref.SecondaryMode:
		doc = append(doc, bsonx.Elem{"mode", bsonx.String("secondary")})
	case readpref.NearestMode:
		doc = append(doc, bsonx.Elem{"mode", bsonx.String("nearest")})
	}

	sets := make([]bsonx.Val, 0, len(rp.TagSets()))
	for _, ts := range rp.TagSets() {
		if len(ts) == 0 {
			continue
		}
		set := bsonx.Doc{}
		for _, t := range ts {
			set = append(set, bsonx.Elem{t.Name, bsonx.String(t.Value)})
		}
		sets = append(sets, bsonx.Document(set))
	}
	if len(sets) > 0 {
		doc = append(doc, bsonx.Elem{"tags", bsonx.Array(sets)})
	}

	if d, ok := rp.MaxStaleness(); ok {
		doc = append(doc, bsonx.Elem{"maxStalenessSeconds", bsonx.Int32(int32(d.Seconds()))})
	}

	return doc
}
