// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth_test

import (
	"testing"

	"reflect"

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

func compareResponses(t *testing.T, wm wiremessage.WireMessage, expectedPayload *bson.Document, dbName string) {
	switch converted := wm.(type) {
	case wiremessage.Query:
		payloadBytes, err := expectedPayload.MarshalBSON()
		if err != nil {
			t.Fatalf("couldn't marshal query bson: %v", err)
		}
		require.True(t, reflect.DeepEqual([]byte(converted.Query), payloadBytes))
	case wiremessage.Msg:
		msgPayload := expectedPayload.Append(bson.EC.String("$db", dbName))
		payloadBytes, err := msgPayload.MarshalBSON()
		if err != nil {
			t.Fatalf("couldn't marshal msg bson: %v", err)
		}

		require.True(t, reflect.DeepEqual([]byte(converted.Sections[0].(wiremessage.SectionBody).Document), payloadBytes))
	}
}
