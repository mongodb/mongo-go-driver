// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"bytes"
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/address"
	"go.mongodb.org/mongo-driver/x/mongo/driver/drivertest"
)

var (
	x509Response bsoncore.Document = bsoncore.BuildDocumentFromElements(nil,
		bsoncore.AppendStringElement(nil, "dbname", "$external"),
		bsoncore.AppendStringElement(nil, "user", "username"),
		bsoncore.AppendInt32Element(nil, "ok", 1),
	)
)

func TestSpeculativeX509(t *testing.T) {
	t.Run("speculative response included", func(t *testing.T) {
		// Tests for X509 when the isMaster response contains a reply to the speculative authentication attempt. The
		// driver should not send any more commands after the isMaster.

		authenticator, err := CreateAuthenticator("MONGODB-X509", &Cred{})
		assert.Nil(t, err, "CreateAuthenticator error: %v", err)
		handshaker := Handshaker(nil, &HandshakeOptions{
			Authenticator: authenticator,
		})

		numResponses := 1
		responses := make(chan []byte, numResponses)
		writeReplies(t, responses, createSpeculativeX509Handshake()...)

		conn := &drivertest.ChannelConn{
			Written:  make(chan []byte, numResponses),
			ReadResp: responses,
		}

		desc, err := handshaker.GetDescription(context.Background(), address.Address("localhost:27017"), conn)
		assert.Nil(t, err, "GetDescription error: %v", err)
		assert.NotNil(t, desc.SpeculativeAuthenticate, "desc.SpeculativeAuthenticate not set")
		conn.Desc = desc

		err = handshaker.FinishHandshake(context.Background(), conn)
		assert.Nil(t, err, "FinishHandshake error: %v", err)
		assert.Equal(t, 0, len(conn.ReadResp), "%d messages left unread", len(conn.ReadResp))

		assert.Equal(t, numResponses, len(conn.Written), "expected %d wire messages to be sent, got %d",
			numResponses, len(conn.Written))
		isMaster, err := drivertest.GetCommandFromQueryWireMessage(<-conn.Written)
		assert.Nil(t, err, "error parsing isMaster command: %v", err)
		assertCommandName(t, isMaster, "isMaster")

		authDocVal, err := isMaster.LookupErr("speculativeAuthenticate")
		assert.Nil(t, err, "expected command %s to contain 'speculativeAuthenticate'", bson.Raw(isMaster))
		authDoc := authDocVal.Document()
		expectedAuthDoc := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendInt32Element(nil, "authenticate", 1),
			bsoncore.AppendStringElement(nil, "mechanism", "MONGODB-X509"),
		)
		assert.True(t, bytes.Equal(expectedAuthDoc, authDoc), "expected speculative auth document %s, got %s",
			expectedAuthDoc, authDoc)
	})
	t.Run("speculative response not included", func(t *testing.T) {
		// Tests for X509 when the isMaster response does not contain a reply to the speculative authentication attempt.
		// The driver should send an authenticate command after the isMaster.

		authenticator, err := CreateAuthenticator("MONGODB-X509", &Cred{})
		assert.Nil(t, err, "CreateAuthenticator error: %v", err)
		handshaker := Handshaker(nil, &HandshakeOptions{
			Authenticator: authenticator,
		})

		numResponses := 2
		responses := make(chan []byte, numResponses)
		writeReplies(t, responses, createRegularX509Handshake()...)

		conn := &drivertest.ChannelConn{
			Written:  make(chan []byte, numResponses),
			ReadResp: responses,
		}

		desc, err := handshaker.GetDescription(context.Background(), address.Address("localhost:27017"), conn)
		assert.Nil(t, err, "GetDescription error: %v", err)
		assert.Nil(t, desc.SpeculativeAuthenticate, "expected desc.SpeculativeAuthenticate to be unset, got %s",
			bson.Raw(desc.SpeculativeAuthenticate))
		conn.Desc = desc

		err = handshaker.FinishHandshake(context.Background(), conn)
		assert.Nil(t, err, "FinishHandshake error: %v", err)
		assert.Equal(t, 0, len(conn.ReadResp), "%d messages left unread", len(conn.ReadResp))

		assert.Equal(t, numResponses, len(conn.Written), "expected %d wire messages to be sent, got %d",
			numResponses, len(conn.Written))
		isMaster, err := drivertest.GetCommandFromQueryWireMessage(<-conn.Written)
		assert.Nil(t, err, "error parsing isMaster command: %v", err)
		assertCommandName(t, isMaster, "isMaster")
		_, err = isMaster.LookupErr("speculativeAuthenticate")
		assert.Nil(t, err, "expected command %s to contain 'speculativeAuthenticate'", bson.Raw(isMaster))

		authenticate, err := drivertest.GetCommandFromQueryWireMessage(<-conn.Written)
		assert.Nil(t, err, "error parsing authenticate command: %v", err)
		assertCommandName(t, authenticate, "authenticate")
	})
}

// createSpeculativeX509Handshake creates the server replies for a successful speculative X509 authentication attempt.
// There is only one reply:
//
// 1. isMaster reply containing a "speculativeAuthenticate" document.
func createSpeculativeX509Handshake() []bsoncore.Document {
	firstAuthElem := bsoncore.AppendDocumentElement(nil, "speculativeAuthenticate", x509Response)
	isMaster := bsoncore.BuildDocumentFromElements(nil, append(handshakeIsMasterElements, firstAuthElem)...)
	return []bsoncore.Document{isMaster}
}

// createSpeculativeX509Handshake creates the server replies for a handshake + X509 authentication attempt.
// There are two replies:
//
// 1. isMaster reply
// 2. authenticate reply
func createRegularX509Handshake() []bsoncore.Document {
	isMaster := bsoncore.BuildDocumentFromElements(nil, handshakeIsMasterElements...)
	return []bsoncore.Document{isMaster, x509Response}
}
