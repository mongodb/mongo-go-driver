// Package result contains the results from various operations.
package result

import (
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// Insert is a result from an Insert command.
type Insert struct {
	N int
}

// Delete is a result from a Delete command.
type Delete struct {
	N int
}

// Distinct is a result from a Distinct command.
type Distinct struct {
	Values []interface{}
}

// FindAndModify is a result from a findAndModify command.
type FindAndModify struct {
	Value           bson.Reader
	LastErrorObject struct {
		UpdatedExisting bool
		Upserted        interface{}
	}
}

// Document is a result from a command that returns a single Document.
type Document struct{}

// ListDatabases is the result from a listDatabases command.
type ListDatabases struct {
	Databases []struct {
		Name       string
		SizeOnDisk int64 `bson:"sizeOnDisk"`
		Empty      bool
	}
	TotalSize int64 `bson:"totalSize"`
}

// Update is a result of an Update command.
type Update struct {
	MatchedCount  int64 `bson:"n"`
	ModifiedCount int64 `bson:"nModified"`
	Upserted      struct {
		ID interface{} `bson:"_id"`
	} `bson:"upserted"`
}

// IsMaster is a result of an IsMaster command.
type IsMaster struct {
	Arbiters            []string          `bson:"arbiters,omitempty"`
	ArbiterOnly         bool              `bson:"arbiterOnly,omitempty"`
	ElectionID          objectid.ObjectID `bson:"electionId,omitempty"`
	Hidden              bool              `bson:"hidden,omitempty"`
	Hosts               []string          `bson:"hosts,omitempty"`
	IsMaster            bool              `bson:"ismaster,omitempty"`
	IsReplicaSet        bool              `bson:"isreplicaset,omitempty"`
	LastWriteTimestamp  time.Time         `bson:"lastWriteDate,omitempty"`
	MaxBSONObjectSize   uint32            `bson:"maxBsonObjectSize,omitempty"`
	MaxMessageSizeBytes uint32            `bson:"maxMessageSizeBytes,omitempty"`
	MaxWriteBatchSize   uint16            `bson:"maxWriteBatchSize,omitempty"`
	Me                  string            `bson:"me,omitempty"`
	MaxWireVersion      int32             `bson:"maxWireVersion,omitempty"`
	MinWireVersion      int32             `bson:"minWireVersion,omitempty"`
	Msg                 string            `bson:"msg,omitempty"`
	OK                  int32             `bson:"ok"`
	Passives            []string          `bson:"passives,omitempty"`
	ReadOnly            bool              `bson:"readOnly,omitempty"`
	Secondary           bool              `bson:"secondary,omitempty"`
	SetName             string            `bson:"setName,omitempty"`
	SetVersion          uint32            `bson:"setVersion,omitempty"`
	Tags                map[string]string `bson:"tags,omitempty"`
}

// BuildInfo is a result of a BuildInfo command.
type BuildInfo struct {
	OK           bool    `bson:"ok"`
	GitVersion   string  `bson:"gitVersion,omitempty"`
	Version      string  `bson:"version,omitempty"`
	VersionArray []uint8 `bson:"versionArray,omitempty"`
}

// IsZero returns true if the BuildInfo is the zero value.
func (bi BuildInfo) IsZero() bool {
	if !bi.OK && bi.GitVersion == "" && bi.Version == "" && bi.VersionArray == nil {
		return true
	}

	return false
}

// GetLastError is a result of a GetLastError command.
type GetLastError struct {
	ConnectionID uint32 `bson:"connectionId"`
}

// KillCursors is a result of a KillCursors command.
type KillCursors struct {
	CursorsKilled   []int64 `bson:"cursorsKilled"`
	CursorsNotFound []int64 `bson:"cursorsNotFound"`
	CursorsAlive    []int64 `bson:"cursorsAlive"`
}
