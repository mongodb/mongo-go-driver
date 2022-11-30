// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/drivertest"
)

const (
	scramSha1Nonce   = "fyko+d2lbbFgONRv9qkxdawL"
	scramSha256Nonce = "rOprNGfwEbeRWgbNEkqO"
)

var (
	scramSha1ShortPayloads = [][]byte{
		[]byte("r=fyko+d2lbbFgONRv9qkxdawLHo+Vgk7qvUOKUwuWLIWg4l/9SraGMHEE,s=rQ9ZY3MntBeuP3E1TDVC4w==,i=10000"),
		[]byte("v=UMWeI25JD1yNYZRMpZ4VHvhZ9e0="),
	}
	scramSha256ShortPayloads = [][]byte{
		[]byte("r=rOprNGfwEbeRWgbNEkqO%hvYDpWUa2RaTCAfuxFIlj)hNlF$k0,s=W22ZaJ0SNY7soEsUEjb6gQ==,i=4096"),
		[]byte("v=6rriTRBi23WpRR/wtup+mMhUZUn/dB5nLTJRsjl95G4="),
	}
	scramSha1LongPayloads   = append(scramSha1ShortPayloads, []byte{})
	scramSha256LongPayloads = append(scramSha256ShortPayloads, []byte{})
)

func TestSCRAM(t *testing.T) {
	t.Run("conversation", func(t *testing.T) {
		testCases := []struct {
			name                  string
			createAuthenticatorFn func(*Cred) (Authenticator, error)
			payloads              [][]byte
			nonce                 string
		}{
			{"scram-sha-1 short conversation", newScramSHA1Authenticator, scramSha1ShortPayloads, scramSha1Nonce},
			{"scram-sha-256 short conversation", newScramSHA256Authenticator, scramSha256ShortPayloads, scramSha256Nonce},
			{"scram-sha-1 long conversation", newScramSHA1Authenticator, scramSha1LongPayloads, scramSha1Nonce},
			{"scram-sha-256 long conversation", newScramSHA256Authenticator, scramSha256LongPayloads, scramSha256Nonce},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				authenticator, err := tc.createAuthenticatorFn(&Cred{
					Username: "user",
					Password: "pencil",
					Source:   "admin",
				})
				assert.Nil(t, err, "error creating authenticator: %v", err)
				sa, _ := authenticator.(*ScramAuthenticator)
				sa.client = sa.client.WithNonceGenerator(func() string {
					return tc.nonce
				})

				responses := make(chan []byte, len(tc.payloads))
				writeReplies(responses, createSCRAMConversation(tc.payloads)...)

				desc := description.Server{
					WireVersion: &description.VersionRange{
						Max: 4,
					},
				}
				conn := &drivertest.ChannelConn{
					Written:  make(chan []byte, len(tc.payloads)),
					ReadResp: responses,
					Desc:     desc,
				}

				err = authenticator.Auth(context.Background(), &Config{Description: desc, Connection: conn})
				assert.Nil(t, err, "Auth error: %v\n", err)

				// Verify that the first command sent is saslStart.
				assert.True(t, len(conn.Written) > 1, "wire messages were written to the connection")
				startCmd, err := drivertest.GetCommandFromQueryWireMessage(<-conn.Written)
				assert.Nil(t, err, "error parsing wire message: %v", err)
				cmdName := startCmd.Index(0).Key()
				assert.Equal(t, cmdName, "saslStart", "cmd name mismatch; expected 'saslStart', got %v", cmdName)

				// Verify that the saslStart command always has {options: {skipEmptyExchange: true}}
				optionsVal, err := startCmd.LookupErr("options")
				assert.Nil(t, err, "no options found in saslStart command")
				optionsDoc := optionsVal.Document()
				assert.Equal(t, optionsDoc, scramStartOptions, "expected options %v, got %v", scramStartOptions, optionsDoc)
			})
		}
	})
}

func createSCRAMConversation(payloads [][]byte) []bsoncore.Document {
	responses := make([]bsoncore.Document, len(payloads))
	for idx, payload := range payloads {
		res := createSCRAMServerResponse(payload, idx == len(payloads)-1)
		responses[idx] = res
	}
	return responses
}

func createSCRAMServerResponse(payload []byte, done bool) bsoncore.Document {
	return bsoncore.BuildDocumentFromElements(nil,
		bsoncore.AppendInt32Element(nil, "conversationId", 1),
		bsoncore.AppendBinaryElement(nil, "payload", 0x00, payload),
		bsoncore.AppendBooleanElement(nil, "done", done),
		bsoncore.AppendInt32Element(nil, "ok", 1),
	)
}

func writeReplies(c chan []byte, docs ...bsoncore.Document) {
	for _, doc := range docs {
		reply := drivertest.MakeReply(doc)
		c <- reply
	}
}
