// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth_test

import (
	"context"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	. "github.com/mongodb/mongo-go-driver/core/auth"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
	"github.com/stretchr/testify/require"
)

func TestCreateAuthenticator(t *testing.T) {

	tests := []struct {
		name   string
		source string
		auther Authenticator
	}{
		{name: "", auther: &DefaultAuthenticator{}},
		{name: "SCRAM-SHA-1", auther: &ScramSHA1Authenticator{}},
		{name: "MONGODB-CR", auther: &MongoDBCRAuthenticator{}},
		{name: "PLAIN", auther: &PlainAuthenticator{}},
		{name: "MONGODB-X509", auther: &MongoDBX509Authenticator{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cred := &Cred{
				Username:    "user",
				Password:    "pencil",
				PasswordSet: true,
			}

			a, err := CreateAuthenticator(test.name, cred)
			require.NoError(t, err)
			require.IsType(t, test.auther, a)
		})
	}
}

type conn struct {
	t        *testing.T
	writeErr error
	written  chan wiremessage.WireMessage
	readResp chan wiremessage.WireMessage
	readErr  chan error
}

func (c *conn) WriteWireMessage(ctx context.Context, wm wiremessage.WireMessage) error {
	select {
	case c.written <- wm:
	default:
		c.t.Error("could not write wiremessage to written channel")
	}
	return c.writeErr
}

func (c *conn) ReadWireMessage(ctx context.Context) (wiremessage.WireMessage, error) {
	var wm wiremessage.WireMessage
	var err error
	select {
	case wm = <-c.readResp:
	case err = <-c.readErr:
	case <-ctx.Done():
	}
	return wm, err
}

func (c *conn) Close() error {
	return nil
}

func (c *conn) Expired() bool {
	return false
}

func (c *conn) Alive() bool {
	return true
}

func (c *conn) ID() string {
	return "faked"
}

func makeReply(t *testing.T, doc *bson.Document) wiremessage.WireMessage {
	rdr, err := doc.MarshalBSON()
	if err != nil {
		t.Fatalf("Could not create document: %v", err)
	}
	return wiremessage.Reply{
		NumberReturned: 1,
		Documents:      []bson.Reader{rdr},
	}
}
