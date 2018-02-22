package command

import (
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/wiremessage"
)

// BuildInfo represents the buildInfo command.
//
// The buildInfo command is used for getting the build information for a
// MongoDB server.
type BuildInfo struct{}

// NewBuildInfo creates a buildInfo command.
func NewBuildInfo() (BuildInfo, error) { return BuildInfo{}, nil }

// Encode will encode this command into a wire message for the given server description.
func (bi BuildInfo) Encode() (wiremessage.WireMessage, error) { return nil, nil }

// Decode will decode the wire message using the provided server description into a result.
func (bi BuildInfo) Decode(wiremessage.WireMessage) (result.BuildInfo, error) {
	return result.BuildInfo{}, nil
}
