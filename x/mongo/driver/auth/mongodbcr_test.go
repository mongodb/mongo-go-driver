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

	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/mnet"
)

func TestMongoDBCRAuthenticator_Fails(t *testing.T) {
	t.Parallel()

	authenticator := auth.MongoDBCRAuthenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
	}

	resps := make(chan []byte, 2)
	writeReplies(resps, bsoncore.BuildDocumentFromElements(nil,
		bsoncore.AppendInt32Element(nil, "ok", 1),
		bsoncore.AppendStringElement(nil, "nonce", "2375531c32080ae8"),
	), bsoncore.BuildDocumentFromElements(nil,
		bsoncore.AppendInt32Element(nil, "ok", 0),
	))

	desc := description.Server{
		WireVersion: &description.VersionRange{
			Max: 6,
		},
	}
	c := &drivertest.ChannelConn{
		Written:  make(chan []byte, 2),
		ReadResp: resps,
		Desc:     desc,
	}

	mnetconn := mnet.NewConnection(c)

	err := authenticator.Auth(context.Background(), &driver.AuthConfig{Connection: mnetconn})
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

	authenticator := auth.MongoDBCRAuthenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
	}

	resps := make(chan []byte, 2)
	writeReplies(resps, bsoncore.BuildDocumentFromElements(nil,
		bsoncore.AppendInt32Element(nil, "ok", 1),
		bsoncore.AppendStringElement(nil, "nonce", "2375531c32080ae8"),
	), bsoncore.BuildDocumentFromElements(nil,
		bsoncore.AppendInt32Element(nil, "ok", 1),
	))

	desc := description.Server{
		WireVersion: &description.VersionRange{
			Max: 6,
		},
	}
	c := &drivertest.ChannelConn{
		Written:  make(chan []byte, 2),
		ReadResp: resps,
		Desc:     desc,
	}

	mnetconn := mnet.NewConnection(c)

	err := authenticator.Auth(context.Background(), &driver.AuthConfig{Connection: mnetconn})
	if err != nil {
		t.Fatalf("expected no error but got \"%s\"", err)
	}

	if len(c.Written) != 2 {
		t.Fatalf("expected 2 messages to be sent but had %d", len(c.Written))
	}

	want := bsoncore.BuildDocumentFromElements(nil, bsoncore.AppendInt32Element(nil, "getnonce", 1))
	compareResponses(t, <-c.Written, want, "source")

	expectedAuthenticateDoc := bsoncore.BuildDocumentFromElements(nil,
		bsoncore.AppendInt32Element(nil, "authenticate", 1),
		bsoncore.AppendStringElement(nil, "user", "user"),
		bsoncore.AppendStringElement(nil, "nonce", "2375531c32080ae8"),
		bsoncore.AppendStringElement(nil, "key", "21742f26431831d5cfca035a08c5bdf6"),
	)
	compareResponses(t, <-c.Written, expectedAuthenticateDoc, "source")
}

func writeReplies(c chan []byte, docs ...bsoncore.Document) {
	for _, doc := range docs {
		reply := drivertest.MakeReply(doc)
		c <- reply
	}
}
