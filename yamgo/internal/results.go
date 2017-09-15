package internal

import (
	"time"

	"github.com/10gen/mongo-go-driver/bson"
)

// IsMasterResult is the result of executing this
// ismaster command.
type IsMasterResult struct {
	Arbiters            []string          `bson:"arbiters,omitempty"`
	ArbiterOnly         bool              `bson:"arbiterOnly,omitempty"`
	ElectionID          bson.ObjectId     `bson:"electionId,omitempty"`
	Hidden              bool              `bson:"hidden,omitempty"`
	Hosts               []string          `bson:"hosts,omitempty"`
	IsMaster            bool              `bson:"ismaster,omitempty"`
	IsReplicaSet        bool              `bson:"isreplicaset,omitempty"`
	LastWriteTimestamp  time.Time         `bson:"lastWriteDate,omitempty"`
	MaxBSONObjectSize   uint32            `bson:"maxBsonObjectSize,omitempty"`
	MaxMessageSizeBytes uint32            `bson:"maxMessageSizeBytes,omitempty"`
	MaxWriteBatchSize   uint16            `bson:"maxWriteBatchSize,omitempty"`
	Me                  string            `bson:"me,omitempty"`
	MaxWireVersion      uint8             `bson:"maxWireVersion,omitempty"`
	MinWireVersion      uint8             `bson:"minWireVersion,omitempty"`
	Msg                 string            `bson:"msg,omitempty"`
	OK                  bool              `bson:"ok"`
	Passives            []string          `bson:"passives,omitempty"`
	ReadOnly            bool              `bson:"readOnly,omitempty"`
	Secondary           bool              `bson:"secondary,omitempty"`
	SetName             string            `bson:"setName,omitempty"`
	SetVersion          uint32            `bson:"setVersion,omitempty"`
	Tags                map[string]string `bson:"tags,omitempty"`
}

// BuildInfoResult is the result of executing the
// buildInfo command.
type BuildInfoResult struct {
	OK           bool    `bson:"ok"`
	GitVersion   string  `bson:"gitVersion,omitempty"`
	Version      string  `bson:"version,omitempty"`
	VersionArray []uint8 `bson:"versionArray,omitempty"`
}

// GetLastErrorResult is the result of executing the
// getLastError command.
type GetLastErrorResult struct {
	ConnectionID uint32 `bson:"connectionId"`
}
