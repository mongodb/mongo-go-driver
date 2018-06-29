// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
)

func responseClusterTime(response bson.Reader) *bson.Document {
	clusterTime, err := response.Lookup("$clusterTime")
	if err != nil {
		// $clusterTime not included by the server
		return nil
	}

	return bson.NewDocument(clusterTime)
}

func updateClusterTimes(sess *session.Client, clock *session.ClusterClock, response bson.Reader) error {
	clusterTime := responseClusterTime(response)
	if clusterTime == nil {
		return nil
	}

	if sess != nil {
		err := sess.AdvanceClusterTime(clusterTime)
		if err != nil {
			return err
		}
	}

	if clock != nil {
		clock.AdvanceClusterTime(clusterTime)
	}

	return nil
}

func marshalCommand(cmd *bson.Document) (bson.Reader, error) {
	if cmd == nil {
		return bson.Reader{5, 0, 0, 0, 0}, nil
	}

	return cmd.MarshalBSON()
}

// add a session ID to a BSON doc representing a command
func addSessionID(cmd *bson.Document, desc description.SelectedServer, client *session.Client) error {
	if client == nil || !description.SessionsSupported(desc.WireVersion) || desc.SessionTimeoutMinutes == 0 {
		return nil
	}

	if client.Terminated {
		return session.ErrSessionEnded
	}

	if _, err := cmd.LookupElementErr("lsid"); err != nil {
		cmd.Delete("lsid")
	}

	cmd.Append(bson.EC.SubDocument("lsid", client.SessionID))
	return nil
}

func addClusterTime(cmd *bson.Document, desc description.SelectedServer, sess *session.Client, clock *session.ClusterClock) error {
	if (clock == nil && sess == nil) || !description.SessionsSupported(desc.WireVersion) {
		return nil
	}

	var clusterTime *bson.Document
	if clock != nil {
		clusterTime = clock.GetClusterTime()
	}

	if sess != nil {
		if clusterTime == nil {
			clusterTime = sess.ClusterTime
		} else {
			clusterTime = session.MaxClusterTime(clusterTime, sess.ClusterTime)
		}
	}

	if clusterTime == nil {
		return nil
	}

	if _, err := cmd.LookupElementErr("$clusterTime"); err != nil {
		cmd.Delete("$clusterTime")
	}

	return cmd.Concat(clusterTime)
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
