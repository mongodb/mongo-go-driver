package command

import (
	"context"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// Write represents a generic write database command.
// This can be used to send arbitrary write commands to the database.
type Write struct {
	DB           string
	Command      *bson.Document
	WriteConcern *writeconcern.WriteConcern

	result bson.Reader
	err    error
}

// Encode c as OP_MSG
func (w *Write) encodeOpMsg(desc description.SelectedServer, cmd *bson.Document) (wiremessage.WireMessage, error) {
	var arr *bson.Array
	var identifier string

	arr, identifier = opmsgRemoveArray(cmd)

	msg := wiremessage.Msg{
		MsgHeader: wiremessage.Header{RequestID: wiremessage.NextRequestID()},
		Sections:  make([]wiremessage.Section, 0),
	}

	fullDocRdr, err := opmsgAddGlobals(cmd, w.DB, nil)
	if err != nil {
		return nil, err
	}

	// type 0 doc
	msg.Sections = append(msg.Sections, wiremessage.SectionBody{
		PayloadType: wiremessage.SingleDocument,
		Document:    fullDocRdr,
	})

	// type 1 doc
	if identifier != "" {
		docSequence, err := opmsgCreateDocSequence(arr, identifier)
		if err != nil {
			return nil, err
		}

		msg.Sections = append(msg.Sections, docSequence)
	}

	// flags
	if !writeconcern.AckWrite(w.WriteConcern) {
		msg.FlagBits |= wiremessage.MoreToCome
	}

	return msg, nil
}

// Encode w as OP_QUERY
func (w *Write) encodeOpQuery(desc description.SelectedServer, cmd *bson.Document) (wiremessage.WireMessage, error) {
	rdr, err := marshalCommand(cmd)
	if err != nil {
		return nil, err
	}

	query := wiremessage.Query{
		MsgHeader:          wiremessage.Header{RequestID: wiremessage.NextRequestID()},
		FullCollectionName: w.DB + ".$cmd",
		Flags:              w.slaveOK(desc),
		NumberToReturn:     -1,
		Query:              rdr,
	}

	return query, nil
}

func (w *Write) slaveOK(desc description.SelectedServer) wiremessage.QueryFlag {
	if desc.Kind == description.Single && desc.Server.Kind != description.Mongos {
		return wiremessage.SlaveOK
	}

	return 0
}

func (w *Write) decodeOpReply(wm wiremessage.WireMessage) *Write {
	reply, ok := wm.(wiremessage.Reply)
	if !ok {
		w.err = fmt.Errorf("unsupported response wiremessage type %T", wm)
		return w
	}
	w.result, w.err = decodeCommandOpReply(reply)
	return w
}

func (w *Write) decodeOpMsg(wm wiremessage.WireMessage) *Write {
	msg, ok := wm.(wiremessage.Msg)
	if !ok {
		w.err = fmt.Errorf("unsupported response wiremessage type %T", wm)
		return w
	}

	w.result, w.err = decodeCommandOpMsg(msg)
	return w
}

// Encode will encode this command into a wire message for the given server description.
func (w *Write) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	cmd := w.Command.Copy()
	err := addWriteConcern(cmd, w.WriteConcern)
	if err != nil {
		return nil, err
	}

	if desc.WireVersion == nil || desc.WireVersion.Max < wiremessage.OpmsgWireVersion {
		return w.encodeOpQuery(desc, cmd)
	}

	return w.encodeOpMsg(desc, cmd)
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (w *Write) Decode(desc description.SelectedServer, wm wiremessage.WireMessage) *Write {
	switch wm.(type) {
	case wiremessage.Reply:
		return w.decodeOpReply(wm)
	default:
		return w.decodeOpMsg(wm)
	}
}

// Result returns the result of a decoded wire message and server description.
func (w *Write) Result() (bson.Reader, error) {
	if w.err != nil {
		return nil, w.err
	}

	return w.result, nil
}

// Err returns the error set on this command.
func (w *Write) Err() error {
	return w.err
}

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriteCloser.
func (w *Write) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriter) (bson.Reader, error) {
	wm, err := w.Encode(desc)
	if err != nil {
		return nil, err
	}

	err = rw.WriteWireMessage(ctx, wm)
	if err != nil {
		return nil, err
	}

	if msg, ok := wm.(wiremessage.Msg); ok {
		// don't expect response if using OP_MSG for an unacknowledged write
		if msg.FlagBits&wiremessage.MoreToCome > 0 {
			return nil, ErrUnacknowledgedWrite
		}
	}

	wm, err = rw.ReadWireMessage(ctx)
	if err != nil {
		return nil, err
	}

	return w.Decode(desc, wm).Result()
}
