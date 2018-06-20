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
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// Find represents the find command.
//
// The find command finds documents within a collection that match a filter.
type Find struct {
	NS          Namespace
	Filter      *bson.Document
	Opts        []option.FindOptioner
	ReadPref    *readpref.ReadPref
	ReadConcern *readconcern.ReadConcern

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

	command := bson.NewDocument(bson.EC.String("find", f.NS.Collection))

	if f.Filter != nil {
		command.Append(bson.EC.SubDocument("filter", f.Filter))
	}

	var limit int64
	var batchSize int32
	var err error

	for _, opt := range f.Opts {
		switch t := opt.(type) {
		case nil:
			continue
		case option.OptLimit:
			limit = int64(t)
			err = opt.Option(command)
		case option.OptBatchSize:
			batchSize = int32(t)
			err = opt.Option(command)
		case option.OptProjection:
			err = t.Option(command)
		default:
			err = opt.Option(command)
		}
		if err != nil {
			return nil, err
		}
	}

	if limit != 0 && batchSize != 0 && limit <= int64(batchSize) {
		command.Append(bson.EC.Boolean("singleBatch", true))
	}

	return &Read{
		DB:          f.NS.DB,
		ReadPref:    f.ReadPref,
		Command:     command,
		ReadConcern: f.ReadConcern,
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

func (f *Find) decode(desc description.SelectedServer, cb CursorBuilder, rdr bson.Reader) *Find {
	opts := make([]option.CursorOptioner, 0)
	for _, opt := range f.Opts {
		curOpt, ok := opt.(option.CursorOptioner)
		if !ok {
			continue
		}
		opts = append(opts, curOpt)
	}

	f.result, f.err = cb.BuildCursor(rdr, opts...)
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
