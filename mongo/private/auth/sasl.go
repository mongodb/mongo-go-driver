// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"bytes"
	"context"

	"github.com/mongodb/mongo-go-driver/mongo/model"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/private/conn"
	"github.com/mongodb/mongo-go-driver/mongo/private/msg"
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
		bson.NewDocument(
			bson.EC.Int32("saslStart", 1),
			bson.EC.String("mechanism", mech),
			bson.EC.Binary("payload", payload)),
	)

	type saslResponse struct {
		ConversationID int    `bson:"conversationId"`
		Code           int    `bson:"code"`
		Done           bool   `bson:"done"`
		Payload        []byte `bson:"payload"`
	}

	var saslResp saslResponse
	rdr, err := conn.ExecuteCommand(ctx, c, saslStartRequest)
	if err != nil {
		return newError(err, mech)
	}

	err = bson.NewDecoder(bytes.NewReader(rdr)).Decode(&saslResp)
	if err != nil {
		return err
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
			bson.NewDocument(
				bson.EC.Int32("saslContinue", 1),
				bson.EC.Int32("conversationId", int32(cid)),
				bson.EC.Binary("payload", payload)),
		)

		rdr, err = conn.ExecuteCommand(ctx, c, saslContinueRequest)
		if err != nil {
			return newError(err, mech)
		}

		err = bson.NewDecoder(bytes.NewReader(rdr)).Decode(&saslResp)
		if err != nil {
			return err
		}

	}
}
