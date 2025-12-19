// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package session

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

// ClientOptions represents all possible options for creating a client session.
type ClientOptions struct {
	CausalConsistency     *bool
	DefaultReadConcern    *readconcern.ReadConcern
	DefaultWriteConcern   *writeconcern.WriteConcern
	DefaultReadPreference *readpref.ReadPref
	Snapshot              *bool
	SnapshotTime          *bson.Timestamp
}

// TransactionOptions represents all possible options for starting a transaction in a session.
type TransactionOptions struct {
	ReadConcern    *readconcern.ReadConcern
	WriteConcern   *writeconcern.WriteConcern
	ReadPreference *readpref.ReadPref
}

func mergeClientOptions(opts ...*ClientOptions) *ClientOptions {
	c := &ClientOptions{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.CausalConsistency != nil {
			c.CausalConsistency = opt.CausalConsistency
		}
		if opt.DefaultReadConcern != nil {
			c.DefaultReadConcern = opt.DefaultReadConcern
		}
		if opt.DefaultReadPreference != nil {
			c.DefaultReadPreference = opt.DefaultReadPreference
		}
		if opt.DefaultWriteConcern != nil {
			c.DefaultWriteConcern = opt.DefaultWriteConcern
		}
		if opt.Snapshot != nil {
			c.Snapshot = opt.Snapshot
		}
		if opt.SnapshotTime != nil {
			c.SnapshotTime = opt.SnapshotTime
		}
	}

	return c
}
