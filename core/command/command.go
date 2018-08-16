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

// DecodeError attempts to decode the wiremessage as an error
func DecodeError(wm wiremessage.WireMessage) error {
	var rdr bson.Reader
	switch msg := wm.(type) {
	case wiremessage.Msg:
		for _, section := range msg.Sections {
			switch converted := section.(type) {
			case wiremessage.SectionBody:
				rdr = converted.Document
			}
		}
	case wiremessage.Reply:
		if msg.ResponseFlags&wiremessage.QueryFailure != wiremessage.QueryFailure {
			return nil
		}
		rdr = msg.Documents[0]
	}

	_, err := rdr.Validate()
	if err != nil {
		return nil
	}

	extractedError := extractError(rdr)

	// If parsed successfully return the error
	if _, ok := extractedError.(Error); ok {
		return err
	}

	return nil
}

// helper method to extract an error from a reader if there is one; first returned item is the
// error if it exists, the second holds parsing errors
func extractError(rdr bson.Reader) error {
	var errmsg, codeName string
	var code int32
	var labels []string
	itr, err := rdr.Iterator()
	if err != nil {
		return err
	}

	for itr.Next() {
		elem := itr.Element()
		switch elem.Key() {
		case "ok":
			switch elem.Value().Type() {
			case bson.TypeInt32:
				if elem.Value().Int32() == 1 {
					return nil
				}
			case bson.TypeInt64:
				if elem.Value().Int64() == 1 {
					return nil
				}
			case bson.TypeDouble:
				if elem.Value().Double() == 1 {
					return nil
				}
			}
		case "errmsg":
			if str, okay := elem.Value().StringValueOK(); okay {
				errmsg = str
			}
		case "codeName":
			if str, okay := elem.Value().StringValueOK(); okay {
				codeName = str
			}
		case "code":
			if c, okay := elem.Value().Int32OK(); okay {
				code = c
			}
		case "errorLabels":
			if arr, okay := elem.Value().MutableArrayOK(); okay {
				iter, err := arr.Iterator()
				if err != nil {
					continue
				}
				for iter.Next() {
					if str, ok := iter.Value().StringValueOK(); ok {
						labels = append(labels, str)
					}
				}

			}
		}
	}

	if errmsg == "" {
		errmsg = "command failed"
	}

	return Error{
		Code:    code,
		Message: errmsg,
		Name:    codeName,
		Labels:  labels,
	}
}

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

func updateOperationTime(sess *session.Client, response bson.Reader) error {
	if sess == nil {
		return nil
	}

	opTimeElem, err := response.Lookup("operationTime")
	if err != nil {
		// operationTime not included by the server
		return nil
	}

	t, i := opTimeElem.Value().Timestamp()
	return sess.AdvanceOperationTime(&bson.Timestamp{
		T: t,
		I: i,
	})
}

func marshalCommand(cmd *bson.Document) (bson.Reader, error) {
	if cmd == nil {
		return bson.Reader{5, 0, 0, 0, 0}, nil
	}

	return cmd.MarshalBSON()
}

// adds session related fields to a BSON doc representing a command
func addSessionFields(cmd *bson.Document, desc description.SelectedServer, client *session.Client) error {
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

	if client.TransactionRunning() ||
		client.RetryingCommit {
		addTransaction(cmd, client)
	}

	client.ApplyCommand() // advance the state machine based on a command executing

	return nil
}

// if in a transaction, add the transaction fields
func addTransaction(cmd *bson.Document, client *session.Client) {
	cmd.Append(bson.EC.Int64("txnNumber", client.TxnNumber))
	if client.TransactionStarting() {
		// When starting transaction, always transition to the next state, even on error
		cmd.Append(bson.EC.Boolean("startTransaction", true))
	}
	cmd.Append(bson.EC.Boolean("autocommit", false))
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
func addReadConcern(cmd *bson.Document, desc description.SelectedServer, rc *readconcern.ReadConcern, sess *session.Client) error {
	// Starting transaction's read concern overrides all others
	if sess != nil && sess.TransactionStarting() && sess.CurrentRc != nil {
		rc = sess.CurrentRc
	}

	// start transaction must append afterclustertime IF causally consistent and operation time exists
	if rc == nil && sess != nil && sess.TransactionStarting() && sess.Consistent && sess.OperationTime != nil {
		rc = readconcern.New()
	}

	if rc == nil {
		return nil
	}

	element, err := rc.MarshalBSONElement()
	if err != nil {
		return err
	}

	rcDoc := element.Value().MutableDocument()
	if description.SessionsSupported(desc.WireVersion) && sess != nil && sess.Consistent && sess.OperationTime != nil {
		rcDoc = rcDoc.Append(
			bson.EC.Timestamp("afterClusterTime", sess.OperationTime.T, sess.OperationTime.I),
		)
	}

	if _, err := cmd.LookupElementErr(element.Key()); err != nil {
		cmd.Delete(element.Key())
	}

	if rcDoc.Len() != 0 {
		cmd.Append(bson.EC.SubDocument("readConcern", rcDoc))
	}
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

// Get the error labels from a command response
func getErrorLabels(rdr *bson.Reader) ([]string, error) {
	var labels []string
	labelsElem, err := rdr.Lookup("errorLabels")
	if err != bson.ErrElementNotFound {
		return nil, err
	}
	if labelsElem != nil {
		labelsIt, err := labelsElem.Value().ReaderArray().Iterator()
		if err != nil {
			return nil, err
		}
		for labelsIt.Next() {
			labels = append(labels, labelsIt.Element().Value().StringValue())
		}
	}
	return labels, nil
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
