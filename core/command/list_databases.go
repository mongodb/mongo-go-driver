// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/result"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// ListDatabases represents the listDatabases command.
//
// The listDatabases command lists the databases in a MongoDB deployment.
type ListDatabases struct {
	Filter *bson.Document
	Opts   []option.ListDatabasesOptioner

	result result.ListDatabases
	err    error
}

// Encode will encode this command into a wire message for the given server description.
func (ld *ListDatabases) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	encoded, err := ld.encode(desc)
	if err != nil {
		return nil, err
	}
	return encoded.Encode(desc)
}

func (ld *ListDatabases) encode(desc description.SelectedServer) (*Read, error) {
	cmd := bson.NewDocument(bson.EC.Int32("listDatabases", 1))

	if ld.Filter != nil {
		cmd.Append(bson.EC.SubDocument("filter", ld.Filter))
	}

	for _, opt := range ld.Opts {
		if opt == nil {
			continue
		}
		err := opt.Option(cmd)
		if err != nil {
			return nil, err
		}
	}

	return &Read{
		DB:      "admin",
		Command: cmd,
	}, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (ld *ListDatabases) Decode(desc description.SelectedServer, wm wiremessage.WireMessage) *ListDatabases {
	rdr, err := (&Read{}).Decode(desc, wm).Result()
	if err != nil {
		ld.err = err
		return ld
	}
	return ld.decode(desc, rdr)
}

func (ld *ListDatabases) decode(desc description.SelectedServer, rdr bson.Reader) *ListDatabases {
	ld.err = bson.Unmarshal(rdr, &ld.result)
	return ld
}

// Result returns the result of a decoded wire message and server description.
func (ld *ListDatabases) Result() (result.ListDatabases, error) {
	if ld.err != nil {
		return result.ListDatabases{}, ld.err
	}
	return ld.result, nil
}

// Err returns the error set on this command.
func (ld *ListDatabases) Err() error { return ld.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (ld *ListDatabases) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriter) (result.ListDatabases, error) {
	cmd, err := ld.encode(desc)
	if err != nil {
		return result.ListDatabases{}, err
	}

	rdr, err := cmd.RoundTrip(ctx, desc, rw)
	if err != nil {
		return result.ListDatabases{}, err
	}

	return ld.decode(desc, rdr).Result()
}
