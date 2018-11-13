// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"context"
	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
	"github.com/mongodb/mongo-go-driver/mongo/readconcern"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
)

// CountDocuments represents the CountDocuments command.
//
// The countDocuments command counts how many documents in a collection match the given query.
type CountDocuments struct {
	NS          Namespace
	Pipeline    bsonx.Arr
	Opts        []bsonx.Elem
	ReadPref    *readpref.ReadPref
	ReadConcern *readconcern.ReadConcern
	Clock       *session.ClusterClock
	Session     *session.Client

	result int64
	err    error
}

// Encode will encode this command into a wire message for the given server description.
func (c *CountDocuments) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	if err := c.NS.Validate(); err != nil {
		return nil, err
	}
	command := bsonx.Doc{{"aggregate", bsonx.String(c.NS.Collection)}, {"pipeline", bsonx.Array(c.Pipeline)}}

	command = append(command, bsonx.Elem{"cursor", bsonx.Document(bsonx.Doc{})})
	command = append(command, c.Opts...)

	return (&Read{DB: c.NS.DB, ReadPref: c.ReadPref, Command: command}).Encode(desc)
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (c *CountDocuments) Decode(ctx context.Context, desc description.SelectedServer, cb CursorBuilder, wm wiremessage.WireMessage) *CountDocuments {
	rdr, err := (&Read{}).Decode(desc, wm).Result()
	if err != nil {
		c.err = err
		return c
	}
	cur, err := cb.BuildCursor(rdr, c.Session, c.Clock)
	if err != nil {
		c.err = err
		return c
	}

	var doc bsonx.Doc
	if cur.Next(ctx) {
		err = cur.Decode(&doc)
		if err != nil {
			c.err = err
			return c
		}
		val, err := doc.LookupErr("n")
		switch err.(type) {
		case bsonx.KeyNotFound:
			c.err = errors.New("Invalid response from server, no 'n' field")
			return c
		case nil:
		default:
			c.err = err
			return c
		}
		switch val.Type() {
		case bson.TypeInt32:
			c.result = int64(val.Int32())
		case bson.TypeInt64:
			c.result = val.Int64()
		default:
			c.err = errors.New("Invalid response from server, value field is not a number")
		}

		return c
	}

	c.result = 0
	return c
}

// Result returns the result of a decoded wire message and server description.
func (c *CountDocuments) Result() (int64, error) {
	if c.err != nil {
		return 0, c.err
	}
	return c.result, nil
}

// Err returns the error set on this command.
func (c *CountDocuments) Err() error { return c.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (c *CountDocuments) RoundTrip(ctx context.Context, desc description.SelectedServer, cb CursorBuilder, rw wiremessage.ReadWriter) (int64, error) {
	wm, err := c.Encode(desc)
	if err != nil {
		return 0, err
	}

	err = rw.WriteWireMessage(ctx, wm)
	if err != nil {
		return 0, err
	}
	wm, err = rw.ReadWireMessage(ctx)
	if err != nil {
		return 0, err
	}
	return c.Decode(ctx, desc, cb, wm).Result()
}
