package connection

import (
	"context"
	"errors"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// CommandIsMaster represents the isMaster command.
//
// The isMaster command is used for setting up a connection to MongoDB and
// for monitoring a MongoDB server.
//
// Since CommandIsMaster can only be run on a connection, there is no Dispatch method.
type CommandIsMaster struct {
	Client *bson.Document

	err error
	res result.IsMaster
}

// Encode will encode this command into a wire message for the given server description.
func (im *CommandIsMaster) Encode() (wiremessage.WireMessage, error) {
	cmd := bson.NewDocument(bson.EC.Int32("isMaster", 1))
	if im.Client != nil {
		cmd.Append(bson.EC.SubDocument("client", im.Client))
	}
	rdr, err := cmd.MarshalBSON()
	if err != nil {
		return nil, err
	}
	query := wiremessage.Query{
		MsgHeader:          wiremessage.Header{RequestID: wiremessage.NextRequestID()},
		FullCollectionName: "admin.$cmd",
		Flags:              wiremessage.SlaveOK,
		NumberToReturn:     -1,
		Query:              rdr,
	}
	return query, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (im *CommandIsMaster) Decode(wm wiremessage.WireMessage) *CommandIsMaster {
	reply, ok := wm.(wiremessage.Reply)
	if !ok {
		im.err = errors.New(fmt.Sprintf("unsupported response wiremessage type %T", wm))
		return im
	}
	rdr, err := decodeOpReply(reply)
	if err != nil {
		im.err = err
		return im
	}
	err = bson.Unmarshal(rdr, &im.res)
	if err != nil {
		im.err = err
		return im
	}
	return im
}

// Result returns the result of a decoded wire message and server description.
func (im *CommandIsMaster) Result() (result.IsMaster, error) {
	if im.err != nil {
		return result.IsMaster{}, im.err
	}

	return im.res, nil
}

// Err returns the error set on this command.
func (im *CommandIsMaster) Err() error { return im.err }

// RoundTrip handles the execution of this command using the provided connection.
func (im *CommandIsMaster) RoundTrip(ctx context.Context, c Connection) (result.IsMaster, error) {
	wm, err := im.Encode()
	if err != nil {
		return result.IsMaster{}, err
	}

	err = c.WriteWireMessage(ctx, wm)
	if err != nil {
		return result.IsMaster{}, err
	}
	wm, err = c.ReadWireMessage(ctx)
	if err != nil {
		return result.IsMaster{}, err
	}
	return im.Decode(wm).Result()
}
