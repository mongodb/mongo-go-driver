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

	"reflect"

	"encoding/base64"

	"github.com/mongodb/mongo-go-driver/bson"
	. "github.com/mongodb/mongo-go-driver/core/auth"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

func TestPlainAuthenticator_Fails(t *testing.T) {
	t.Parallel()

	authenticator := PlainAuthenticator{
		Username: "user",
		Password: "pencil",
	}

	resps := make(chan wiremessage.WireMessage, 1)
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", []byte{}),
		bson.EC.Int32("code", 143),
		bson.EC.Boolean("done", true)),
	)

	c := &conn{written: make(chan wiremessage.WireMessage, 1), readResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{}, c)
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
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", []byte{}),
		bson.EC.Boolean("done", false)),
	)
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", []byte{}),
		bson.EC.Boolean("done", true)),
	)

	c := &conn{written: make(chan wiremessage.WireMessage, 1), readResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{}, c)
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
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", []byte{}),
		bson.EC.Boolean("done", true)),
	)

	c := &conn{written: make(chan wiremessage.WireMessage, 1), readResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{}, c)
	if err != nil {
		t.Fatalf("expected no error but got \"%s\"", err)
	}

	if len(c.written) != 1 {
		t.Fatalf("expected 1 messages to be sent but had %d", len(c.written))
	}

	saslStartRequest := (<-c.written).(wiremessage.Query)
	payload, _ := base64.StdEncoding.DecodeString("AHVzZXIAcGVuY2ls")
	expectedCmd, err := bson.NewDocument(
		bson.EC.Int32("saslStart", 1),
		bson.EC.String("mechanism", "PLAIN"),
		bson.EC.Binary("payload", payload),
	).MarshalBSON()
	if err != nil {
		t.Fatalf("couldn't marshal bson: %v", err)
	}

	if !reflect.DeepEqual(saslStartRequest.Query, bson.Reader(expectedCmd)) {
		t.Fatalf("saslStart command was incorrect. got %v; want %v", saslStartRequest.Query, bson.Reader(expectedCmd))
	}
}
