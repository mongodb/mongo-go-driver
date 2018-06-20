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
	Acknowledged bool
	DB           string
	Command      interface{}
	ReadPref     *readpref.ReadPref
	isWrite      bool

	result bson.Reader
	err    error
}

// Remove command arguments for insert, update, and delete commands from the BSON document so they can be encoded
// as a Section 1 payload in OP_MSG
func (c *Command) opmsgRemoveArray() (array *bson.Array, id string) {
	if converted, ok := c.Command.(*bson.Document); ok {
		keys := []string{"documents", "updates", "deletes"}

		for _, key := range keys {
			val := converted.Lookup(key)
			if val == nil {
				continue
			}

			array = val.MutableArray()
			converted.Delete(key)
			id = key
			break
		}
	}

	return
}

// Add the $db and $readPreference keys to the command
// rdr is a bson.Reader for the initial marshaled command (does not contain $db and $readPreference keys)
func (c *Command) opmsgAddGlobals(rdr bson.Reader, desc description.SelectedServer) (bson.Reader, error) {
	// reconstruct command with $db and $readPreference keys
	fullDoc := bson.NewDocument()
	rdrIter, err := rdr.Iterator()
	if err != nil {
		return nil, err
	}

	for rdrIter.Next() {
		fullDoc.Append(rdrIter.Element().Clone())
	}

	fullDoc.Append(bson.EC.String("$db", c.DB))
	readPrefDoc := c.findReadPref(c.ReadPref, desc.Server.Kind)
	if readPrefDoc != nil {
		fullDoc.Append(bson.EC.SubDocument("$readPreference", readPrefDoc))
	}

	fullDocRdr, err := fullDoc.MarshalBSON()
	if err != nil {
		return nil, err
	}

	return fullDocRdr, nil
}

func (c *Command) opmsgCreateDocSequence(arr *bson.Array, identifier string) (wiremessage.SectionDocumentSequence, error) {
	docSequence := wiremessage.SectionDocumentSequence{
		PayloadType: wiremessage.DocumentSequence,
		Identifier:  identifier,
		Documents:   make([]bson.Reader, 0, arr.Len()),
	}

	iter, err := arr.Iterator()
	if err != nil {
		return wiremessage.SectionDocumentSequence{}, err
	}

	for iter.Next() {
		docSequence.Documents = append(docSequence.Documents, iter.Value().ReaderDocument())
	}

	docSequence.Size = int32(docSequence.PayloadLen())
	return docSequence, nil
}

// Encode c as OP_MSG
func (c *Command) encodeOpMsg(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	arr, identifier := c.opmsgRemoveArray()

	rdr, err := c.marshalCommand()
	if err != nil {
		return nil, err
	}

	msg := wiremessage.Msg{
		MsgHeader: wiremessage.Header{RequestID: wiremessage.NextRequestID()},
		Sections:  make([]wiremessage.Section, 0),
	}

	fullDocRdr, err := c.opmsgAddGlobals(rdr, desc)
	if err != nil {
		return nil, err
	}

	// type 0 doc
	msg.Sections = append(msg.Sections, wiremessage.SectionBody{
		PayloadType: wiremessage.SingleDocument,
		Document:    fullDocRdr,
	})

	// type 1 doc
	if identifier != "" {
		docSequence, err := c.opmsgCreateDocSequence(arr, identifier)
		if err != nil {
			return nil, err
		}

		msg.Sections = append(msg.Sections, docSequence)
	}

	// flags
	if c.isWrite && !c.Acknowledged {
		msg.FlagBits |= wiremessage.MoreToCome
	}

	return msg, nil
}

// Encode c as OP_QUERY
func (c *Command) encodeOpQuery(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	rdr, err := c.marshalCommand()
	if err != nil {
		return nil, err
	}

	if !c.isWrite {
		rdr, err = c.addReadPref(c.ReadPref, desc.Server.Kind, rdr)
		if err != nil {
			return nil, err
		}
	}

	query := wiremessage.Query{
		MsgHeader:          wiremessage.Header{RequestID: wiremessage.NextRequestID()},
		FullCollectionName: c.DB + ".$cmd",
		Flags:              c.slaveOK(desc),
		NumberToReturn:     -1,
		Query:              rdr,
	}

	return query, nil
}

// Encode will encode this command into a wire message for the given server description.
func (c *Command) Encode(desc description.SelectedServer) (wiremessage.WireMessage, error) {
	if desc.WireVersion == nil || desc.WireVersion.Max < wiremessage.OpmsgWireVersion {
		return c.encodeOpQuery(desc)
	}

	return c.encodeOpMsg(desc)
}

func (c *Command) slaveOK(desc description.SelectedServer) wiremessage.QueryFlag {
	if desc.Kind == description.Single && desc.Server.Kind != description.Mongos {
		return wiremessage.SlaveOK
	}

	if c.ReadPref == nil {
		// assume primary
		return 0
	}

	if c.ReadPref.Mode() != readpref.PrimaryMode {
		return wiremessage.SlaveOK
	}

	return 0
}

func (c *Command) findReadPref(rp *readpref.ReadPref, kind description.ServerKind) *bson.Document {
	if kind != description.Mongos || rp == nil {
		return nil
	}

	// simple Primary or SecondaryPreferred is communicated via slaveOk to Mongos.
	if rp.Mode() == readpref.PrimaryMode || rp.Mode() == readpref.SecondaryPreferredMode {
		if _, ok := rp.MaxStaleness(); !ok && len(rp.TagSets()) == 0 {
			return nil
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

	return doc
}

// addReadPref will add a read preference to the query document.
//
// NOTE: This method must always return either a valid bson.Reader or an error.
func (c *Command) addReadPref(rp *readpref.ReadPref, kind description.ServerKind, query bson.Reader) (bson.Reader, error) {
	doc := c.findReadPref(rp, kind)
	if doc == nil {
		return query, nil
	}

	return bson.NewDocument(
		bson.EC.SubDocumentFromReader("$query", query),
		bson.EC.SubDocument("$readPreference", doc),
	).MarshalBSON()
}

func (c *Command) decodeOpMsg(wm wiremessage.WireMessage) *Command {
	msg, ok := wm.(wiremessage.Msg)
	if !ok {
		c.err = fmt.Errorf("unsupported response wiremessage type %T", wm)
		return c
	}

	c.result, c.err = decodeCommandOpMsg(msg)
	return c
}

func (c *Command) decodeOpReply(wm wiremessage.WireMessage) *Command {
	reply, ok := wm.(wiremessage.Reply)
	if !ok {
		c.err = fmt.Errorf("unsupported response wiremessage type %T", wm)
		return c
	}
	c.result, c.err = decodeCommandOpReply(reply)
	return c
}

// Decode will decode the wire message using the provided server description. Errors during decoding
// are deferred until either the Result or Err methods are called.
func (c *Command) Decode(desc description.SelectedServer, wm wiremessage.WireMessage) *Command {
	switch wm.(type) {
	case wiremessage.Reply:
		return c.decodeOpReply(wm)
	default:
		return c.decodeOpMsg(wm)
	}
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
