package server

import (
	"fmt"
	"time"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal"

	"gopkg.in/mgo.v2/bson"
)

// UnsetRTT is the unset value for a round trip time.
const UnsetRTT = -1 * time.Millisecond

// Desc is a description of a server.
type Desc struct {
	Endpoint conn.Endpoint

	AverageRTT        time.Duration
	AverageRTTSet     bool
	CanonicalEndpoint conn.Endpoint
	ElectionID        bson.ObjectId
	HeartbeatInterval time.Duration
	LastError         error
	LastUpdateTime    time.Time
	LastWriteTime     time.Time
	MaxBatchCount     uint16
	MaxDocumentSize   uint32
	MaxMessageSize    uint32
	Members           []conn.Endpoint
	SetName           string
	SetVersion        uint32
	GitVersion        string
	Tags              TagSet
	Type              Type
	WireVersion       conn.Range
	Version           conn.Version
}

// SetAverageRTT sets the average rount trip time.
func (d *Desc) SetAverageRTT(rtt time.Duration) {
	d.AverageRTT = rtt
	if rtt == UnsetRTT {
		d.AverageRTTSet = false
	} else {
		d.AverageRTTSet = true
	}
}

// BuildDesc builds a server.Desc from an endpoint, IsMasterResult, and a BuildInfoResult.
func BuildDesc(endpoint conn.Endpoint, isMasterResult *internal.IsMasterResult, buildInfoResult *internal.BuildInfoResult) *Desc {
	s := &Desc{
		Endpoint: endpoint,

		CanonicalEndpoint: conn.Endpoint(isMasterResult.Me),
		ElectionID:        isMasterResult.ElectionID,
		GitVersion:        buildInfoResult.GitVersion,
		LastUpdateTime:    time.Now().UTC(),
		LastWriteTime:     isMasterResult.LastWriteTimestamp,
		MaxBatchCount:     isMasterResult.MaxWriteBatchSize,
		MaxDocumentSize:   isMasterResult.MaxBSONObjectSize,
		MaxMessageSize:    isMasterResult.MaxMessageSizeBytes,
		SetName:           isMasterResult.SetName,
		SetVersion:        isMasterResult.SetVersion,
		Tags:              nil, // TODO: get tags
		Version: conn.Version{
			Desc:  buildInfoResult.Version,
			Parts: buildInfoResult.VersionArray,
		},
		WireVersion: conn.Range{
			Min: isMasterResult.MinWireVersion,
			Max: isMasterResult.MaxWireVersion,
		},
	}

	if s.CanonicalEndpoint == "" {
		s.CanonicalEndpoint = endpoint
	}

	if !isMasterResult.OK {
		s.LastError = fmt.Errorf("not ok")
		return s
	}

	for _, host := range isMasterResult.Hosts {
		s.Members = append(s.Members, conn.Endpoint(host).Canonicalize())
	}

	for _, passive := range isMasterResult.Passives {
		s.Members = append(s.Members, conn.Endpoint(passive).Canonicalize())
	}

	for _, arbiter := range isMasterResult.Arbiters {
		s.Members = append(s.Members, conn.Endpoint(arbiter).Canonicalize())
	}

	s.Type = Standalone

	if isMasterResult.IsReplicaSet {
		s.Type = RSGhost
	} else if isMasterResult.SetName != "" {
		if isMasterResult.IsMaster {
			s.Type = RSPrimary
		} else if isMasterResult.Hidden {
			s.Type = RSMember
		} else if isMasterResult.Secondary {
			s.Type = RSSecondary
		} else if isMasterResult.ArbiterOnly {
			s.Type = RSArbiter
		} else {
			s.Type = RSMember
		}
	} else if isMasterResult.Msg == "isdbgrid" {
		s.Type = Mongos
	}

	return s
}
