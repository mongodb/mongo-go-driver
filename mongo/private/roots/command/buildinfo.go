package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/mongo/private/roots/connection"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/topology"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// BuildInfo represents the buildInfo command.
//
// The buildInfo command is used for getting the build information for a
// MongoDB server.
//
// Since BuildInfo can only be run on a connection, there is no Dispatch method.
type BuildInfo struct{}

// Encode will encode this command into a wire message for the given server description.
func (bi *BuildInfo) Encode() (wiremessage.WireMessage, error) { return nil, nil }

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (bi *BuildInfo) Decode(wiremessage.WireMessage) *BuildInfo {
	return nil
}

// Result returns the result of a decoded wire message and server description.
func (bi *BuildInfo) Result() (result.BuildInfo, error) { return result.BuildInfo{}, nil }

// Err returns the error set on this command.
func (bi *BuildInfo) Err() error { return nil }

// RoundTrip handles the execution of this command using the provided connection.
func (bi *BuildInfo) RoundTrip(context.Context, topology.ServerDescription, connection.Connection) (result.BuildInfo, error) {
	return result.BuildInfo{}, nil
}
