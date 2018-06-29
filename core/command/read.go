package command

import (
	"context"

	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// Read represents a generic database read command.
type Read struct {
	DB          string
	Command     *bson.Document
	ReadPref    *readpref.ReadPref
	ReadConcern *readconcern.ReadConcern
	Clock       *session.ClusterClock
	Session     *session.Client

	result bson.Reader
	err    error
}

func (r *Read) createReadPref(kind description.ServerKind) *bson.Document {
	if r.ReadPref == nil {
		return nil
	}

	doc := bson.NewDocument()

	switch r.ReadPref.Mode() {
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

	sets := make([]*bson.Value, 0, len(r.ReadPref.TagSets()))
	for _, ts := range r.ReadPref.TagSets() {
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

	if d, ok := r.ReadPref.MaxStaleness(); ok {
		doc.Append(bson.EC.Int32("maxStalenessSeconds", int32(d.Seconds())))
	}

	return doc
}

// addReadPref will add a read preference to the query document.
//
// NOTE: This method must always return either a valid bson.Reader or an error.
func (r *Read) addReadPref(rp *readpref.ReadPref, kind description.ServerKind, query bson.Reader) (bson.Reader, error) {
	doc := r.createReadPref(kind)
	if doc == nil {
		return query, nil
	}

	return bson.NewDocument(
		bson.EC.SubDocumentFromReader("$query", query),
		bson.EC.SubDocument("$readPreference", doc),
	).MarshalBSON()
}

// Encode r as OP_MSG
func (r *Read) encodeOpMsg(desc description.SelectedServer, cmd *bson.Document) (wiremessage.WireMessage, error) {
	msg := wiremessage.Msg{
		MsgHeader: wiremessage.Header{RequestID: wiremessage.NextRequestID()},
		Sections:  make([]wiremessage.Section, 0),
	}

	readPrefDoc := r.createReadPref(desc.Server.Kind)
	fullDocRdr, err := opmsgAddGlobals(cmd, r.DB, readPrefDoc)
	if err != nil {
		return nil, err
	}

	// type 0 doc
	msg.Sections = append(msg.Sections, wiremessage.SectionBody{
		PayloadType: wiremessage.SingleDocument,
		Document:    fullDocRdr,
	})

	// no flags to add

	return msg, nil
}

func (r *Read) slaveOK(desc description.SelectedServer) wiremessage.QueryFlag {
	if desc.Kind == description.Single && desc.Server.Kind != description.Mongos {
		return wiremessage.SlaveOK
	}

	if r.ReadPref == nil {
		// assume primary
		return 0
	}

	if r.ReadPref.Mode() != readpref.PrimaryMode {
		return wiremessage.SlaveOK
	}

	return 0
}

// return true if a read preference needs to be added when encoding r as a OP_QUERY message
func (r *Read) queryNeedsReadPref(kind description.ServerKind) bool {
	if kind != description.Mongos || r.ReadPref == nil {
		return false
	}

	// simple Primary or SecondaryPreferred is communicated via slaveOk to Mongos.
	if r.ReadPref.Mode() == readpref.PrimaryMode || r.ReadPref.Mode() == readpref.SecondaryPreferredMode {
		if _, ok := r.ReadPref.MaxStaleness(); !ok && len(r.ReadPref.TagSets()) == 0 {
			return false
		}
	}

	return true
}

// Encode c as OP_QUERY
func (r *Read) encodeOpQuery(desc description.SelectedServer, cmd *bson.Document) (wiremessage.WireMessage, error) {
	rdr, err := marshalCommand(cmd)
	if err != nil {
		return nil, err
	}

	if r.queryNeedsReadPref(desc.Server.Kind) {
		rdr, err = r.addReadPref(r.ReadPref, desc.Server.Kind, rdr)
		if err != nil {
			return nil, err
		}
	}

	query := wiremessage.Query{
		MsgHeader:          wiremessage.Header{RequestID: wiremessage.NextRequestID()},
		FullCollectionName: r.DB + ".$cmd",
		Flags:              r.slaveOK(desc),
		NumberToReturn:     -1,
		Query:              rdr,
	}

	return query, nil
}

func (r *Read) decodeOpMsg(wm wiremessage.WireMessage) {
	msg, ok := wm.(wiremessage.Msg)
	if !ok {
		r.err = fmt.Errorf("unsupported response wiremessage type %T", wm)
		return
	}

	r.result, r.err = decodeCommandOpMsg(msg)
}

func (r *Read) decodeOpReply(wm wiremessage.WireMessage) {
	reply, ok := wm.(wiremessage.Reply)
	if !ok {
		r.err = fmt.Errorf("unsupported response wiremessage type %T", wm)
		return
	}
	r.result, r.err = decodeCommandOpReply(reply)
}

// Encode will encode this command into a wire message for the given server description.
func (r *Read) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	cmd := r.Command.Copy()
	err := addReadConcern(cmd, r.ReadConcern)
	if err != nil {
		return nil, err
	}

	err = addSessionID(cmd, desc, r.Session)
	if err != nil {
		return nil, err
	}

	err = addClusterTime(cmd, desc, r.Session, r.Clock)
	if err != nil {
		return nil, nil
	}

	if desc.WireVersion == nil || desc.WireVersion.Max < wiremessage.OpmsgWireVersion {
		return r.encodeOpQuery(desc, cmd)
	}

	return r.encodeOpMsg(desc, cmd)
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (r *Read) Decode(desc description.SelectedServer, wm wiremessage.WireMessage) *Read {
	switch wm.(type) {
	case wiremessage.Reply:
		r.decodeOpReply(wm)
	default:
		r.decodeOpMsg(wm)
	}

	if r.err != nil {
		// decode functions set error if an invalid response document was returned or if the OK flag in the response was 0
		// if the OK flag was 0, a type Error is returned. otherwise, a special type is returned
		if _, ok := r.err.(Error); !ok {
			return r // for missing/invalid response docs, don't update cluster times
		}
	}

	_ = updateClusterTimes(r.Session, r.Clock, r.result)

	return r
}

// Result returns the result of a decoded wire message and server description.
func (r *Read) Result() (bson.Reader, error) {
	if r.err != nil {
		return nil, r.err
	}

	return r.result, nil
}

// Err returns the error set on this command.
func (r *Read) Err() error {
	return r.err
}

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (r *Read) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriter) (bson.Reader, error) {
	wm, err := r.Encode(desc)
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

	if r.Session != nil {
		err = r.Session.UpdateUseTime()
		if err != nil {
			return nil, err
		}
	}
	return r.Decode(desc, wm).Result()
}
