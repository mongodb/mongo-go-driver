// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	. "go.mongodb.org/mongo-driver/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/x/mongo/driver/wiremessage"
)

func TestCreateAuthenticator(t *testing.T) {

	tests := []struct {
		name   string
		source string
		auth   Authenticator
	}{
		{name: "", auth: &DefaultAuthenticator{}},
		{name: "SCRAM-SHA-1", auth: &ScramAuthenticator{}},
		{name: "SCRAM-SHA-256", auth: &ScramAuthenticator{}},
		{name: "MONGODB-CR", auth: &MongoDBCRAuthenticator{}},
		{name: "PLAIN", auth: &PlainAuthenticator{}},
		{name: "MONGODB-X509", auth: &MongoDBX509Authenticator{}},
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
			require.IsType(t, test.auth, a)
		})
	}
}

func compareResponses(t *testing.T, wm []byte, expectedPayload bsoncore.Document, dbName string) {
	hdr, wm, ok := bsoncore.ReadBytes(wm, 16)
	if !ok {
		t.Fatalf("wiremessage is too short to unmarshal header")
	}
	_, _, _, opcode, ok := wiremessage.ParseHeader(hdr)
	if !ok {
		t.Fatalf("wiremessage is too short to unmarshal header")
	}
	var actualPayload bsoncore.Document
	switch opcode {
	case wiremessage.OpQuery:
		_, wm, ok := bsoncore.ReadInt32(wm)
		if !ok {
			t.Fatalf("wiremessage is too short to unmarshal queryFlags")
		}
		_, wm, ok = bsoncore.ReadCString(wm)
		if !ok {
			t.Fatalf("wiremessage is too short to unmarshal fullCollectionName")
		}
		_, wm, ok = bsoncore.ReadInt32(wm)
		if !ok {
			t.Fatalf("wiremessage is too short to unmarshal numberToSkip")
		}
		_, wm, ok = bsoncore.ReadInt32(wm)
		if !ok {
			t.Fatalf("wiremessage is too short to unmarshal numberToReturn")
		}
		actualPayload, _, ok = bsoncore.ReadDocument(wm)
		if !ok {
			t.Fatalf("wiremessage is too short to unmarshal document")
		}
	case wiremessage.OpMsg:
		// Append the $db field.
		elems, err := expectedPayload.Elements()
		if err != nil {
			t.Fatalf("expectedPayload is not valid: %v", err)
		}
		elems = append(elems, bsoncore.AppendStringElement(nil, "$db", dbName))
		elems = append(elems, bsoncore.AppendDocumentElement(nil,
			"$readPreference",
			bsoncore.BuildDocumentFromElements(nil, bsoncore.AppendStringElement(nil, "mode", "primaryPreferred")),
		))
		bslc := make([][]byte, 0, len(elems)) // BuildDocumentFromElements takes a [][]byte, not a []bsoncore.Element.
		for _, elem := range elems {
			bslc = append(bslc, elem)
		}
		expectedPayload = bsoncore.BuildDocumentFromElements(nil, bslc...)

		_, wm, ok := bsoncore.ReadInt32(wm)
		if !ok {
			t.Fatalf("wiremessage is too short to unmarshal flags")
		}
	loop:
		for {
			var secType byte
			secType, wm, ok = bsoncore.ReadByte(wm)
			if !ok {
				t.Fatalf("wiremessage is too short to unmarshal type")
				break
			}
			switch stype := wiremessage.SectionType(secType); stype {
			case wiremessage.DocumentSequence:
				_, _, wm, ok = bsoncore.ReadMsgSectionDocumentSequence(wm)
				if !ok {
					t.Fatalf("wiremessage is too short to unmarshal")
					break loop
				}
			case wiremessage.SingleDocument:
				actualPayload, _, ok = bsoncore.ReadDocument(wm)
				if !ok {
					t.Fatalf("wiremessage is too short to unmarshal")
				}
				break loop
			}
		}
	}

	if !cmp.Equal(actualPayload, expectedPayload) {
		t.Errorf("Payloads don't match. got %v; want %v", actualPayload, expectedPayload)
	}
}
