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
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// DropCollection represents the drop command.
//
// The dropCollections command drops collection for a database.
type DropCollection struct {
	DB           string
	Collection   string
	WriteConcern *writeconcern.WriteConcern
	Clock        *session.ClusterClock
	Session      *session.Client

	result bson.Reader
	err    error
}

// Encode will encode this command into a wire message for the given server description.
func (dc *DropCollection) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	cmd, err := dc.encode(desc)
	if err != nil {
		return nil, err
	}

	return cmd.Encode(desc)
}

func (dc *DropCollection) encode(desc description.SelectedServer) (*Write, error) {
	cmd := bson.NewDocument(
		bson.EC.String("drop", dc.Collection),
	)

	return &Write{
		Clock:        dc.Clock,
		WriteConcern: dc.WriteConcern,
		DB:           dc.DB,
		Command:      cmd,
		Session:      dc.Session,
	}, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (dc *DropCollection) Decode(desc description.SelectedServer, wm wiremessage.WireMessage) *DropCollection {
	rdr, err := (&Write{}).Decode(desc, wm).Result()
	if err != nil {
		dc.err = err
		return dc
	}

	return dc.decode(desc, rdr)
}

func (dc *DropCollection) decode(desc description.SelectedServer, rdr bson.Reader) *DropCollection {
	dc.result = rdr
	return dc
}

// Result returns the result of a decoded wire message and server description.
func (dc *DropCollection) Result() (bson.Reader, error) {
	if dc.err != nil {
		return nil, dc.err
	}

	return dc.result, nil
}

// Err returns the error set on this command.
func (dc *DropCollection) Err() error { return dc.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (dc *DropCollection) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriter) (bson.Reader, error) {
	cmd, err := dc.encode(desc)
	if err != nil {
		return nil, err
	}

	rdr, err := cmd.RoundTrip(ctx, desc, rw)
	if err != nil {
		return nil, err
	}

	return dc.decode(desc, rdr).Result()
}
