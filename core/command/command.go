// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

// extract cluster time from the entire server response
func responseClusterTime(response bson.Reader) (*bson.Document, error) {
	clusterTime, err := response.Lookup("$clusterTime")
	if err != nil {
		// server didn't return $clusterTime
		return nil, nil
	}

	return ResponseClusterTime(clusterTime.Value().MutableDocument())
}

// ResponseClusterTime extracts the cluster time from a server response as a bson document
// The document has 1 element with key clusterTime and a bson timestamp as the value
func ResponseClusterTime(response *bson.Document) (*bson.Document, error) {
	clusterTime, err := response.LookupErr("clusterTime")
	if err != nil {
		return nil, err
	}

	epoch, ordinal := clusterTime.Timestamp()
	return bson.NewDocument(
		bson.EC.Timestamp("clusterTime", epoch, ordinal),
	), nil
}

func marshalCommand(cmd *bson.Document) (bson.Reader, error) {
	if cmd == nil {
		return bson.Reader{5, 0, 0, 0, 0}, nil
	}

	return cmd.MarshalBSON()
}

// add a session ID to a BSON doc representing a command
func addSessionID(cmd *bson.Document, client *session.Client) error {
	if client == nil {
		return nil
	}

	if _, err := cmd.LookupElementErr("lsid"); err != nil {
		cmd.Delete("lsid")
	}

	cmd.Append(bson.EC.SubDocument("lsid", client.SessionID))
	return nil
}

// add a read concern to a BSON doc representing a command
func addReadConcern(cmd *bson.Document, rc *readconcern.ReadConcern) error {
	if rc == nil {
		return nil
	}

	element, err := rc.MarshalBSONElement()
	if err != nil {
		return err
	}

	if _, err := cmd.LookupElementErr(element.Key()); err != nil {
		cmd.Delete(element.Key())
	}

	cmd.Append(element)
	return nil
}

// add a write concern to a BSON doc representing a command
func addWriteConcern(cmd *bson.Document, wc *writeconcern.WriteConcern) error {
	if wc == nil {
		return nil
	}

	element, err := wc.MarshalBSONElement()
	if err != nil {
		return err
	}

	if _, err := cmd.LookupElementErr(element.Key()); err != nil {
		// doc already has write concern
		cmd.Delete(element.Key())
	}

	cmd.Append(element)
	return nil
}

// Remove command arguments for insert, update, and delete commands from the BSON document so they can be encoded
// as a Section 1 payload in OP_MSG
func opmsgRemoveArray(cmdDoc *bson.Document) (*bson.Array, string) {
	var array *bson.Array
	var id string

	keys := []string{"documents", "updates", "deletes"}

	for _, key := range keys {
		val := cmdDoc.Lookup(key)
		if val == nil {
			continue
		}

		array = val.MutableArray()
		cmdDoc.Delete(key)
		id = key
		break
	}

	return array, id
}

// Add the $db and $readPreference keys to the command
// If the command has no read preference, pass nil for rpDoc
func opmsgAddGlobals(cmd *bson.Document, dbName string, rpDoc *bson.Document) (bson.Reader, error) {
	cmd.Append(bson.EC.String("$db", dbName))
	if rpDoc != nil {
		cmd.Append(bson.EC.SubDocument("$readPreference", rpDoc))
	}

	fullDocRdr, err := cmd.MarshalBSON()
	if err != nil {
		return nil, err
	}

	return fullDocRdr, nil
}

func opmsgCreateDocSequence(arr *bson.Array, identifier string) (wiremessage.SectionDocumentSequence, error) {
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

// ErrUnacknowledgedWrite is returned from functions that have an unacknowledged
// write concern.
var ErrUnacknowledgedWrite = errors.New("unacknowledged write")
