// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package result contains the results from various operations.
package result

import (
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// Insert is a result from an Insert command.
type Insert struct {
	N                 int
	WriteErrors       []WriteError       `bson:"writeErrors"`
	WriteConcernError *WriteConcernError `bson:"writeConcernError"`
}

// StartSession is a result from a StartSession command.
type StartSession struct {
	ID *bson.Document `bson:"id"`
}

// EndSessions is a result from an EndSessions command.
type EndSessions struct{}

// Delete is a result from a Delete command.
type Delete struct {
	N                 int
	WriteErrors       []WriteError       `bson:"writeErrors"`
	WriteConcernError *WriteConcernError `bson:"writeConcernError"`
}

// Update is a result of an Update command.
type Update struct {
	MatchedCount  int64 `bson:"n"`
	ModifiedCount int64 `bson:"nModified"`
	Upserted      []struct {
		ID interface{} `bson:"_id"`
	} `bson:"upserted"`
	WriteErrors       []WriteError       `bson:"writeErrors"`
	WriteConcernError *WriteConcernError `bson:"writeConcernError"`
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

// WriteError is an error from a write operation that is not a write concern
// error.
type WriteError struct {
	Index  int
	Code   int
	ErrMsg string
}

// WriteConcernError is an error related to a write concern.
type WriteConcernError struct {
	Code    int
	ErrMsg  string
	ErrInfo bson.Reader
}

// ListDatabases is the result from a listDatabases command.
type ListDatabases struct {
	Databases []struct {
		Name       string
		SizeOnDisk int64 `bson:"sizeOnDisk"`
		Empty      bool
	}
	TotalSize int64 `bson:"totalSize"`
}

// IsMaster is a result of an IsMaster command.
type IsMaster struct {
	Arbiters                     []string          `bson:"arbiters,omitempty"`
	ArbiterOnly                  bool              `bson:"arbiterOnly,omitempty"`
	ClusterTime                  *bson.Document    `bson:"$clusterTime,omitempty"`
	Compression                  []string          `bson:"compression,omitempty"`
	ElectionID                   objectid.ObjectID `bson:"electionId,omitempty"`
	Hidden                       bool              `bson:"hidden,omitempty"`
	Hosts                        []string          `bson:"hosts,omitempty"`
	IsMaster                     bool              `bson:"ismaster,omitempty"`
	IsReplicaSet                 bool              `bson:"isreplicaset,omitempty"`
	LastWriteTimestamp           time.Time         `bson:"lastWriteDate,omitempty"`
	LogicalSessionTimeoutMinutes uint32            `bson:"logicalSessionTimeoutMinutes,omitempty"`
	MaxBSONObjectSize            uint32            `bson:"maxBsonObjectSize,omitempty"`
	MaxMessageSizeBytes          uint32            `bson:"maxMessageSizeBytes,omitempty"`
	MaxWriteBatchSize            uint32            `bson:"maxWriteBatchSize,omitempty"`
	Me                           string            `bson:"me,omitempty"`
	MaxWireVersion               int32             `bson:"maxWireVersion,omitempty"`
	MinWireVersion               int32             `bson:"minWireVersion,omitempty"`
	Msg                          string            `bson:"msg,omitempty"`
	OK                           int32             `bson:"ok"`
	Passives                     []string          `bson:"passives,omitempty"`
	ReadOnly                     bool              `bson:"readOnly,omitempty"`
	Secondary                    bool              `bson:"secondary,omitempty"`
	SetName                      string            `bson:"setName,omitempty"`
	SetVersion                   uint32            `bson:"setVersion,omitempty"`
	Tags                         map[string]string `bson:"tags,omitempty"`
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

// CreateIndexes is a result of a CreateIndexes command.
type CreateIndexes struct {
	CreatedCollectionAutomatically bool `bson:"createdCollectionAutomatically"`
	IndexesBefore                  int  `bson:"numIndexesBefore"`
	IndexesAfter                   int  `bson:"numIndexesAfter"`
}
