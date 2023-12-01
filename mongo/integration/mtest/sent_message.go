// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mtest

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

// SentMessage represents a message sent by the driver to the server.
type SentMessage struct {
	RequestID  int32
	RawMessage wiremessage.WireMessage
	Command    bsoncore.Document
	OpCode     wiremessage.OpCode

	// The $readPreference document. This is separated into its own field even though it's included in the larger
	// command document in both OP_QUERY and OP_MSG because OP_QUERY separates the command into a $query sub-document
	// if there is a read preference. To unify OP_QUERY and OP_MSG, we pull this out into a separate field and set
	// the Command field to the $query sub-document.
	ReadPreference bsoncore.Document

	// The documents sent for an insert, update, or delete command. This is separated into its own field because it's
	// sent as part of the command document in OP_QUERY and as a document sequence outside the command document in
	// OP_MSG.
	DocumentSequence *bsoncore.DocumentSequence
}

type sentMsgParseFn func([]byte) (*SentMessage, error)

func getSentMessageParser(opcode wiremessage.OpCode) (sentMsgParseFn, bool) {
	switch opcode {
	case wiremessage.OpQuery:
		return parseOpQuery, true
	case wiremessage.OpMsg:
		return parseSentOpMsg, true
	case wiremessage.OpCompressed:
		return parseSentOpCompressed, true
	default:
		return nil, false
	}
}

func parseOpQuery(wm []byte) (*SentMessage, error) {
	var ok bool

	if _, wm, ok = wiremessage.ReadQueryFlags(wm); !ok {
		return nil, errors.New("failed to read query flags")
	}
	if _, wm, ok = wiremessage.ReadQueryFullCollectionName(wm); !ok {
		return nil, errors.New("failed to read full collection name")
	}
	if _, wm, ok = wiremessage.ReadQueryNumberToSkip(wm); !ok {
		return nil, errors.New("failed to read number to skip")
	}
	if _, wm, ok = wiremessage.ReadQueryNumberToReturn(wm); !ok {
		return nil, errors.New("failed to read number to return")
	}

	query, wm, ok := wiremessage.ReadQueryQuery(wm)
	if !ok {
		return nil, errors.New("failed to read query")
	}

	// If there is no read preference document, the command document is query.
	// Otherwise, query is in the format {$query: <command document>, $readPreference: <read preference document>}.
	commandDoc := query
	var rpDoc bsoncore.Document

	dollarQueryVal, err := query.LookupErr("$query")
	if err == nil {
		commandDoc = dollarQueryVal.Document()

		rpVal, err := query.LookupErr("$readPreference")
		if err != nil {
			return nil, fmt.Errorf("query %s contains $query but not $readPreference fields", query)
		}
		rpDoc = rpVal.Document()
	}

	// For OP_QUERY, inserts, updates, and deletes are sent as a BSON array of documents inside the main command
	// document. Pull these sequences out into an ArrayStyle DocumentSequence.
	var docSequence *bsoncore.DocumentSequence
	cmdElems, _ := commandDoc.Elements()
	for _, elem := range cmdElems {
		switch elem.Key() {
		case "documents", "updates", "deletes":
			docSequence = &bsoncore.DocumentSequence{
				Style: bsoncore.ArrayStyle,
				Data:  elem.Value().Array(),
			}
		}
		if docSequence != nil {
			// There can only be one of these arrays in a well-formed command, so we exit the loop once one is found.
			break
		}
	}

	sm := &SentMessage{
		Command:          commandDoc,
		ReadPreference:   rpDoc,
		DocumentSequence: docSequence,
	}
	return sm, nil
}

func parseSentMessage(wm []byte) (*SentMessage, error) {
	// Re-assign the wire message to "remaining" so "wm" continues to point to the entire message after parsing.
	_, requestID, _, opcode, remaining, ok := wiremessage.ReadHeader(wm)
	if !ok {
		return nil, errors.New("failed to read wiremessage header")
	}

	parseFn, ok := getSentMessageParser(opcode)
	if !ok {
		return nil, fmt.Errorf("unknown opcode: %v", opcode)
	}
	sent, err := parseFn(remaining)
	if err != nil {
		return nil, fmt.Errorf("error parsing wiremessage with opcode %s: %w", opcode, err)
	}

	sent.RequestID = requestID
	sent.RawMessage = wm
	sent.OpCode = opcode
	return sent, nil
}

func parseSentOpMsg(wm []byte) (*SentMessage, error) {
	var ok bool
	var err error

	if _, wm, ok = wiremessage.ReadMsgFlags(wm); !ok {
		return nil, errors.New("failed to read flags")
	}

	if wm, err = assertMsgSectionType(wm, wiremessage.SingleDocument); err != nil {
		return nil, fmt.Errorf("error verifying section type for command document: %w", err)
	}

	var commandDoc bsoncore.Document
	commandDoc, wm, ok = wiremessage.ReadMsgSectionSingleDocument(wm)
	if !ok {
		return nil, errors.New("failed to read command document")
	}

	var rpDoc bsoncore.Document
	if rpVal, err := commandDoc.LookupErr("$readPreference"); err == nil {
		rpDoc = rpVal.Document()
	}

	var docSequence *bsoncore.DocumentSequence
	if len(wm) != 0 {
		// If there are bytes remaining in the wire message, they must correspond to a DocumentSequence section.
		if wm, err = assertMsgSectionType(wm, wiremessage.DocumentSequence); err != nil {
			return nil, fmt.Errorf("error verifying section type for document sequence: %w", err)
		}

		var data []byte
		_, data, wm, ok = wiremessage.ReadMsgSectionRawDocumentSequence(wm)
		if !ok {
			return nil, errors.New("failed to read document sequence")
		}

		docSequence = &bsoncore.DocumentSequence{
			Style: bsoncore.SequenceStyle,
			Data:  data,
		}
	}

	sm := &SentMessage{
		Command:          commandDoc,
		ReadPreference:   rpDoc,
		DocumentSequence: docSequence,
	}
	return sm, nil
}

func parseSentOpCompressed(wm []byte) (*SentMessage, error) {
	originalOpcode, wm, err := parseOpCompressed(wm)
	if err != nil {
		return nil, err
	}

	parser, ok := getSentMessageParser(originalOpcode)
	if !ok {
		return nil, fmt.Errorf("unknown original opcode %v", originalOpcode)
	}
	return parser(wm)
}
