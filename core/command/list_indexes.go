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
	"github.com/mongodb/mongo-go-driver/x/bsonx"
)

// ListIndexes represents the listIndexes command.
//
// The listIndexes command lists the indexes for a namespace.
type ListIndexes struct {
	Clock      *session.ClusterClock
	NS         Namespace
	CursorOpts []bsonx.Elem
	Opts       []bsonx.Elem
	Session    *session.Client

	result Cursor
	err    error
}

// Encode will encode this command into a wire message for the given server description.
func (li *ListIndexes) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	encoded, err := li.encode(desc)
	if err != nil {
		return nil, err
	}
	return encoded.Encode(desc)
}

func (li *ListIndexes) encode(desc description.SelectedServer) (*Read, error) {
	cmd := bsonx.Doc{{"listIndexes", bsonx.String(li.NS.Collection)}}
	cmd = append(cmd, li.Opts...)

	return &Read{
		Clock:   li.Clock,
		DB:      li.NS.DB,
		Command: cmd,
		Session: li.Session,
	}, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoling
// are deferred until either the Result or Err methods are called.
func (li *ListIndexes) Decode(desc description.SelectedServer, cb CursorBuilder, wm wiremessage.WireMessage) *ListIndexes {
	rdr, err := (&Read{}).Decode(desc, wm).Result()
	if err != nil {
		if IsNotFound(err) {
			li.result = emptyCursor{}
			return li
		}
		li.err = err
		return li
	}

	return li.decode(desc, cb, rdr)
}

func (li *ListIndexes) decode(desc description.SelectedServer, cb CursorBuilder, rdr bson.Raw) *ListIndexes {
	labels, err := getErrorLabels(&rdr)
	li.err = err

	res, err := cb.BuildCursor(rdr, li.Session, li.Clock, li.CursorOpts...)
	li.result = res
	if err != nil {
		li.err = Error{Message: err.Error(), Labels: labels}
	}

	return li
}

// Result returns the result of a decoded wire message and server description.
func (li *ListIndexes) Result() (Cursor, error) {
	if li.err != nil {
		return nil, li.err
	}
	return li.result, nil
}

// Err returns the error set on this command.
func (li *ListIndexes) Err() error { return li.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (li *ListIndexes) RoundTrip(ctx context.Context, desc description.SelectedServer, cb CursorBuilder, rw wiremessage.ReadWriter) (Cursor, error) {
	cmd, err := li.encode(desc)
	if err != nil {
		return nil, err
	}

	rdr, err := cmd.RoundTrip(ctx, desc, rw)
	if err != nil {
		if IsNotFound(err) {
			return emptyCursor{}, nil
		}
		return nil, err
	}

	return li.decode(desc, cb, rdr).Result()
}
