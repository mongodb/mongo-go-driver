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
	"github.com/mongodb/mongo-go-driver/mongo/readconcern"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
)

// Find represents the find command.
//
// The find command finds documents within a collection that match a filter.
type Find struct {
	NS          Namespace
	Filter      bsonx.Doc
	CursorOpts  []bsonx.Elem
	Opts        []bsonx.Elem
	ReadPref    *readpref.ReadPref
	ReadConcern *readconcern.ReadConcern
	Clock       *session.ClusterClock
	Session     *session.Client

	result Cursor
	err    error
}

// Encode will encode this command into a wire message for the given server description.
func (f *Find) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	cmd, err := f.encode(desc)
	if err != nil {
		return nil, err
	}

	return cmd.Encode(desc)
}

func (f *Find) encode(desc description.SelectedServer) (*Read, error) {
	if err := f.NS.Validate(); err != nil {
		return nil, err
	}

	command := bsonx.Doc{{"find", bsonx.String(f.NS.Collection)}}

	if f.Filter != nil {
		command = append(command, bsonx.Elem{"filter", bsonx.Document(f.Filter)})
	}

	command = append(command, f.Opts...)

	return &Read{
		Clock:       f.Clock,
		DB:          f.NS.DB,
		ReadPref:    f.ReadPref,
		Command:     command,
		ReadConcern: f.ReadConcern,
		Session:     f.Session,
	}, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (f *Find) Decode(desc description.SelectedServer, cb CursorBuilder, wm wiremessage.WireMessage) *Find {
	rdr, err := (&Read{}).Decode(desc, wm).Result()
	if err != nil {
		f.err = err
		return f
	}

	return f.decode(desc, cb, rdr)
}

func (f *Find) decode(desc description.SelectedServer, cb CursorBuilder, rdr bson.Raw) *Find {
	labels, err := getErrorLabels(&rdr)
	f.err = err

	res, err := cb.BuildCursor(rdr, f.Session, f.Clock, f.CursorOpts...)
	f.result = res
	if err != nil {
		f.err = Error{Message: err.Error(), Labels: labels}
	}
	return f
}

// Result returns the result of a decoded wire message and server description.
func (f *Find) Result() (Cursor, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.result, nil
}

// Err returns the error set on this command.
func (f *Find) Err() error { return f.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (f *Find) RoundTrip(ctx context.Context, desc description.SelectedServer, cb CursorBuilder, rw wiremessage.ReadWriter) (Cursor, error) {
	cmd, err := f.encode(desc)
	if err != nil {
		return nil, err
	}

	rdr, err := cmd.RoundTrip(ctx, desc, rw)
	if err != nil {
		return nil, err
	}

	return f.decode(desc, cb, rdr).Result()
}
