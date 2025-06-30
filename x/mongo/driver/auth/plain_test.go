// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth_test

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/mnet"
)

func TestPlainAuthenticator_Fails(t *testing.T) {
	t.Parallel()

	authenticator := auth.PlainAuthenticator{
		Username: "user",
		Password: "pencil",
	}

	resps := make(chan []byte, 1)
	writeReplies(resps, bsoncore.BuildDocumentFromElements(nil,
		bsoncore.AppendInt32Element(nil, "ok", 1),
		bsoncore.AppendInt32Element(nil, "conversationId", 1),
		bsoncore.AppendBinaryElement(nil, "payload", 0x00, []byte{}),
		bsoncore.AppendInt32Element(nil, "code", 143),
		bsoncore.AppendBooleanElement(nil, "done", true),
	))

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

	mnetconn := mnet.NewConnection(c)

	err := authenticator.Auth(context.Background(), &driver.AuthConfig{Connection: mnetconn})
	if err == nil {
		t.Fatalf("expected an error but got none")
	}

	errPrefix := "unable to authenticate using mechanism \"PLAIN\""
	if !strings.HasPrefix(err.Error(), errPrefix) {
		t.Fatalf("expected an err starting with \"%s\" but got \"%s\"", errPrefix, err)
	}
}

func TestPlainAuthenticator_Extra_server_message(t *testing.T) {
	t.Parallel()

	authenticator := auth.PlainAuthenticator{
		Username: "user",
		Password: "pencil",
	}

	resps := make(chan []byte, 2)
	writeReplies(resps, bsoncore.BuildDocumentFromElements(nil,
		bsoncore.AppendInt32Element(nil, "ok", 1),
		bsoncore.AppendInt32Element(nil, "conversationId", 1),
		bsoncore.AppendBinaryElement(nil, "payload", 0x00, []byte{}),
		bsoncore.AppendBooleanElement(nil, "done", false),
	), bsoncore.BuildDocumentFromElements(nil,
		bsoncore.AppendInt32Element(nil, "ok", 1),
		bsoncore.AppendInt32Element(nil, "conversationId", 1),
		bsoncore.AppendBinaryElement(nil, "payload", 0x00, []byte{}),
		bsoncore.AppendBooleanElement(nil, "done", true),
	))

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

	mnetconn := mnet.NewConnection(c)

	err := authenticator.Auth(context.Background(), &driver.AuthConfig{Connection: mnetconn})
	if err == nil {
		t.Fatalf("expected an error but got none")
	}

	errPrefix := "unable to authenticate using mechanism \"PLAIN\": unexpected server challenge"
	if !strings.HasPrefix(err.Error(), errPrefix) {
		t.Fatalf("expected an err starting with \"%s\" but got \"%s\"", errPrefix, err)
	}
}

func TestPlainAuthenticator_Succeeds(t *testing.T) {
	t.Parallel()

	authenticator := auth.PlainAuthenticator{
		Username: "user",
		Password: "pencil",
	}

	resps := make(chan []byte, 1)
	writeReplies(resps, bsoncore.BuildDocumentFromElements(nil,
		bsoncore.AppendInt32Element(nil, "ok", 1),
		bsoncore.AppendInt32Element(nil, "conversationId", 1),
		bsoncore.AppendBinaryElement(nil, "payload", 0x00, []byte{}),
		bsoncore.AppendBooleanElement(nil, "done", true),
	))

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

	mnetconn := mnet.NewConnection(c)

	err := authenticator.Auth(context.Background(), &driver.AuthConfig{Connection: mnetconn})
	if err != nil {
		t.Fatalf("expected no error but got \"%s\"", err)
	}

	if len(c.Written) != 1 {
		t.Fatalf("expected 1 messages to be sent but had %d", len(c.Written))
	}

	payload, _ := base64.StdEncoding.DecodeString("AHVzZXIAcGVuY2ls")
	expectedCmd := bsoncore.BuildDocumentFromElements(nil,
		bsoncore.AppendInt32Element(nil, "saslStart", 1),
		bsoncore.AppendStringElement(nil, "mechanism", "PLAIN"),
		bsoncore.AppendBinaryElement(nil, "payload", 0x00, payload),
	)
	compareResponses(t, <-c.Written, expectedCmd, "$external")
}

func TestPlainAuthenticator_SucceedsBoolean(t *testing.T) {
	t.Parallel()

	authenticator := auth.PlainAuthenticator{
		Username: "user",
		Password: "pencil",
	}

	resps := make(chan []byte, 1)
	writeReplies(resps, bsoncore.BuildDocumentFromElements(nil,
		bsoncore.AppendBooleanElement(nil, "ok", true),
		bsoncore.AppendInt32Element(nil, "conversationId", 1),
		bsoncore.AppendBinaryElement(nil, "payload", 0x00, []byte{}),
		bsoncore.AppendBooleanElement(nil, "done", true),
	))

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

	mnetconn := mnet.NewConnection(c)

	err := authenticator.Auth(context.Background(), &driver.AuthConfig{Connection: mnetconn})
	require.NoError(t, err, "Auth error")
	require.Len(t, c.Written, 1, "expected 1 messages to be sent")

	payload, err := base64.StdEncoding.DecodeString("AHVzZXIAcGVuY2ls")
	require.NoError(t, err, "DecodeString error")

	expectedCmd := bsoncore.BuildDocumentFromElements(nil,
		bsoncore.AppendInt32Element(nil, "saslStart", 1),
		bsoncore.AppendStringElement(nil, "mechanism", "PLAIN"),
		bsoncore.AppendBinaryElement(nil, "payload", 0x00, payload),
	)
	compareResponses(t, <-c.Written, expectedCmd, "$external")
}

func writeReplies(c chan []byte, docs ...bsoncore.Document) {
	for _, doc := range docs {
		reply := drivertest.MakeReply(doc)
		c <- reply
	}
}
