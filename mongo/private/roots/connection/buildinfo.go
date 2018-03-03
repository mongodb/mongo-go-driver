package connection

import (
	"context"
	"errors"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// CommandBuildInfo represents the buildInfo command.
//
// The buildInfo command is used for getting the build information for a
// MongoDB server.
//
// Since CommandBuildInfo can only be run on a connection, there is no Dispatch method.
type CommandBuildInfo struct {
	err error
	res result.BuildInfo
}

// Encode will encode this command into a wire message for the given server description.
func (bi *CommandBuildInfo) Encode() (wiremessage.WireMessage, error) {
	// This can probably just be a global variable that we reuse.
	cmd := bson.NewDocument(bson.EC.Int32("buildInfo", 1))
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
func (bi *CommandBuildInfo) Decode(wm wiremessage.WireMessage) *CommandBuildInfo {
	reply, ok := wm.(wiremessage.Reply)
	if !ok {
		bi.err = errors.New(fmt.Sprintf("unsupported response wiremessage type %T", wm))
		return bi
	}
	rdr, err := decodeOpReply(reply)
	if err != nil {
		bi.err = err
		return bi
	}
	err = bson.Unmarshal(rdr, &bi.res)
	if err != nil {
		bi.err = err
		return bi
	}
	return bi
}

// Result returns the result of a decoded wire message and server description.
func (bi *CommandBuildInfo) Result() (result.BuildInfo, error) {
	if bi.err != nil {
		return result.BuildInfo{}, bi.err
	}

	return bi.res, nil
}

// Err returns the error set on this command.
func (bi *CommandBuildInfo) Err() error { return bi.err }

// RoundTrip handles the execution of this command using the provided connection.
func (bi *CommandBuildInfo) RoundTrip(ctx context.Context, c Connection) (result.BuildInfo, error) {
	wm, err := bi.Encode()
	if err != nil {
		return result.BuildInfo{}, err
	}

	err = c.WriteWireMessage(ctx, wm)
	if err != nil {
		return result.BuildInfo{}, err
	}
	wm, err = c.ReadWireMessage(ctx)
	if err != nil {
		return result.BuildInfo{}, err
	}
	return bi.Decode(wm).Result()
}
