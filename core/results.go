package core

import (
	"time"

	"github.com/10gen/mongo-go-driver/core/desc"

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

func (r *isMasterResult) Members() []desc.Endpoint {
	var members []desc.Endpoint
	for _, host := range r.Hosts {
		members = append(members, desc.Endpoint(host).Canonicalize())
	}

	for _, passive := range r.Passives {
		members = append(members, desc.Endpoint(passive).Canonicalize())
	}

	for _, arbiter := range r.Arbiters {
		members = append(members, desc.Endpoint(arbiter).Canonicalize())
	}

	return members
}

func (r *isMasterResult) ServerType() desc.ServerType {
	if !r.OK {
		return desc.UnknownServerType
	}

	if r.IsReplicaSet {
		return desc.RSGhost
	}

	if r.SetName != "" {
		if r.IsMaster {
			return desc.RSPrimary
		}
		if r.Hidden {
			return desc.RSMember
		}
		if r.Secondary {
			return desc.RSSecondary
		}
		if r.ArbiterOnly {
			return desc.RSArbiter
		}

		return desc.RSMember
	}

	if r.Msg == "isdbgrid" {
		return desc.Mongos
	}

	return desc.Standalone
}

type buildInfoResult struct {
	GitVersion   string  `bson:"gitVersion"`
	Version      string  `bson:"version"`
	VersionArray []uint8 `bson:"versionArray"`
}

type getLastErrorResult struct {
	ConnectionID uint32 `bson:"connectionId"`
}
