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
	"github.com/mongodb/mongo-go-driver/mongo/internal/conntest"
	"github.com/mongodb/mongo-go-driver/mongo/internal/msgtest"
	. "github.com/mongodb/mongo-go-driver/mongo/private/auth"
	"github.com/mongodb/mongo-go-driver/mongo/private/msg"
)

func TestMongoDBCRAuthenticator_Fails(t *testing.T) {
	t.Parallel()

	authenticator := MongoDBCRAuthenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
	}

	getNonceReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.String("nonce", "2375531c32080ae8")))

	authenticateReply := msgtest.CreateCommandReply(bson.NewDocument(bson.EC.Int32("ok", 0)))

	conn := &conntest.MockConnection{
		ResponseQ: []*msg.Reply{getNonceReply, authenticateReply},
	}

	err := authenticator.Auth(context.Background(), conn)
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

	getNonceReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.String("nonce", "2375531c32080ae8")))

	authenticateReply := msgtest.CreateCommandReply(bson.NewDocument(bson.EC.Int32("ok", 1)))

	conn := &conntest.MockConnection{
		ResponseQ: []*msg.Reply{getNonceReply, authenticateReply},
	}

	err := authenticator.Auth(context.Background(), conn)
	if err != nil {
		t.Fatalf("expected no error but got \"%s\"", err)
	}

	if len(conn.Sent) != 2 {
		t.Fatalf("expected 2 messages to be sent but had %d", len(conn.Sent))
	}

	getNonceRequest := conn.Sent[0].(*msg.Query)
	if !reflect.DeepEqual(getNonceRequest.Query, bson.NewDocument(bson.EC.Int32("getnonce", 1))) {
		t.Fatalf("getnonce command was incorrect: %v", getNonceRequest.Query)
	}

	authenticateRequest := conn.Sent[1].(*msg.Query)
	expectedAuthenticateDoc := bson.NewDocument(
		bson.EC.Int32("authenticate", 1),
		bson.EC.String("user", "user"),
		bson.EC.String("nonce", "2375531c32080ae8"),
		bson.EC.String("key", "21742f26431831d5cfca035a08c5bdf6"))

	if !reflect.DeepEqual(authenticateRequest.Query, expectedAuthenticateDoc) {
		t.Fatalf("authenticate command was incorrect: %v", authenticateRequest.Query)
	}
}
