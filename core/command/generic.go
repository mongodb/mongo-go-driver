package command

import (
	"context"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// GenericRequest represents a generic command.
//
// GenericRequest maintains semantic roundtripability, but does not maintain
// byte roundtripability. This means that GenericRequest may rearrange fields
// within the command and may encode itself to different wire message type. The
// meaning of the command, however, will be maintained.
type GenericRequest struct {
	DB           string
	Name         string
	Command      *bson.Document
	ReadPref     *readpref.ReadPref
	ReadConcern  *readconcern.ReadConcern
	WriteConcern *writeconcern.WriteConcern

	result bson.Reader
	err    error
}

// Encode encodes this request.
func (g *GenericRequest) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	cmd := g.Command.Copy()
	if err := addWriteConcern(cmd, g.WriteConcern); err != nil {
		return nil, err
	}
	if err := addReadConcern(cmd, g.ReadConcern); err != nil {
		return nil, err
	}

	if desc.WireVersion == nil || desc.WireVersion.Max < wiremessage.OpmsgWireVersion {
		return g.encodeOpQuery(desc, cmd)
	}

	return g.encodeOpMsg(desc, cmd)
}

// Decode decodes this request.
func (g *GenericRequest) Decode(wm wiremessage.WireMessage) *GenericRequest {
	switch t := wm.(type) {
	case wiremessage.Reply:
		return g.decodeOpReply(t)
	case wiremessage.Msg:
		return g.decodeOpMsg(t)
	case wiremessage.Query:
		return g.decodeOpQuery(t)
	default:
		g.err = fmt.Errorf("%T cannot decode a wiremessage.WireMessage of type %T", g, t)
	}
	return g
}

// Result returns the result of the response to this request.
func (g *GenericRequest) Result() (bson.Reader, error) {
	if g.err != nil {
		return nil, g.err
	}

	return g.result, nil
}

// Err returns the error from this request.
func (g *GenericRequest) Err() error { return g.err }

// RoundTrip handles roundtripping this request and decoding the response.
func (g *GenericRequest) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriter) (bson.Reader, error) {
	wm, err := g.Encode(desc)
	if err != nil {
		return nil, err
	}

	err = rw.WriteWireMessage(ctx, wm)
	if err != nil {
		return nil, err
	}
	wm, err = rw.ReadWireMessage(ctx)
	if err != nil {
		return nil, err
	}

	return g.Decode(wm).Result()
}

func (g *GenericRequest) encodeOpMsg(desc description.SelectedServer, cmd *bson.Document) (wiremessage.Msg, error) {
	arr, identifier := opmsgRemoveArray(cmd)
	msg := wiremessage.Msg{
		MsgHeader: wiremessage.Header{RequestID: wiremessage.NextRequestID()},
		Sections:  make([]wiremessage.Section, 0),
	}
	readPrefDoc := g.createReadPref(desc.Server.Kind)
	fulldoc, err := opmsgAddGlobals(cmd, g.DB, readPrefDoc)
	if err != nil {
		return wiremessage.Msg{}, err
	}

	// type 0 doc
	msg.Sections = append(msg.Sections, wiremessage.SectionBody{
		PayloadType: wiremessage.SingleDocument,
		Document:    fulldoc,
	})

	// type 1 doc
	if identifier != "" {
		docsequence, err := opmsgCreateDocSequence(arr, identifier)
		if err != nil {
			return wiremessage.Msg{}, err
		}

		msg.Sections = append(msg.Sections, docsequence)
	}

	// flags
	if !writeconcern.AckWrite(g.WriteConcern) {
		msg.FlagBits |= wiremessage.MoreToCome // We only set moreToCome if we aren't reencoding.
	}

	return msg, nil
}

