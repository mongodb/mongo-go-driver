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
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// Aggregate represents the aggregate command.
//
// The aggregate command performs an aggregation.
type Aggregate struct {
	NS           Namespace
	Pipeline     *bson.Array
	Opts         []option.AggregateOptioner
	ReadPref     *readpref.ReadPref
	WriteConcern *writeconcern.WriteConcern
	ReadConcern  *readconcern.ReadConcern
	Clock        *session.ClusterClock
	Session      *session.Client

	result Cursor
	err    error
}

// Encode will encode this command into a wire message for the given server description.
func (a *Aggregate) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	cmd, err := a.encode(desc)
	if err != nil {
		return nil, err
	}

	return cmd.Encode(desc)
}

func (a *Aggregate) encode(desc description.SelectedServer) (*Read, error) {
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

	// add write concern because it won't be added by the Read command's Encode()
	if a.WriteConcern != nil {
		element, err := a.WriteConcern.MarshalBSONElement()
		if err != nil {
			return nil, err
		}

		command.Append(element)
	}

	return &Read{
		DB:          a.NS.DB,
		Command:     command,
		ReadPref:    a.ReadPref,
		ReadConcern: a.ReadConcern,
		Clock:       a.Clock,
		Session:     a.Session,
	}, nil
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
	rdr, err := (&Read{}).Decode(desc, wm).Result()
	if err != nil {
		a.err = err
		return a
	}

	return a.decode(desc, cb, rdr)
}

func (a *Aggregate) decode(desc description.SelectedServer, cb CursorBuilder, rdr bson.Reader) *Aggregate {
	opts := make([]option.CursorOptioner, 0)
	for _, opt := range a.Opts {
		curOpt, ok := opt.(option.CursorOptioner)
		if !ok {
			continue
		}
		opts = append(opts, curOpt)
	}

	a.result, a.err = cb.BuildCursor(rdr, a.Session, a.Clock, opts...)
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
	cmd, err := a.encode(desc)
	if err != nil {
		return nil, err
	}

	rdr, err := cmd.RoundTrip(ctx, desc, rw)
	if err != nil {
		return nil, err
	}

	return a.decode(desc, cb, rdr).Result()
}
