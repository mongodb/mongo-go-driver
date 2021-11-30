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
	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/drivertest"
)

var (
	// The base elements for a hello response.
	handshakeHelloElements = [][]byte{
		bsoncore.AppendInt32Element(nil, "ok", 1),
		bsoncore.AppendBooleanElement(nil, internal.LegacyHelloLowercase, true),
		bsoncore.AppendInt32Element(nil, "maxBsonObjectSize", 16777216),
		bsoncore.AppendInt32Element(nil, "maxMessageSizeBytes", 48000000),
		bsoncore.AppendInt32Element(nil, "minWireVersion", 0),
		bsoncore.AppendInt32Element(nil, "maxWireVersion", 4),
	}
	// The first payload sent by the driver for SCRAM-SHA-1/256 authentication.
	firstScramSha1ClientPayload   = []byte("n,,n=user,r=fyko+d2lbbFgONRv9qkxdawL")
	firstScramSha256ClientPayload = []byte("n,,n=user,r=rOprNGfwEbeRWgbNEkqO")
)

func TestSpeculativeSCRAM(t *testing.T) {
	cred := &Cred{
		Username:    "user",
		Password:    "pencil",
		PasswordSet: true,
		Source:      "admin",
	}

	t.Run("speculative response included", func(t *testing.T) {
		// Tests for SCRAM-SHA1 and SCRAM-SHA-256 when the hello response contains a reply to the speculative
		// authentication attempt. The driver should only send a saslContinue after the hello to complete
		// authentication.

		testCases := []struct {
			name               string
			mechanism          string
			firstClientPayload []byte
			payloads           [][]byte
			nonce              string
		}{
			{"SCRAM-SHA-1", "SCRAM-SHA-1", firstScramSha1ClientPayload, scramSha1ShortPayloads, scramSha1Nonce},
			{"SCRAM-SHA-256", "SCRAM-SHA-256", firstScramSha256ClientPayload, scramSha256ShortPayloads, scramSha256Nonce},
			{"Default", "", firstScramSha256ClientPayload, scramSha256ShortPayloads, scramSha256Nonce},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Create a SCRAM authenticator and overwrite the nonce generator to make the conversation
				// deterministic.
				authenticator, err := CreateAuthenticator(tc.mechanism, cred)
				assert.Nil(t, err, "CreateAuthenticator error: %v", err)
				setNonce(t, authenticator, tc.nonce)

				// Create a Handshaker and fake connection to authenticate.
				handshaker := Handshaker(nil, &HandshakeOptions{
					Authenticator: authenticator,
					DBUser:        "admin.user",
				})
				responses := make(chan []byte, len(tc.payloads))
				writeReplies(responses, createSpeculativeSCRAMHandshake(tc.payloads)...)

				conn := &drivertest.ChannelConn{
					Written:  make(chan []byte, len(tc.payloads)),
					ReadResp: responses,
				}

				// Do both parts of the handshake.
				info, err := handshaker.GetHandshakeInformation(context.Background(), address.Address("localhost:27017"), conn)
				assert.Nil(t, err, "GetHandshakeInformation error: %v", err)
				assert.NotNil(t, info.SpeculativeAuthenticate, "desc.SpeculativeAuthenticate not set")
				conn.Desc = info.Description // Set conn.Desc so the new description will be used for the authentication.

				err = handshaker.FinishHandshake(context.Background(), conn)
				assert.Nil(t, err, "FinishHandshake error: %v", err)
				assert.Equal(t, 0, len(conn.ReadResp), "%d messages left unread", len(conn.ReadResp))

				// Assert that the driver sent hello with the speculative authentication message.
				assert.Equal(t, len(tc.payloads), len(conn.Written), "expected %d wire messages to be sent, got %d",
					len(tc.payloads), (conn.Written))
				helloCmd, err := drivertest.GetCommandFromQueryWireMessage(<-conn.Written)
				assert.Nil(t, err, "error parsing hello command: %v", err)
				assertCommandName(t, helloCmd, internal.LegacyHello)

				// Assert that the correct document was sent for speculative authentication.
				authDocVal, err := helloCmd.LookupErr("speculativeAuthenticate")
				assert.Nil(t, err, "expected command %s to contain 'speculativeAuthenticate'", bson.Raw(helloCmd))
				authDoc := authDocVal.Document()
				sentMechanism := tc.mechanism
				if sentMechanism == "" {
					sentMechanism = "SCRAM-SHA-256"
				}

				expectedAuthDoc := bsoncore.BuildDocumentFromElements(nil,
					bsoncore.AppendInt32Element(nil, "saslStart", 1),
					bsoncore.AppendStringElement(nil, "mechanism", sentMechanism),
					bsoncore.AppendBinaryElement(nil, "payload", 0x00, tc.firstClientPayload),
					bsoncore.AppendStringElement(nil, "db", "admin"),
					bsoncore.AppendDocumentElement(nil, "options", bsoncore.BuildDocumentFromElements(nil,
						bsoncore.AppendBooleanElement(nil, "skipEmptyExchange", true),
					)),
				)
				assert.True(t, bytes.Equal(expectedAuthDoc, authDoc), "expected speculative auth document %s, got %s",
					bson.Raw(expectedAuthDoc), authDoc)

				// Assert that the last command sent in the handshake is saslContinue.
				saslContinueCmd, err := drivertest.GetCommandFromQueryWireMessage(<-conn.Written)
				assert.Nil(t, err, "error parsing saslContinue command: %v", err)
				assertCommandName(t, saslContinueCmd, "saslContinue")
			})
		}
	})
	t.Run("speculative response not included", func(t *testing.T) {
		// Tests for SCRAM-SHA-1 and SCRAM-SHA-256 when the hello response does not contain a reply to the
		// speculative authentication attempt. The driver should send both saslStart and saslContinue after the initial
		// hello.

		// There is no test for the default mechanism because we can't control the nonce used for the actual
		// authentication attempt after the speculative attempt fails.

		testCases := []struct {
			mechanism string
			payloads  [][]byte
			nonce     string
		}{
			{"SCRAM-SHA-1", scramSha1ShortPayloads, scramSha1Nonce},
			{"SCRAM-SHA-256", scramSha256ShortPayloads, scramSha256Nonce},
		}

		for _, tc := range testCases {
			t.Run(tc.mechanism, func(t *testing.T) {
				authenticator, err := CreateAuthenticator(tc.mechanism, cred)
				assert.Nil(t, err, "CreateAuthenticator error: %v", err)
				setNonce(t, authenticator, tc.nonce)

				handshaker := Handshaker(nil, &HandshakeOptions{
					Authenticator: authenticator,
					DBUser:        "admin.user",
				})
				numResponses := len(tc.payloads) + 1 // +1 for hello response
				responses := make(chan []byte, numResponses)
				writeReplies(responses, createRegularSCRAMHandshake(tc.payloads)...)

				conn := &drivertest.ChannelConn{
					Written:  make(chan []byte, numResponses),
					ReadResp: responses,
				}

				info, err := handshaker.GetHandshakeInformation(context.Background(), address.Address("localhost:27017"), conn)
				assert.Nil(t, err, "GetHandshakeInformation error: %v", err)
				assert.Nil(t, info.SpeculativeAuthenticate, "expected desc.SpeculativeAuthenticate to be unset, got %s",
					bson.Raw(info.SpeculativeAuthenticate))
				conn.Desc = info.Description

				err = handshaker.FinishHandshake(context.Background(), conn)
				assert.Nil(t, err, "FinishHandshake error: %v", err)
				assert.Equal(t, 0, len(conn.ReadResp), "%d messages left unread", len(conn.ReadResp))

				assert.Equal(t, numResponses, len(conn.Written), "expected %d wire messages to be sent, got %d",
					numResponses, len(conn.Written))
				hello, err := drivertest.GetCommandFromQueryWireMessage(<-conn.Written)
				assert.Nil(t, err, "error parsing hello command: %v", err)
				assertCommandName(t, hello, internal.LegacyHello)
				_, err = hello.LookupErr("speculativeAuthenticate")
				assert.Nil(t, err, "expected command %s to contain 'speculativeAuthenticate'", bson.Raw(hello))

				saslStart, err := drivertest.GetCommandFromQueryWireMessage(<-conn.Written)
				assert.Nil(t, err, "error parsing saslStart command: %v", err)
				assertCommandName(t, saslStart, "saslStart")

				saslContinue, err := drivertest.GetCommandFromQueryWireMessage(<-conn.Written)
				assert.Nil(t, err, "error parsing saslContinue command: %v", err)
				assertCommandName(t, saslContinue, "saslContinue")
			})
		}
	})
}

