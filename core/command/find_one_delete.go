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
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// FindOneAndDelete represents the findOneAndDelete operation.
//
// The findOneAndDelete command deletes a single document that matches a query and returns it.
type FindOneAndDelete struct {
	NS           Namespace
	Query        *bson.Document
	Opts         []option.FindOneAndDeleteOptioner
	WriteConcern *writeconcern.WriteConcern
	Clock        *session.ClusterClock
	Session      *session.Client

	result result.FindAndModify
	err    error
}

// Encode will encode this command into a wire message for the given server description.
func (f *FindOneAndDelete) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	cmd, err := f.encode(desc)
	if err != nil {
		return nil, err
	}

	return cmd.Encode(desc)
}

func (f *FindOneAndDelete) encode(desc description.SelectedServer) (*Write, error) {
	if err := f.NS.Validate(); err != nil {
		return nil, err
	}

	command := bson.NewDocument(
		bson.EC.String("findAndModify", f.NS.Collection),
		bson.EC.SubDocument("query", f.Query),
		bson.EC.Boolean("remove", true),
	)

	for _, opt := range f.Opts {
		if opt == nil {
			continue
		}
		err := opt.Option(command)
		if err != nil {
			return nil, err
		}
	}

	return &Write{
		Clock:        f.Clock,
		DB:           f.NS.DB,
		Command:      command,
		WriteConcern: f.WriteConcern,
		Session:      f.Session,
	}, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (f *FindOneAndDelete) Decode(desc description.SelectedServer, wm wiremessage.WireMessage) *FindOneAndDelete {
	rdr, err := (&Write{}).Decode(desc, wm).Result()
	if err != nil {
		f.err = err
		return f
	}

	return f.decode(desc, rdr)
}

func (f *FindOneAndDelete) decode(desc description.SelectedServer, rdr bson.Reader) *FindOneAndDelete {
	f.result, f.err = unmarshalFindAndModifyResult(rdr)
	return f
}

// Result returns the result of a decoded wire message and server description.
func (f *FindOneAndDelete) Result() (result.FindAndModify, error) {
	if f.err != nil {
		return result.FindAndModify{}, f.err
	}
	return f.result, nil
}

// Err returns the error set on this command.
func (f *FindOneAndDelete) Err() error { return f.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (f *FindOneAndDelete) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriter) (result.FindAndModify, error) {
	cmd, err := f.encode(desc)
	if err != nil {
		return result.FindAndModify{}, err
	}

	rdr, err := cmd.RoundTrip(ctx, desc, rw)
	if err != nil {
		return result.FindAndModify{}, err
	}

	return f.decode(desc, rdr).Result()
}
