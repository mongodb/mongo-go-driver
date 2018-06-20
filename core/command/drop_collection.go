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

	result bson.Reader
	err    error
}

// Encode will encode this command into a wire message for the given server description.
func (di *DropCollection) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	cmd := bson.NewDocument(
		bson.EC.String("drop", di.Collection),
	)

	return (&Write{
		WriteConcern: di.WriteConcern,
		DB:           di.DB,
		Command:      cmd,
	}).Encode(desc)
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (di *DropCollection) Decode(desc description.SelectedServer, wm wiremessage.WireMessage) *DropCollection {
	di.result, di.err = (&Write{}).Decode(desc, wm).Result()
	return di
}

// Result returns the result of a decoded wire message and server description.
func (di *DropCollection) Result() (bson.Reader, error) {
	if di.err != nil {
		return nil, di.err
	}
	return di.result, nil
}

// Err returns the error set on this command.
func (di *DropCollection) Err() error { return di.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (di *DropCollection) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriteCloser) (bson.Reader, error) {
	wm, err := di.Encode(desc)
	if err != nil {
		return nil, err
	}

	if !ackWrite(di.WriteConcern) {
		go func() {
			defer func() { _ = recover() }()
			defer func() { _ = rw.Close() }()

			err = rw.WriteWireMessage(ctx, wm)
			if err != nil {
				return
			}
			_, _ = rw.ReadWireMessage(ctx)
		}()
		return nil, ErrUnacknowledgedWrite
	}

	defer func() { _ = rw.Close() }()
	err = rw.WriteWireMessage(ctx, wm)
	if err != nil {
		return nil, err
	}
	wm, err = rw.ReadWireMessage(ctx)
	if err != nil {
		return nil, err
	}
	return di.Decode(desc, wm).Result()
}
