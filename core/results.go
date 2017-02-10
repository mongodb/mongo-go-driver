package core

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

type isMasterResult struct {
	Arbiters            []string      `bson:"arbiters"`
	ArbiterOnly         bool          `bson:"arbiterOnly"`
	ElectionID          bson.ObjectId `bson:"electionId"`
	Hidden              bool          `bson:"hidden"`
	Hosts               []string      `bson:"hosts"`
	IsMaster            bool          `bson:"ismaster"`
	IsReplicaSet        bool          `bson:"isreplicaset"`
	LastWriteTimestamp  time.Time     `bson:"lastWriteDate"`
	MaxBSONObjectSize   uint32        `bson:"maxBsonObjectSize"`
	MaxMessageSizeBytes uint32        `bson:"maxMessageSizeBytes"`
	MaxWriteBatchSize   uint16        `bson:"maxWriteBatchSize"`
	Me                  string        `bson:"me"`
	MaxWireVersion      uint8         `bson:"maxWireVersion"`
	MinWireVersion      uint8         `bson:"minWireVersion"`
	Msg                 string        `bson:"msg"`
	OK                  bool          `bson:"ok"`
	Passives            []string      `bson:"passives"`
	ReadOnly            bool          `bson:"readOnly"`
	Secondary           bool          `bson:"secondary"`
	SetName             string        `bson:"setName"`
	SetVersion          uint32        `bson:"setVersion"`
	Tags                []bson.D      `bson:"tags"`
}

func (r *isMasterResult) Members() []Endpoint {
	var members []Endpoint
	for _, host := range r.Hosts {
		members = append(members, Endpoint(host).Canonicalize())
	}

	for _, passive := range r.Passives {
		members = append(members, Endpoint(passive).Canonicalize())
	}

	for _, arbiter := range r.Arbiters {
		members = append(members, Endpoint(arbiter).Canonicalize())
	}

	return members
}

func (r *isMasterResult) ServerType() ServerType {
	if !r.OK {
		return UnknownServerType
	}

	if r.IsReplicaSet {
		return RSGhost
	}

	if r.SetName != "" {
		if r.IsMaster {
			return RSPrimary
		}
		if r.Hidden {
			return RSMember
		}
		if r.Secondary {
			return RSSecondary
		}
		if r.ArbiterOnly {
			return RSArbiter
		}

		return RSMember
	}

	if r.Msg == "isdbgrid" {
		return Mongos
	}

	return Standalone
}

type buildInfoResult struct {
	GitVersion   string  `bson:"gitVersion"`
	Version      string  `bson:"version"`
	VersionArray []uint8 `bson:"versionArray"`
}

type getLastErrorResult struct {
	ConnectionID uint32 `bson:"connectionId"`
}

// The result of a find command
type FindResult struct {
	// The cursor
	Cursor FirstBatchCursorResult `bson:"cursor"`
}

// An interface describe the initial results of a cursor
type CursorResult interface {
	// The namespace the cursor is in
	Namespace() *Namespace
	// The initial batch of results, which may be empty
	InitialBatch() []bson.Raw
	// The cursor id, which may be zero if no cursor was established
	CursorId() int64
}

// The first batch of a cursor
type FirstBatchCursorResult struct {
	// The first batch of the cursor
	FirstBatch []bson.Raw `bson:"firstBatch"`
	// The namespace to use for iterating the cursor
	NS string `bson:"ns"`
	// The cursor id
	ID int64 `bson:"id"`
}

func (cursorResult *FirstBatchCursorResult) Namespace() *Namespace {
	return NewNamespace(cursorResult.NS)
}

func (cursorResult *FirstBatchCursorResult) InitialBatch() []bson.Raw {
	return cursorResult.FirstBatch
}

func (cursorResult *FirstBatchCursorResult) CursorId() int64 {
	return cursorResult.ID
}
