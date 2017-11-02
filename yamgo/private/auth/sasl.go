// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"context"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/yamgo/model"

	"github.com/10gen/mongo-go-driver/yamgo/private/conn"
	"github.com/10gen/mongo-go-driver/yamgo/private/msg"
)

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

// ConductSaslConversation handles running a sasl conversation with MongoDB.
func ConductSaslConversation(ctx context.Context, c conn.Connection, db string, client SaslClient) error {

	// Arbiters cannot be authenticated
	if c.Model().Kind == model.RSArbiter {
		return nil
	}

	if db == "" {
		db = defaultAuthDB
	}

	if closer, ok := client.(SaslClientCloser); ok {
		defer closer.Close()
	}

	mech, payload, err := client.Start()
	if err != nil {
		return newError(err, mech)
	}

	saslStartRequest := msg.NewCommand(
		msg.NextRequestID(),
		db,
		true,
		bson.D{
			{Name: "saslStart", Value: 1},
			{Name: "mechanism", Value: mech},
			{Name: "payload", Value: payload},
		},
	)

	type saslResponse struct {
		ConversationID int    `bson:"conversationId"`
		Code           int    `bson:"code"`
		Done           bool   `bson:"done"`
		Payload        []byte `bson:"payload"`
	}

	var saslResp saslResponse
	err = conn.ExecuteCommand(ctx, c, saslStartRequest, &saslResp)
	if err != nil {
		return newError(err, mech)
	}

	cid := saslResp.ConversationID

	for {
		if saslResp.Code != 0 {
			return newError(err, mech)
		}

		if saslResp.Done && client.Completed() {
			return nil
		}

		payload, err = client.Next(saslResp.Payload)
		if err != nil {
			return newError(err, mech)
		}

		if saslResp.Done && client.Completed() {
			return nil
		}

		saslContinueRequest := msg.NewCommand(
			msg.NextRequestID(),
			db,
			true,
			bson.D{
				{Name: "saslContinue", Value: 1},
				{Name: "conversationId", Value: cid},
				{Name: "payload", Value: payload},
			},
		)

		err = conn.ExecuteCommand(ctx, c, saslContinueRequest, &saslResp)
		if err != nil {
			return newError(err, mech)
		}
	}
}
