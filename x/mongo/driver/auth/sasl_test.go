// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth_test

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/mnet"
)

// closeCountingSaslClient is a single-step SaslClientCloser that records how
// many times Close is called.
type closeCountingSaslClient struct {
	closed int
}

var _ auth.SaslClientCloser = (*closeCountingSaslClient)(nil)

func (c *closeCountingSaslClient) Start() (string, []byte, error) {
	return "MOCK", []byte("client-first"), nil
}

func (c *closeCountingSaslClient) Next(context.Context, []byte) ([]byte, error) {
	return nil, nil
}

func (c *closeCountingSaslClient) Completed() bool {
	return true
}

func (c *closeCountingSaslClient) Close() {
	c.closed++
}

func TestConductSaslConversation_ClosesClient(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		reply   bsoncore.Document
		wantErr bool
	}{
		{
			name: "saslStart succeeds",
			reply: bsoncore.BuildDocumentFromElements(nil,
				bsoncore.AppendInt32Element(nil, "ok", 1),
				bsoncore.AppendInt32Element(nil, "conversationId", 1),
				bsoncore.AppendBinaryElement(nil, "payload", 0x00, []byte{}),
				bsoncore.AppendBooleanElement(nil, "done", true),
			),
		},
		{
			name: "saslStart fails",
			reply: bsoncore.BuildDocumentFromElements(nil,
				bsoncore.AppendInt32Element(nil, "ok", 0),
				bsoncore.AppendInt32Element(nil, "code", 18),
				bsoncore.AppendStringElement(nil, "errmsg", "Authentication failed."),
			),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resps := make(chan []byte, 1)
			writeReplies(resps, tc.reply)

			desc := description.Server{
				WireVersion: &description.VersionRange{
					Max: 6,
				},
			}
			c := &drivertest.ChannelConn{
				Written:  make(chan []byte, 1),
				ReadResp: resps,
				Desc:     desc,
			}

			client := &closeCountingSaslClient{}
			cfg := &driver.AuthConfig{Connection: mnet.NewConnection(c)}
			err := auth.ConductSaslConversation(context.Background(), cfg, "$external", client)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, 1, client.closed, "expected Close to be called exactly once")
		})
	}
}
