// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
)

// SaslClient is the client piece of a sasl conversation.
type SaslClient interface {
	Start(addr address.Address) ([]byte, error)
	Next(addr address.Address, challenge []byte) ([]byte, error)
	Completed() bool
	GetMechanism() string
}

// SaslClientCloser is a SaslClient that has resources to clean up.
type SaslClientCloser interface {
	SaslClient
	Close(addr address.Address)
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
	speculative bool
}

var _ SpeculativeConversation = (*saslConversation)(nil)

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

func (sc *saslConversation) GetHandshakeInformation(ctx context.Context, hello *operation.Hello, addr address.Address, c driver.Connection) (driver.HandshakeInformation, error) {
	payload, err := sc.client.Start(addr)
	if err != nil {
		return driver.HandshakeInformation{}, err
	}
	doc := sc.composeFirstMessage(payload)
	return hello.SpeculativeAuthenticate(doc).GetHandshakeInformation(ctx, addr, c)
}

// FirstMessage returns the first message to be sent to the server. This message contains a "db" field so it can be used
// for speculative authentication.
func (sc *saslConversation) composeFirstMessage(payload []byte) bsoncore.Document {
	saslCmdElements := [][]byte{
		bsoncore.AppendInt32Element(nil, "saslStart", 1),
		bsoncore.AppendStringElement(nil, "mechanism", sc.client.GetMechanism()),
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

	return bsoncore.BuildDocumentFromElements(nil, saslCmdElements...)
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
		defer closer.Close(cfg.Description.Addr)
	}

	if firstResponse == nil {
		return nil
	}

	var saslResp saslResponse
	err := bson.Unmarshal(firstResponse, &saslResp)
	if err != nil {
		fullErr := fmt.Errorf("unmarshal error: %v", err)
		return newError(fullErr, sc.client.GetMechanism())
	}

	cid := saslResp.ConversationID
	var payload []byte
	var rdr bsoncore.Document
	for {
		if saslResp.Code != 0 {
			return newError(err, sc.client.GetMechanism())
		}

		if saslResp.Done && sc.client.Completed() {
			return nil
		}

		payload, err = sc.client.Next(cfg.Description.Addr, saslResp.Payload)
		if err != nil {
			return newError(err, sc.client.GetMechanism())
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
			return newError(err, sc.client.GetMechanism())
		}
		rdr = saslContinueCmd.Result()

		err = bson.Unmarshal(rdr, &saslResp)
		if err != nil {
			fullErr := fmt.Errorf("unmarshal error: %v", err)
			return newError(fullErr, sc.client.GetMechanism())
		}
	}
}

// ConductSaslConversation runs a full SASL conversation to authenticate the given connection.
func ConductSaslConversation(ctx context.Context, cfg *Config, authSource string, client SaslClient) error {
	// Create a non-speculative SASL conversation.
	conversation := newSaslConversation(client, authSource, false)

	payload, err := conversation.client.Start(cfg.Description.Addr)
	if err != nil {
		return newError(err, conversation.client.GetMechanism())
	}
	saslStartDoc := conversation.composeFirstMessage(payload)
	saslStartCmd := operation.NewCommand(saslStartDoc).
		Database(authSource).
		Deployment(driver.SingleConnectionDeployment{C: cfg.Connection}).
		ClusterClock(cfg.ClusterClock).
		ServerAPI(cfg.ServerAPI)
	if err := saslStartCmd.Execute(ctx); err != nil {
		if closer, ok := client.(SaslClientCloser); ok {
			defer closer.Close(cfg.Description.Addr)
		}
		return newError(err, conversation.client.GetMechanism())
	}

	return conversation.Finish(ctx, cfg, saslStartCmd.Result())
}
