// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth_test

import (
	"context"
	"testing"

	"strings"

	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/x/bsonx"
	. "go.mongodb.org/mongo-driver/x/mongo/driverlegacy/auth"
	"go.mongodb.org/mongo-driver/x/network/description"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
)

func TestMongoDBCRAuthenticator_Fails(t *testing.T) {
	t.Parallel()

	authenticator := MongoDBCRAuthenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
	}

	resps := make(chan wiremessage.WireMessage, 2)
	writeReplies(t, resps, bsonx.Doc{
		{"ok", bsonx.Int32(1)},
		{"nonce", bsonx.String("2375531c32080ae8")},
	}, bsonx.Doc{
		{"ok", bsonx.Int32(0)},
	})

	c := &internal.ChannelConn{Written: make(chan wiremessage.WireMessage, 2), ReadResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{
		WireVersion: &description.VersionRange{
			Max: 6,
		},
	}, c)
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
	writeReplies(t, resps, bsonx.Doc{
		{"ok", bsonx.Int32(1)},
		{"nonce", bsonx.String("2375531c32080ae8")},
	}, bsonx.Doc{
		{"ok", bsonx.Int32(1)},
	})

	c := &internal.ChannelConn{Written: make(chan wiremessage.WireMessage, 2), ReadResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{
		WireVersion: &description.VersionRange{
			Max: 6,
		},
	}, c)
	if err != nil {
		t.Fatalf("expected no error but got \"%s\"", err)
	}

	if len(c.Written) != 2 {
		t.Fatalf("expected 2 messages to be sent but had %d", len(c.Written))
	}

	want := bsonx.Doc{{"getnonce", bsonx.Int32(1)}}
	compareResponses(t, <-c.Written, want, "source")

	expectedAuthenticateDoc := bsonx.Doc{
		{"authenticate", bsonx.Int32(1)},
		{"user", bsonx.String("user")},
		{"nonce", bsonx.String("2375531c32080ae8")},
		{"key", bsonx.String("21742f26431831d5cfca035a08c5bdf6")},
	}
	compareResponses(t, <-c.Written, expectedAuthenticateDoc, "source")
}

func writeReplies(t *testing.T, c chan wiremessage.WireMessage, docs ...bsonx.Doc) {
	for _, doc := range docs {
		reply, err := internal.MakeReply(doc)
		if err != nil {
			t.Fatalf("error constructing reply: %v", err)
		}

		c <- reply
	}
}