func setNonce(t *testing.T, authenticator Authenticator, nonce string) {
	t.Helper()
	nonceGenerator := func() string {
		return nonce
	}

	switch converted := authenticator.(type) {
	case *ScramAuthenticator:
		converted.client = converted.client.WithNonceGenerator(nonceGenerator)
	case *DefaultAuthenticator:
		sa := converted.speculativeAuthenticator.(*ScramAuthenticator)
		sa.client = sa.client.WithNonceGenerator(nonceGenerator)
	default:
		t.Fatalf("invalid authenticator type %T", authenticator)
	}
}

// createSpeculativeSCRAMHandshake creates the server replies for a successful speculative SCRAM authentication attempt.
// There are two replies:
//
// 1. hello reply containing a "speculativeAuthenticate" document.
// 2. saslContinue reply with done:true
func createSpeculativeSCRAMHandshake(payloads [][]byte) []bsoncore.Document {
	firstAuthResponse := createSCRAMServerResponse(payloads[0], false)
	firstAuthElem := bsoncore.AppendDocumentElement(nil, "speculativeAuthenticate", firstAuthResponse)
	hello := bsoncore.BuildDocumentFromElements(nil, append(handshakeHelloElements, firstAuthElem)...)

	responses := []bsoncore.Document{hello}
	for idx := 1; idx < len(payloads); idx++ {
		responses = append(responses, createSCRAMServerResponse(payloads[idx], idx == len(payloads)-1))
	}
	return responses
}

// createRegularSCRAMHandshake creates the server replies for a handshake + SCRAM authentication attempt. There are
// three replies:
//
// 1. hello reply
// 2. saslStart reply with done:false
// 3. saslContinue reply with done:true
func createRegularSCRAMHandshake(payloads [][]byte) []bsoncore.Document {
	hello := bsoncore.BuildDocumentFromElements(nil, handshakeHelloElements...)
	responses := []bsoncore.Document{hello}

	for idx, payload := range payloads {
		responses = append(responses, createSCRAMServerResponse(payload, idx == len(payloads)-1))
	}
	return responses
}

func assertCommandName(t *testing.T, cmd bsoncore.Document, expectedName string) {
	t.Helper()

	actualName := cmd.Index(0).Key()
	assert.Equal(t, expectedName, actualName, "expected command name '%s', got '%s'", expectedName, actualName)
}
