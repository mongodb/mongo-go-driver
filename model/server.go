package model

import (
	"fmt"
	"time"

	"github.com/10gen/mongo-go-driver/internal"

	"github.com/10gen/mongo-go-driver/bson"
)

// UnsetRTT is the unset value for a round trip time.
const UnsetRTT = -1 * time.Millisecond

// Server is a description of a server.
type Server struct {
	Addr Addr

	AverageRTT        time.Duration
	AverageRTTSet     bool
	CanonicalAddr     Addr
	ElectionID        bson.ObjectId
	GitVersion        string
	HeartbeatInterval time.Duration
	LastError         error
	LastUpdateTime    time.Time
	LastWriteTime     time.Time
	MaxBatchCount     uint16
	MaxDocumentSize   uint32
	MaxMessageSize    uint32
	Members           []Addr
	ReadOnly          bool
	SetName           string
	SetVersion        uint32
	Tags              TagSet
	Kind              ServerKind
	WireVersion       Range
	Version           Version
}

// SetAverageRTT sets the average rount trip time.
func (i *Server) SetAverageRTT(rtt time.Duration) {
	i.AverageRTT = rtt
	if rtt == UnsetRTT {
		i.AverageRTTSet = false
	} else {
		i.AverageRTTSet = true
	}
}

// BuildServer builds a server.Desc from an endpoint, IsMasterResult, and a BuildInfoResult.
func BuildServer(addr Addr, isMasterResult *internal.IsMasterResult, buildInfoResult *internal.BuildInfoResult) *Server {
	i := &Server{
		Addr: addr,

		CanonicalAddr:   Addr(isMasterResult.Me).Canonicalize(),
		ElectionID:      isMasterResult.ElectionID,
		GitVersion:      buildInfoResult.GitVersion,
		LastUpdateTime:  time.Now().UTC(),
		LastWriteTime:   isMasterResult.LastWriteTimestamp,
		MaxBatchCount:   isMasterResult.MaxWriteBatchSize,
		MaxDocumentSize: isMasterResult.MaxBSONObjectSize,
		MaxMessageSize:  isMasterResult.MaxMessageSizeBytes,
		SetName:         isMasterResult.SetName,
		SetVersion:      isMasterResult.SetVersion,
		Tags:            NewTagSetFromMap(isMasterResult.Tags),
		Version: Version{
			Desc:  buildInfoResult.Version,
			Parts: buildInfoResult.VersionArray,
		},
		WireVersion: Range{
			Min: isMasterResult.MinWireVersion,
			Max: isMasterResult.MaxWireVersion,
		},
	}

	if i.CanonicalAddr == "" {
		i.CanonicalAddr = addr
	}

	if !isMasterResult.OK {
		i.LastError = fmt.Errorf("not ok")
		return i
	}

	for _, host := range isMasterResult.Hosts {
		i.Members = append(i.Members, Addr(host).Canonicalize())
	}

	for _, passive := range isMasterResult.Passives {
		i.Members = append(i.Members, Addr(passive).Canonicalize())
	}

	for _, arbiter := range isMasterResult.Arbiters {
		i.Members = append(i.Members, Addr(arbiter).Canonicalize())
	}

	i.Kind = Standalone

	if isMasterResult.IsReplicaSet {
		i.Kind = RSGhost
	} else if isMasterResult.SetName != "" {
		if isMasterResult.IsMaster {
			i.Kind = RSPrimary
		} else if isMasterResult.Hidden {
			i.Kind = RSMember
		} else if isMasterResult.Secondary {
			i.Kind = RSSecondary
		} else if isMasterResult.ArbiterOnly {
			i.Kind = RSArbiter
		} else {
			i.Kind = RSMember
		}
	} else if isMasterResult.Msg == "isdbgrid" {
		i.Kind = Mongos
	}

	return i
}
