package desc

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

// UnsetRTT is the unset value for a round trip time.
const UnsetRTT = -1 * time.Millisecond

// Server is a description of a server.
type Server struct {
	Endpoint Endpoint

	CanonicalEndpoint Endpoint
	ElectionID        bson.ObjectId
	HeartbeatInterval time.Duration
	LastError         error
	LastUpdateTime    time.Time
	LastWriteTime     time.Time
	MaxBatchCount     uint16
	MaxDocumentSize   uint32
	MaxMessageSize    uint32
	Members           []Endpoint
	SetName           string
	SetVersion        uint32
	Tags              TagSet
	Type              ServerType
	WireVersion       Range
	Version           Version

	averageRTT    time.Duration
	averageRTTSet bool
}

// AverageRTT is the average round trip time for the server. If
// the second return value is false, it means that the average
// round trip time has not been set.
func (d *Server) AverageRTT() (time.Duration, bool) {
	return d.averageRTT, d.averageRTTSet
}

// SetAverageRTT sets the average rount trip time.
func (d *Server) SetAverageRTT(rtt time.Duration) {
	d.averageRTT = rtt
	if rtt == UnsetRTT {
		d.averageRTTSet = false
	} else {
		d.averageRTTSet = true
	}
}

// ServerType represents a type of server.
type ServerType uint32

// ServerType constants.
const (
	UnknownServerType ServerType = 0
	Standalone        ServerType = 1
	RSMember          ServerType = 2
	RSPrimary         ServerType = 4 + RSMember
	RSSecondary       ServerType = 8 + RSMember
	RSArbiter         ServerType = 16 + RSMember
	RSGhost           ServerType = 32 + RSMember
	Mongos            ServerType = 256
)
