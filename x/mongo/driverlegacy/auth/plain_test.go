// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth_test

import (
	"context"
	"strings"
	"testing"

	"encoding/base64"

	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/x/bsonx"
	. "go.mongodb.org/mongo-driver/x/mongo/driverlegacy/auth"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
)

func TestPlainAuthenticator_Fails(t *testing.T) {
	t.Parallel()

	authenticator := PlainAuthenticator{
		Username: "user",
		Password: "pencil",
	}

	resps := make(chan wiremessage.WireMessage, 1)
	writeReplies(t, resps, bsonx.Doc{
		{"ok", bsonx.Int32(1)},
		{"conversationId", bsonx.Int32(1)},
		{"payload", bsonx.Binary(0x00, []byte{})},
		{"code", bsonx.Int32(143)},
		{"done", bsonx.Boolean(true)},
	})

	c := &internal.ChannelConn{Written: make(chan wiremessage.WireMessage, 1), ReadResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{
		WireVersion: &description.VersionRange{
			Max: 6,
		},
	}, c)
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

	authenticator := PlainAuthenticator{
		Username: "user",
		Password: "pencil",
	}

	resps := make(chan wiremessage.WireMessage, 2)
	writeReplies(t, resps, bsonx.Doc{
		{"ok", bsonx.Int32(1)},
		{"conversationId", bsonx.Int32(1)},
		{"payload", bsonx.Binary(0x00, []byte{})},
		{"done", bsonx.Boolean(false)},
	}, bsonx.Doc{
		{"ok", bsonx.Int32(1)},
		{"conversationId", bsonx.Int32(1)},
		{"payload", bsonx.Binary(0x00, []byte{})},
		{"done", bsonx.Boolean(true)},
	})

	c := &internal.ChannelConn{Written: make(chan wiremessage.WireMessage, 1), ReadResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{
		WireVersion: &description.VersionRange{
			Max: 6,
		},
	}, c)
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

	authenticator := PlainAuthenticator{
		Username: "user",
		Password: "pencil",
	}

	resps := make(chan wiremessage.WireMessage, 1)
	writeReplies(t, resps, bsonx.Doc{
		{"ok", bsonx.Int32(1)},
		{"conversationId", bsonx.Int32(1)},
		{"payload", bsonx.Binary(0x00, []byte{})},
		{"done", bsonx.Boolean(true)},
	})

	c := &internal.ChannelConn{Written: make(chan wiremessage.WireMessage, 1), ReadResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{
		WireVersion: &description.VersionRange{
			Max: 6,
		},
	}, c)
	if err != nil {
		t.Fatalf("expected no error but got \"%s\"", err)
	}

	if len(c.Written) != 1 {
		t.Fatalf("expected 1 messages to be sent but had %d", len(c.Written))
	}

	payload, _ := base64.StdEncoding.DecodeString("AHVzZXIAcGVuY2ls")
	expectedCmd := bsonx.Doc{
		{"saslStart", bsonx.Int32(1)},
		{"mechanism", bsonx.String("PLAIN")},
		{"payload", bsonx.Binary(0x00, payload)},
	}
	compareResponses(t, <-c.Written, expectedCmd, "$external")
}
