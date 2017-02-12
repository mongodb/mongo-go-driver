package internal

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

// IsMasterResult is the result of executing this
// ismaster command.
type IsMasterResult struct {
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

// BuildInfoResult is the result of executing the
// buildInfo command.
type BuildInfoResult struct {
	GitVersion   string  `bson:"gitVersion"`
	Version      string  `bson:"version"`
	VersionArray []uint8 `bson:"versionArray"`
}

// GetLastErrorResult is the result of executing the
// getLastError command.
type GetLastErrorResult struct {
	ConnectionID uint32 `bson:"connectionId"`
}
