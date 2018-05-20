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
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// Aggregate represents the aggregate command.
//
// The aggregate command performs an aggregation.
type Aggregate struct {
	NS       Namespace
	Pipeline *bson.Array
	Opts     []option.AggregateOptioner
	ReadPref *readpref.ReadPref

	result Cursor
	err    error
}

// Encode will encode this command into a wire message for the given server description.
func (a *Aggregate) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	if err := a.NS.Validate(); err != nil {
		return nil, err
	}

	command := bson.NewDocument()
	command.Append(bson.EC.String("aggregate", a.NS.Collection), bson.EC.Array("pipeline", a.Pipeline))

	cursor := bson.NewDocument()
	command.Append(bson.EC.SubDocument("cursor", cursor))

	for _, opt := range a.Opts {
		switch t := opt.(type) {
		case nil:
			continue
		case option.OptBatchSize:
			if t == 0 && a.HasDollarOut() {
				continue
			}
			err := opt.Option(cursor)
			if err != nil {
				return nil, err
			}
		default:
			err := opt.Option(command)
			if err != nil {
				return nil, err
			}
		}
	}

	return (&Command{DB: a.NS.DB, Command: command, ReadPref: a.ReadPref}).Encode(desc)
}

// HasDollarOut returns true if the Pipeline field contains a $out stage.
func (a *Aggregate) HasDollarOut() bool {
	if a.Pipeline == nil {
		return false
	}
	if a.Pipeline.Len() == 0 {
		return false
	}

	val, err := a.Pipeline.Lookup(uint(a.Pipeline.Len() - 1))
	if err != nil {
		return false
	}

	doc, ok := val.MutableDocumentOK()
	if !ok || doc.Len() != 1 {
		return false
	}
	elem, ok := doc.ElementAtOK(0)
	if !ok {
		return false
	}
	return elem.Key() == "$out"
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (a *Aggregate) Decode(desc description.SelectedServer, cb CursorBuilder, wm wiremessage.WireMessage) *Aggregate {
	rdr, err := (&Command{}).Decode(desc, wm).Result()
	if err != nil {
		a.err = err
		return a
	}

	opts := make([]option.CursorOptioner, 0)
	for _, opt := range a.Opts {
		curOpt, ok := opt.(option.CursorOptioner)
		if !ok {
			continue
		}
		opts = append(opts, curOpt)
	}

	a.result, a.err = cb.BuildCursor(rdr, opts...)

	return a
}

// Result returns the result of a decoded wire message and server description.
func (a *Aggregate) Result() (Cursor, error) {
	if a.err != nil {
		return nil, a.err
	}
	return a.result, nil
}

// Err returns the error set on this command.
func (a *Aggregate) Err() error { return a.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (a *Aggregate) RoundTrip(ctx context.Context, desc description.SelectedServer, cb CursorBuilder, rw wiremessage.ReadWriter) (Cursor, error) {
	wm, err := a.Encode(desc)
	if err != nil {
		return nil, err
	}

	err = rw.WriteWireMessage(ctx, wm)
	if err != nil {
		return nil, err
	}
	wm, err = rw.ReadWireMessage(ctx)
	if err != nil {
		return nil, err
	}
	return a.Decode(desc, cb, wm).Result()
}