func (g *GenericRequest) encodeOpQuery(desc description.SelectedServer, cmd *bson.Document) (wiremessage.Query, error) {
	rdr, err := marshalCommand(cmd)
	if err != nil {
		return wiremessage.Query{}, err
	}

	if g.queryNeedsReadPref(desc.Server.Kind) {
		rdr, err = g.addReadPref(g.ReadPref, desc.Server.Kind, rdr)
		if err != nil {
			return wiremessage.Query{}, err
		}
	}

	query := wiremessage.Query{MsgHeader: wiremessage.Header{RequestID: wiremessage.NextRequestID()}}

	query.FullCollectionName = g.DB + "$cmd"
	query.Flags = g.slaveOK(desc)
	query.NumberToReturn = -1
	query.Query = rdr

	return query, nil
}

func (g *GenericRequest) decodeOpMsg(wm wiremessage.WireMessage) *GenericRequest {
	return g
}

func (g *GenericRequest) decodeOpQuery(wm wiremessage.WireMessage) *GenericRequest {
	return g
}

func (g *GenericRequest) decodeOpReply(wm wiremessage.WireMessage) *GenericRequest {
	return g
}

func (g *GenericRequest) slaveOK(desc description.SelectedServer) wiremessage.QueryFlag {
	if desc.Kind == description.Single && desc.Server.Kind != description.Mongos {
		return wiremessage.SlaveOK
	}

	if g.ReadPref == nil {
		// assume primary
		return 0
	}

	if g.ReadPref.Mode() != readpref.PrimaryMode {
		return wiremessage.SlaveOK
	}

	return 0
}

func (g *GenericRequest) createReadPref(kind description.ServerKind) *bson.Document {
	if g.ReadPref == nil {
		return nil
	}

	doc := bson.NewDocument()

	switch g.ReadPref.Mode() {
	case readpref.PrimaryMode:
		doc.Append(bson.EC.String("mode", "primary"))
	case readpref.PrimaryPreferredMode:
		doc.Append(bson.EC.String("mode", "primaryPreferred"))
	case readpref.SecondaryPreferredMode:
		doc.Append(bson.EC.String("mode", "secondaryPreferred"))
	case readpref.SecondaryMode:
		doc.Append(bson.EC.String("mode", "secondary"))
	case readpref.NearestMode:
		doc.Append(bson.EC.String("mode", "nearest"))
	}

	sets := make([]*bson.Value, 0, len(g.ReadPref.TagSets()))
	for _, ts := range g.ReadPref.TagSets() {
		if len(ts) == 0 {
			continue
		}
		set := bson.NewDocument()
		for _, t := range ts {
			set.Append(bson.EC.String(t.Name, t.Value))
		}
		sets = append(sets, bson.VC.Document(set))
	}
	if len(sets) > 0 {
		doc.Append(bson.EC.ArrayFromElements("tags", sets...))
	}

	if d, ok := g.ReadPref.MaxStaleness(); ok {
		doc.Append(bson.EC.Int32("maxStalenessSeconds", int32(d.Seconds())))
	}

	return doc
}

func (g *GenericRequest) queryNeedsReadPref(kind description.ServerKind) bool {
	if kind != description.Mongos || g.ReadPref == nil {
		return false
	}

	// simple Primary or SecondaryPreferred is communicated via slaveOk to Mongos.
	if g.ReadPref.Mode() == readpref.PrimaryMode || g.ReadPref.Mode() == readpref.SecondaryPreferredMode {
		if _, ok := g.ReadPref.MaxStaleness(); !ok && len(g.ReadPref.TagSets()) == 0 {
			return false
		}
	}

	return true
}

func (g *GenericRequest) addReadPref(rp *readpref.ReadPref, kind description.ServerKind, query bson.Reader) (bson.Reader, error) {
	doc := g.createReadPref(kind)
	if doc == nil {
		return query, nil
	}

	return bson.NewDocument(
		bson.EC.SubDocumentFromReader("$query", query),
		bson.EC.SubDocument("$readPreference", doc),
	).MarshalBSON()
}
