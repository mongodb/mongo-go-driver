// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"bytes"
	"context"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/builder"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

// Command represents a generic database command.
//
// This can be used to send arbitrary commands to the database, e.g. runCommand.
type Command struct {
	DB       string
	Command  interface{}
	ReadPref *readpref.ReadPref
	isWrite  bool

	result bson.Reader
	err    error
}

// Encode will encode this command into a wire message for the given server description.
func (c *Command) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	// TODO(skriptble): When we do OP_MSG support we'll have to update this to read the
	// wire version.
	rdr, err := c.marshalCommand()
	if err != nil {
		return nil, err
	}

	var flags wiremessage.QueryFlag
	if desc.Kind == description.Single && desc.Server.Kind != description.Mongos {
		flags |= wiremessage.SlaveOK
	}

	if !c.isWrite {
		rdr, err = c.addReadPref(c.ReadPref, desc.Server.Kind, rdr)
		if err != nil {
			return nil, err
		}

		if c.ReadPref == nil || c.ReadPref.Mode() == readpref.PrimaryMode {
			// assume primary
			flags &^= wiremessage.SlaveOK
		}
	}

	query := wiremessage.Query{
		MsgHeader:          wiremessage.Header{RequestID: wiremessage.NextRequestID()},
		FullCollectionName: c.DB + ".$cmd",
		Flags:              flags,
		NumberToReturn:     -1,
		Query:              rdr,
	}

	return query, nil
}

// addReadPref will add a read preference to the query document.
//
// NOTE: This method must always return either a valid bson.Reader or an error.
func (c *Command) addReadPref(rp *readpref.ReadPref, kind description.ServerKind, query bson.Reader) (bson.Reader, error) {
	if kind != description.Mongos || rp == nil {
		return query, nil
	}

	// simple Primary or SecondaryPreferred is communicated via slaveOk to Mongos.
	if rp.Mode() == readpref.PrimaryMode || rp.Mode() == readpref.SecondaryPreferredMode {
		if _, ok := rp.MaxStaleness(); !ok && len(rp.TagSets()) == 0 {
			return query, nil
		}
	}

	doc := bson.NewDocument()

	switch rp.Mode() {
	case readpref.PrimaryMode:
		doc.Append(bson.EC.String("mode", "primary"))
	case readpref.PrimaryPreferredMode:
		doc.Append(bson.EC.String("mode", "primaryPreferred"))
	case readpref.SecondaryPreferredMode:
		doc.Append(bson.EC.String("mode", "secondaryPreferred"))
	case readpref.SecondaryMode:
		doc.Append(bson.EC.String("mode", "secondary"))
	case readpref.NearestMode:
		doc.Append(bson.EC.String("mode", "nearest"))
	}

	sets := make([]*bson.Value, 0, len(rp.TagSets()))
	for _, ts := range rp.TagSets() {
		if len(ts) == 0 {
			continue
		}
		set := bson.NewDocument()
		for _, t := range ts {
			set.Append(bson.EC.String(t.Name, t.Value))
		}
		sets = append(sets, bson.VC.Document(set))
	}
	if len(sets) > 0 {
		doc.Append(bson.EC.ArrayFromElements("tags", sets...))
	}

	if d, ok := rp.MaxStaleness(); ok {
		doc.Append(bson.EC.Int32("maxStalenessSeconds", int32(d.Seconds())))
	}

	return bson.NewDocument(
		bson.EC.SubDocumentFromReader("$query", query),
		bson.EC.SubDocument("$readPreference", doc),
	).MarshalBSON()
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (c *Command) Decode(_ description.SelectedServer, wm wiremessage.WireMessage) *Command {
	reply, ok := wm.(wiremessage.Reply)
	if !ok {
		c.err = fmt.Errorf("unsupported response wiremessage type %T", wm)
		return c
	}
	c.result, c.err = decodeCommandOpReply(reply)
	return c
}

// Result returns the result of a decoded wire message and server description.
func (c *Command) Result() (bson.Reader, error) {
	if c.err != nil {
		return nil, c.err
	}

	return c.result, nil
}

// Err returns the error set on this command.
func (c *Command) Err() error { return c.err }

// RoundTrip handles the execution of this command using the provided wiremessage.ReadWriter.
func (c *Command) RoundTrip(ctx context.Context, desc description.SelectedServer, rw wiremessage.ReadWriter) (bson.Reader, error) {
	wm, err := c.Encode(desc)
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
	return c.Decode(desc, wm).Result()
}

func (c *Command) marshalCommand() (bson.Reader, error) {
	if c.Command == nil {
		return bson.Reader{5, 0, 0, 0, 0}, nil
	}

	var dataBytes bson.Reader
	var err error

	switch t := c.Command.(type) {
	// NOTE: bson.Document is covered by bson.Marshaler
	case bson.Marshaler:
		dataBytes, err = t.MarshalBSON()
		if err != nil {
			return nil, err
		}
	case bson.Reader:
		_, err = t.Validate()
		if err != nil {
			return nil, err
		}
		dataBytes = t
	case builder.DocumentBuilder:
		dataBytes = make([]byte, t.RequiredBytes())
		_, err = t.WriteDocument(dataBytes)
		if err != nil {
			return nil, err
		}
	case []byte:
		_, err = bson.Reader(t).Validate()
		if err != nil {
			return nil, err
		}
		dataBytes = t
	default:
		var buf bytes.Buffer
		err = bson.NewEncoder(&buf).Encode(c.Command)
		if err != nil {
			return nil, err
		}
		dataBytes = buf.Bytes()
	}

	return dataBytes, nil
}
