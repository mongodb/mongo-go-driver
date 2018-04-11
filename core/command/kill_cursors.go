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
	"github.com/mongodb/mongo-go-driver/core/result"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// KillCursors represents the killCursors command.
//
// The killCursors command kills a set of cursors.
type KillCursors struct {
	NS  Namespace
	IDs []int64

	result result.KillCursors
	err    error
}

// Encode will encode this command into a wire message for the given server description.
func (kc *KillCursors) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	idVals := make([]*bson.Value, 0, len(kc.IDs))
	for _, id := range kc.IDs {
		idVals = append(idVals, bson.VC.Int64(id))
	}
	cmd := bson.NewDocument(
		bson.EC.String("killCursors", kc.NS.Collection),
		bson.EC.ArrayFromElements("cursors", idVals...),
	)
	return (&Command{DB: kc.NS.DB, Command: cmd}).Encode(desc)
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (kc *KillCursors) Decode(desc description.SelectedServer, wm wiremessage.WireMessage) *KillCursors {
	rdr, err := (&Command{}).Decode(desc, wm).Result()
	if err != nil {
		kc.err = err
		return kc
	}

	err = bson.Unmarshal(rdr, &kc.result)
	if err != nil {
		kc.err = err
		return kc
	}
	return kc
}

// Result returns the result of a decoded wire message and server description.
func (kc *KillCursors) Result() (result.KillCursors, error) {
	if kc.err != nil {
		return result.KillCursors{}, kc.err
	}

	return kc.result, nil
}

// Err returns the error set on this command.
func (kc *KillCursors) Err() error { return kc.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (kc *KillCursors) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriter) (result.KillCursors, error) {
	wm, err := kc.Encode(desc)
	if err != nil {
		return result.KillCursors{}, err
	}

	err = rw.WriteWireMessage(ctx, wm)
	if err != nil {
		return result.KillCursors{}, err
	}
	wm, err = rw.ReadWireMessage(ctx)
	if err != nil {
		return result.KillCursors{}, err
	}
	return kc.Decode(desc, wm).Result()
}
