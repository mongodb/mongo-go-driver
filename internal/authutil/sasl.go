// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package authutil

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
)

const defaultAuthDB = "admin"

// Error is an error that occurred during authentication.
type Error struct {
	message string
	inner   error
}

func (e *Error) Error() string {
	if e.inner == nil {
		return e.message
	}
	return fmt.Sprintf("%s: %s", e.message, e.inner)
}

// Inner returns the wrapped error.
func (e *Error) Inner() error {
	return e.inner
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.inner
}

// Message returns the message.
func (e *Error) Message() string {
	return e.message
}

func newError(err error, mech string) error {
	return &Error{
		message: fmt.Sprintf("unable to authenticate using mechanism \"%s\"", mech),
		inner:   err,
	}
}

// SaslClient is the client piece of a sasl conversation.
type SaslClient interface {
	Start() (string, []byte, error)
	Next(challenge []byte) ([]byte, error)
	Completed() bool
}

// SaslClientCloser is a SaslClient that has resources to clean up.
type SaslClientCloser interface {
	SaslClient
	Close()
}

// ExtraOptionsSaslClient is a SaslClient that appends options to the saslStart command.
type ExtraOptionsSaslClient interface {
	StartCommandOptions() bsoncore.Document
}

// saslConversation represents a SASL conversation. This type implements the SpeculativeConversation interface so the
// conversation can be executed in multi-step speculative fashion.
type saslConversation struct {
	client      SaslClient
	source      string
	mechanism   string
	speculative bool
}

func newSaslConversation(client SaslClient, source string, speculative bool) *saslConversation {
	authSource := source
	if authSource == "" {
		authSource = defaultAuthDB
	}
	return &saslConversation{
		client:      client,
		source:      authSource,
		speculative: speculative,
	}
}

// FirstMessage returns the first message to be sent to the server. This message contains a "db" field so it can be used
// for speculative authentication.
func (sc *saslConversation) FirstMessage() (bsoncore.Document, error) {
	var payload []byte
	var err error
	sc.mechanism, payload, err = sc.client.Start()
	if err != nil {
		return nil, err
	}

	saslCmdElements := [][]byte{
		bsoncore.AppendInt32Element(nil, "saslStart", 1),
		bsoncore.AppendStringElement(nil, "mechanism", sc.mechanism),
		bsoncore.AppendBinaryElement(nil, "payload", 0x00, payload),
	}
	if sc.speculative {
		// The "db" field is only appended for speculative auth because the hello command is executed against admin
		// so this is needed to tell the server the user's auth source. For a non-speculative attempt, the SASL commands
		// will be executed against the auth source.
		saslCmdElements = append(saslCmdElements, bsoncore.AppendStringElement(nil, "db", sc.source))
	}
	if extraOptionsClient, ok := sc.client.(ExtraOptionsSaslClient); ok {
		optionsDoc := extraOptionsClient.StartCommandOptions()
		saslCmdElements = append(saslCmdElements, bsoncore.AppendDocumentElement(nil, "options", optionsDoc))
	}

	return bsoncore.BuildDocumentFromElements(nil, saslCmdElements...), nil
}

type saslResponse struct {
	ConversationID int    `bson:"conversationId"`
	Code           int    `bson:"code"`
	Done           bool   `bson:"done"`
	Payload        []byte `bson:"payload"`
}

// Finish completes the conversation based on the first server response to authenticate the given connection.
func (sc *saslConversation) Finish(ctx context.Context, cfg *Config, firstResponse bsoncore.Document) error {
	if closer, ok := sc.client.(SaslClientCloser); ok {
		defer closer.Close()
	}

	var saslResp saslResponse
	err := bson.Unmarshal(firstResponse, &saslResp)
	if err != nil {
		fullErr := fmt.Errorf("unmarshal error: %w", err)
		return newError(fullErr, sc.mechanism)
	}

	cid := saslResp.ConversationID
	var payload []byte
	var rdr bsoncore.Document
	for {
		if saslResp.Code != 0 {
			return newError(err, sc.mechanism)
		}

		if saslResp.Done && sc.client.Completed() {
			return nil
		}

		payload, err = sc.client.Next(saslResp.Payload)
		if err != nil {
			return newError(err, sc.mechanism)
		}

		if saslResp.Done && sc.client.Completed() {
			return nil
		}

		doc := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendInt32Element(nil, "saslContinue", 1),
			bsoncore.AppendInt32Element(nil, "conversationId", int32(cid)),
			bsoncore.AppendBinaryElement(nil, "payload", 0x00, payload),
		)
		saslContinueCmd := operation.NewCommand(doc).
			Database(sc.source).
			Deployment(driver.SingleConnectionDeployment{cfg.Connection}).
			ClusterClock(cfg.ClusterClock).
			ServerAPI(cfg.ServerAPI)

		err = saslContinueCmd.Execute(ctx)
		if err != nil {
			return newError(err, sc.mechanism)
		}
		rdr = saslContinueCmd.Result()

		err = bson.Unmarshal(rdr, &saslResp)
		if err != nil {
			fullErr := fmt.Errorf("unmarshal error: %w", err)
			return newError(fullErr, sc.mechanism)
		}
	}
}

// Config holds the information necessary to perform an authentication attempt.
type Config struct {
	Description   description.Server
	Connection    driver.Connection
	ClusterClock  *session.ClusterClock
	HandshakeInfo driver.HandshakeInformation
	ServerAPI     *driver.ServerAPIOptions
}

// ConductSaslConversation runs a full SASL conversation to authenticate the given connection.
func ConductSaslConversation(ctx context.Context, cfg *Config, authSource string, client SaslClient) error {
	// Create a non-speculative SASL conversation.
	conversation := newSaslConversation(client, authSource, false)

	saslStartDoc, err := conversation.FirstMessage()
	if err != nil {
		return newError(err, conversation.mechanism)
	}
	saslStartCmd := operation.NewCommand(saslStartDoc).
		Database(authSource).
		Deployment(driver.SingleConnectionDeployment{cfg.Connection}).
		ClusterClock(cfg.ClusterClock).
		ServerAPI(cfg.ServerAPI)
	if err := saslStartCmd.Execute(ctx); err != nil {
		return newError(err, conversation.mechanism)
	}

	return conversation.Finish(ctx, cfg, saslStartCmd.Result())
}
