// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth_test

import (
	"context"
	"testing"

	"reflect"

	"strings"

	"github.com/mongodb/mongo-go-driver/bson"
	. "github.com/mongodb/mongo-go-driver/core/auth"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
)

func TestMongoDBCRAuthenticator_Fails(t *testing.T) {
	t.Parallel()

	authenticator := MongoDBCRAuthenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
	}

	resps := make(chan wiremessage.WireMessage, 2)
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.String("nonce", "2375531c32080ae8"),
	))

	resps <- makeReply(t, bson.NewDocument(bson.EC.Int32("ok", 0)))

	c := &conn{written: make(chan wiremessage.WireMessage, 2), readResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{}, c)
	if err == nil {
		t.Fatalf("expected an error but got none")
	}

	errPrefix := "unable to authenticate using mechanism \"MONGODB-CR\""
	if !strings.HasPrefix(err.Error(), errPrefix) {
		t.Fatalf("expected an err starting with \"%s\" but got \"%s\"", errPrefix, err)
	}
}

func TestMongoDBCRAuthenticator_Succeeds(t *testing.T) {
	t.Parallel()

	authenticator := MongoDBCRAuthenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
	}

	resps := make(chan wiremessage.WireMessage, 2)

	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.String("nonce", "2375531c32080ae8"),
	))

	resps <- makeReply(t, bson.NewDocument(bson.EC.Int32("ok", 1)))

	c := &conn{written: make(chan wiremessage.WireMessage, 2), readResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{}, c)
	if err != nil {
		t.Fatalf("expected no error but got \"%s\"", err)
	}

	if len(c.written) != 2 {
		t.Fatalf("expected 2 messages to be sent but had %d", len(c.written))
	}

	getNonceRequest := (<-c.written).(wiremessage.Query)
	var want bson.Reader
	want, err = bson.NewDocument(bson.EC.Int32("getnonce", 1)).MarshalBSON()
	if err != nil {
		t.Fatalf("couldn't marshal bson: %v", err)
	}
	if !reflect.DeepEqual(getNonceRequest.Query, want) {
		t.Fatalf("getnonce command was incorrect: %v", getNonceRequest.Query)
	}

	authenticateRequest := (<-c.written).(wiremessage.Query)
	var expectedAuthenticateDoc bson.Reader
	expectedAuthenticateDoc, err = bson.NewDocument(
		bson.EC.Int32("authenticate", 1),
		bson.EC.String("user", "user"),
		bson.EC.String("nonce", "2375531c32080ae8"),
		bson.EC.String("key", "21742f26431831d5cfca035a08c5bdf6"),
	).MarshalBSON()
	if err != nil {
		t.Fatalf("couldn't marshal bson: %v", err)
	}

	if !reflect.DeepEqual(authenticateRequest.Query, expectedAuthenticateDoc) {
		t.Fatalf("authenticate command was incorrect: %v", authenticateRequest.Query)
	}
}
