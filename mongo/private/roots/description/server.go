package description

import (
	"fmt"
	"time"

	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/addr"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
)

// UnsetRTT is the unset value for a round trip time.
const UnsetRTT = -1 * time.Millisecond

type Topology struct{}

func (t Topology) Server(addr.Addr) (Server, bool) { return Server{}, false }

type TopologyDiff struct{}

func DiffTopology(Topology, Topology) TopologyDiff { return TopologyDiff{} }

type Kind uint32

const (
	Single                Kind = 1
	ReplicaSet            Kind = 2
	ReplicaSetNoPrimary   Kind = 4 + ReplicaSet
	ReplicaSetWithPrimary Kind = 8 + ReplicaSet
	Sharded               Kind = 256
)

type Server struct {
	Addr addr.Addr

	AverageRTT        time.Duration
	AverageRTTSet     bool
	CanonicalAddr     addr.Addr
	ElectionID        objectid.ObjectID
	GitVersion        string
	HeartbeatInterval time.Duration
	LastError         error
	LastUpdateTime    time.Time
	LastWriteTime     time.Time
	MaxBatchCount     uint16
	MaxDocumentSize   uint32
	MaxMessageSize    uint32
	Members           []addr.Addr
	ReadOnly          bool
	SetName           string
	SetVersion        uint32
	Tags              TagSet
	Kind              ServerKind
	WireVersion       *VersionRange
	Version           Version
}

func (s Server) SetAverageRTT(rtt time.Duration) Server {
	s.AverageRTT = rtt
	if rtt == UnsetRTT {
		s.AverageRTTSet = false
	} else {
		s.AverageRTTSet = true
	}

	return s
}

func NewServer(address addr.Addr, isMaster result.IsMaster, buildInfo result.BuildInfo) Server {
	i := Server{
		Addr: address,

		CanonicalAddr:   addr.Addr(isMaster.Me).Canonicalize(),
		ElectionID:      isMaster.ElectionID,
		LastUpdateTime:  time.Now().UTC(),
		LastWriteTime:   isMaster.LastWriteTimestamp,
		MaxBatchCount:   isMaster.MaxWriteBatchSize,
		MaxDocumentSize: isMaster.MaxBSONObjectSize,
		MaxMessageSize:  isMaster.MaxMessageSizeBytes,
		SetName:         isMaster.SetName,
		SetVersion:      isMaster.SetVersion,
		Tags:            NewTagSetFromMap(isMaster.Tags),
	}

	if !buildInfo.IsZero() {
		i.GitVersion = buildInfo.GitVersion
		i.Version.Desc = buildInfo.Version
		i.Version.Parts = buildInfo.VersionArray
	}

	if i.CanonicalAddr == "" {
		i.CanonicalAddr = address
	}

	if isMaster.OK != 1 {
		i.LastError = fmt.Errorf("not ok")
		return i
	}

	for _, host := range isMaster.Hosts {
		i.Members = append(i.Members, addr.Addr(host).Canonicalize())
	}

	for _, passive := range isMaster.Passives {
		i.Members = append(i.Members, addr.Addr(passive).Canonicalize())
	}

	for _, arbiter := range isMaster.Arbiters {
		i.Members = append(i.Members, addr.Addr(arbiter).Canonicalize())
	}

	i.Kind = Standalone

	if isMaster.IsReplicaSet {
		i.Kind = RSGhost
	} else if isMaster.SetName != "" {
		if isMaster.IsMaster {
			i.Kind = RSPrimary
		} else if isMaster.Hidden {
			i.Kind = RSMember
		} else if isMaster.Secondary {
			i.Kind = RSSecondary
		} else if isMaster.ArbiterOnly {
			i.Kind = RSArbiter
		} else {
			i.Kind = RSMember
		}
	} else if isMaster.Msg == "isdbgrid" {
		i.Kind = Mongos
	}

	i.WireVersion = &VersionRange{
		Min: isMaster.MinWireVersion,
		Max: isMaster.MaxWireVersion,
	}

	return i
}

type ServerKind uint32

const (
	Standalone  ServerKind = 1
	RSMember    ServerKind = 2
	RSPrimary   ServerKind = 4 + RSMember
	RSSecondary ServerKind = 8 + RSMember
	RSArbiter   ServerKind = 16 + RSMember
	RSGhost     ServerKind = 32 + RSMember
	Mongos      ServerKind = 256
)

type Version struct {
	Desc  string
	Parts []uint8
}

type VersionRange struct {
	Min int32
	Max int32
}
