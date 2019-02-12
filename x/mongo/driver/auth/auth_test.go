// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth_test

import (
	"testing"

	"reflect"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/x/bsonx"
	. "go.mongodb.org/mongo-driver/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
)

func TestCreateAuthenticator(t *testing.T) {

	tests := []struct {
		name   string
		source string
		auther Authenticator
	}{
		{name: "", auther: &DefaultAuthenticator{}},
		{name: "SCRAM-SHA-1", auther: &ScramAuthenticator{}},
		{name: "SCRAM-SHA-256", auther: &ScramAuthenticator{}},
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

func compareResponses(t *testing.T, wm wiremessage.WireMessage, expectedPayload bsonx.Doc, dbName string) {
	switch converted := wm.(type) {
	case wiremessage.Query:
		payloadBytes, err := expectedPayload.MarshalBSON()
		if err != nil {
			t.Fatalf("couldn't marshal query bson: %v", err)
		}
		require.True(t, reflect.DeepEqual([]byte(converted.Query), payloadBytes))
	case wiremessage.Msg:
		msgPayload := append(expectedPayload, bsonx.Elem{"$db", bsonx.String(dbName)})
		payloadBytes, err := msgPayload.MarshalBSON()
		if err != nil {
			t.Fatalf("couldn't marshal msg bson: %v", err)
		}

		require.True(t, reflect.DeepEqual([]byte(converted.Sections[0].(wiremessage.SectionBody).Document), payloadBytes))
	}
}
