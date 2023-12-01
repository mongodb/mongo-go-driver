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

// ReceivedMessage represents a message received from the server.
type ReceivedMessage struct {
	ResponseTo int32
	RawMessage wiremessage.WireMessage
	Response   bsoncore.Document
}

type receivedMsgParseFn func([]byte) (*ReceivedMessage, error)

func getReceivedMessageParser(opcode wiremessage.OpCode) (receivedMsgParseFn, bool) {
	switch opcode {
	case wiremessage.OpReply:
		return parseOpReply, true
	case wiremessage.OpMsg:
		return parseReceivedOpMsg, true
	case wiremessage.OpCompressed:
		return parseReceivedOpCompressed, true
	default:
		return nil, false
	}
}

func parseReceivedMessage(wm []byte) (*ReceivedMessage, error) {
	// Re-assign the wire message to "remaining" so "wm" continues to point to the entire message after parsing.
	_, _, responseTo, opcode, remaining, ok := wiremessage.ReadHeader(wm)
	if !ok {
		return nil, errors.New("failed to read wiremessage header")
	}

	parseFn, ok := getReceivedMessageParser(opcode)
	if !ok {
		return nil, fmt.Errorf("unknown opcode: %s", opcode)
	}
	received, err := parseFn(remaining)
	if err != nil {
		return nil, fmt.Errorf("error parsing wiremessage with opcode %s: %w", opcode, err)
	}

	received.ResponseTo = responseTo
	received.RawMessage = wm
	return received, nil
}

func parseOpReply(wm []byte) (*ReceivedMessage, error) {
	var ok bool

	if _, wm, ok = wiremessage.ReadReplyFlags(wm); !ok {
		return nil, errors.New("failed to read reply flags")
	}
	if _, wm, ok = wiremessage.ReadReplyCursorID(wm); !ok {
		return nil, errors.New("failed to read cursor ID")
	}
	if _, wm, ok = wiremessage.ReadReplyStartingFrom(wm); !ok {
		return nil, errors.New("failed to read starting from")
	}
	if _, wm, ok = wiremessage.ReadReplyNumberReturned(wm); !ok {
		return nil, errors.New("failed to read number returned")
	}

	var replyDocuments []bsoncore.Document
	replyDocuments, wm, ok = wiremessage.ReadReplyDocuments(wm)
	if !ok {
		return nil, errors.New("failed to read reply documents")
	}
	if len(replyDocuments) == 0 {
		return nil, errors.New("no documents in response")
	}

	rm := &ReceivedMessage{
		Response: replyDocuments[0],
	}
	return rm, nil
}

func parseReceivedOpMsg(wm []byte) (*ReceivedMessage, error) {
	var ok bool
	var err error

	if _, wm, ok = wiremessage.ReadMsgFlags(wm); !ok {
		return nil, errors.New("failed to read flags")
	}

	if wm, err = assertMsgSectionType(wm, wiremessage.SingleDocument); err != nil {
		return nil, fmt.Errorf("error verifying section type for response document: %w", err)
	}

	response, wm, ok := wiremessage.ReadMsgSectionSingleDocument(wm)
	if !ok {
		return nil, errors.New("failed to read response document")
	}
	rm := &ReceivedMessage{
		Response: response,
	}
	return rm, nil
}

func parseReceivedOpCompressed(wm []byte) (*ReceivedMessage, error) {
	originalOpcode, wm, err := parseOpCompressed(wm)
	if err != nil {
		return nil, err
	}

	parser, ok := getReceivedMessageParser(originalOpcode)
	if !ok {
		return nil, fmt.Errorf("unknown original opcode %v", originalOpcode)
	}
	return parser(wm)
}
