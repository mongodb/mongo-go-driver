// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"context"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/result"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// IsMaster represents the isMaster command.
//
// The isMaster command is used for setting up a connection to MongoDB and
// for monitoring a MongoDB server.
//
// Since IsMaster can only be run on a connection, there is no Dispatch method.
type IsMaster struct {
	Client      *bson.Document
	Compressors []string

	err error
	res result.IsMaster
}

// Encode will encode this command into a wire message for the given server description.
func (im *IsMaster) Encode() (wiremessage.WireMessage, error) {
	cmd := bson.NewDocument(bson.EC.Int32("isMaster", 1))
	if im.Client != nil {
		cmd.Append(bson.EC.SubDocument("client", im.Client))
	}

	// always send compressors even if empty slice
	array := bson.NewArray()
	for _, compressor := range im.Compressors {
		array.Append(bson.VC.String(compressor))
	}

	cmd.Append(bson.EC.Array("compression", array))

	rdr, err := cmd.MarshalBSON()
	if err != nil {
		return nil, err
	}
	query := wiremessage.Query{
		MsgHeader:          wiremessage.Header{RequestID: wiremessage.NextRequestID()},
		FullCollectionName: "admin.$cmd",
		Flags:              wiremessage.SlaveOK,
		NumberToReturn:     -1,
		Query:              rdr,
	}
	return query, nil
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (im *IsMaster) Decode(wm wiremessage.WireMessage) *IsMaster {
	reply, ok := wm.(wiremessage.Reply)
	if !ok {
		im.err = fmt.Errorf("unsupported response wiremessage type %T", wm)
		return im
	}
	rdr, err := decodeCommandOpReply(reply)
	if err != nil {
		im.err = err
		return im
	}
	err = bson.Unmarshal(rdr, &im.res)
	if err != nil {
		im.err = err
		return im
	}
	return im
}

// Result returns the result of a decoded wire message and server description.
func (im *IsMaster) Result() (result.IsMaster, error) {
	if im.err != nil {
		return result.IsMaster{}, im.err
	}

	return im.res, nil
}

// Err returns the error set on this command.
func (im *IsMaster) Err() error { return im.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (im *IsMaster) RoundTrip(ctx context.Context, rw wiremessage.ReadWriter) (result.IsMaster, error) {
	wm, err := im.Encode()
	if err != nil {
		return result.IsMaster{}, err
	}

	err = rw.WriteWireMessage(ctx, wm)
	if err != nil {
		return result.IsMaster{}, err
	}
	wm, err = rw.ReadWireMessage(ctx)
	if err != nil {
		return result.IsMaster{}, err
	}
	return im.Decode(wm).Result()
}
