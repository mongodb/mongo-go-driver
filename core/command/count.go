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
	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// Count represents the count command.
//
// The count command counts how many documents in a collection match the given query.
type Count struct {
	NS          Namespace
	Query       *bson.Document
	Opts        []option.CountOptioner
	ReadPref    *readpref.ReadPref
	ReadConcern *readconcern.ReadConcern

	result int64
	err    error
}

// Encode will encode this command into a wire message for the given server description.
func (c *Count) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	cmd, err := c.encode(desc)
	if err != nil {
		return nil, err
	}

	return cmd.Encode(desc)
}

func (c *Count) encode(desc description.SelectedServer) (*Read, error) {
	if err := c.NS.Validate(); err != nil {
		return nil, err
	}

	command := bson.NewDocument(bson.EC.String("count", c.NS.Collection), bson.EC.SubDocument("query", c.Query))
	for _, opt := range c.Opts {
		if opt == nil {
			continue
		}
		err := opt.Option(command)
		if err != nil {
			return nil, err
		}
	}

	return &Read{
		DB:          c.NS.DB,
		ReadPref:    c.ReadPref,
		Command:     command,
		ReadConcern: c.ReadConcern,
	}, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (c *Count) Decode(desc description.SelectedServer, wm wiremessage.WireMessage) *Count {
	rdr, err := (&Read{}).Decode(desc, wm).Result()
	if err != nil {
		c.err = err
		return c
	}

	return c.decode(desc, rdr)
}

func (c *Count) decode(desc description.SelectedServer, rdr bson.Reader) *Count {
	val, err := rdr.Lookup("n")
	switch {
	case err == bson.ErrElementNotFound:
		c.err = errors.New("invalid response from server, no 'n' field")
		return c
	case err != nil:
		c.err = err
		return c
	}

	switch val.Value().Type() {
	case bson.TypeInt32:
		c.result = int64(val.Value().Int32())
	case bson.TypeInt64:
		c.result = val.Value().Int64()
	default:
		c.err = errors.New("invalid response from server, value field is not a number")
	}

	return c
}

// Result returns the result of a decoded wire message and server description.
func (c *Count) Result() (int64, error) {
	if c.err != nil {
		return 0, c.err
	}
	return c.result, nil
}

// Err returns the error set on this command.
func (c *Count) Err() error { return c.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (c *Count) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriter) (int64, error) {
	cmd, err := c.encode(desc)
	if err != nil {
		return 0, err
	}

	rdr, err := cmd.RoundTrip(ctx, desc, rw)
	if err != nil {
		return 0, err
	}

	return c.decode(desc, rdr).Result()
}
