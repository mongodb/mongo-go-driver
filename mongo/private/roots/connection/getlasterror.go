package connection

import (
	"context"
	"errors"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// CommandGetLastError represents the getLastError command.
//
// The getLastError command is used for getting the last
// error from the last command on a connection.
//
// Since CommandGetLastError only makes sense in the context of
// a single connection, there is no Dispatch method.
type CommandGetLastError struct {
	err error
	res result.GetLastError
}

// Encode will encode this command into a wire message for the given server description.
func (gle *CommandGetLastError) Encode() (wiremessage.WireMessage, error) {
	// This can probably just be a global variable that we reuse.
	cmd := bson.NewDocument(bson.EC.Int32("getLastError", 1))
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
func (gle *CommandGetLastError) Decode(wm wiremessage.WireMessage) *CommandGetLastError {
	reply, ok := wm.(wiremessage.Reply)
	if !ok {
		gle.err = errors.New(fmt.Sprintf("unsupported response wiremessage type %T", wm))
		return gle
	}
	rdr, err := decodeOpReply(reply)
	if err != nil {
		gle.err = err
		return gle
	}
	err = bson.Unmarshal(rdr, &gle.res)
	if err != nil {
		gle.err = err
		return gle
	}
	return gle
}

// Result returns the result of a decoded wire message and server description.
func (gle *CommandGetLastError) Result() (result.GetLastError, error) {
	if gle.err != nil {
		return result.GetLastError{}, gle.err
	}

	return gle.res, nil
}

// Err returns the error set on this command.
func (gle *CommandGetLastError) Err() error { return gle.err }

// RoundTrip handles the execution of this command using the provided connection.
func (gle *CommandGetLastError) RoundTrip(ctx context.Context, c Connection) (result.GetLastError, error) {
	wm, err := gle.Encode()
	if err != nil {
		return result.GetLastError{}, err
	}

	err = c.WriteWireMessage(ctx, wm)
	if err != nil {
		return result.GetLastError{}, err
	}
	wm, err = c.ReadWireMessage(ctx)
	if err != nil {
		return result.GetLastError{}, err
	}
	return gle.Decode(wm).Result()
}
