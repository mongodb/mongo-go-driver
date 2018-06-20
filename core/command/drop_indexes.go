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
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// DropIndexes represents the dropIndexes command.
//
// The dropIndexes command drops indexes for a namespace.
type DropIndexes struct {
	NS           Namespace
	Index        string
	Opts         []option.DropIndexesOptioner
	WriteConcern *writeconcern.WriteConcern

	result bson.Reader
	err    error
}

// Encode will encode this command into a wire message for the given server description.
func (di *DropIndexes) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	cmd, err := di.encode(desc)
	if err != nil {
		return nil, err
	}

	return cmd.Encode(desc)
}

func (di *DropIndexes) encode(desc description.SelectedServer) (*Write, error) {
	cmd := bson.NewDocument(
		bson.EC.String("dropIndexes", di.NS.Collection),
		bson.EC.String("index", di.Index),
	)

	for _, opt := range di.Opts {
		if opt == nil {
			continue
		}
		err := opt.Option(cmd)
		if err != nil {
			return nil, err
		}
	}

	return &Write{
		DB:           di.NS.DB,
		Command:      cmd,
		WriteConcern: di.WriteConcern,
	}, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (di *DropIndexes) Decode(desc description.SelectedServer, wm wiremessage.WireMessage) *DropIndexes {
	di.result, di.err = (&Write{}).Decode(desc, wm).Result()
	return di
}

// Result returns the result of a decoded wire message and server description.
func (di *DropIndexes) Result() (bson.Reader, error) {
	if di.err != nil {
		return nil, di.err
	}
	return di.result, nil
}

// Err returns the error set on this command.
func (di *DropIndexes) Err() error { return di.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (di *DropIndexes) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriter) (bson.Reader, error) {
	cmd, err := di.encode(desc)
	if err != nil {
		return nil, err
	}

	di.result, err = cmd.RoundTrip(ctx, desc, rw)
	if err != nil {
		return nil, err
	}

	return di.Result()
}
