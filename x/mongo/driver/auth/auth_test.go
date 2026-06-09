// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/mnet"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/wiremessage"
)

func TestCreateAuthenticator(t *testing.T) {
	tests := []struct {
		name   string
		source string
		auth   auth.Authenticator
		err    error
	}{
		{name: "", auth: &auth.DefaultAuthenticator{}},
		{name: "SCRAM-SHA-1", auth: &auth.ScramAuthenticator{}},
		{name: "SCRAM-SHA-256", auth: &auth.ScramAuthenticator{}},
		{name: "MONGODB-CR", err: errors.New(`auth mechanism "MONGODB-CR" is no longer available in any supported version of MongoDB`)},
		{name: "PLAIN", auth: &auth.PlainAuthenticator{}},
		{name: "MONGODB-X509", auth: &auth.MongoDBX509Authenticator{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cred := &auth.Cred{
				Username:    "user",
				Password:    "pencil",
				PasswordSet: true,
			}

			a, err := auth.CreateAuthenticator(test.name, cred, &http.Client{})
			if test.err != nil {
				require.EqualError(t, err, test.err.Error())
				return
			}
			require.NoError(t, err)
			require.IsType(t, test.auth, a)
		})
	}
}

func compareResponses(t *testing.T, wm []byte, expectedPayload bsoncore.Document, dbName string) {
	_, _, _, opcode, wm, ok := wiremessage.ReadHeader(wm)
	if !ok {
		t.Fatalf("wiremessage is too short to unmarshal")
	}
	var actualPayload bsoncore.Document
	if opcode == wiremessage.OpMsg {
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

		_, wm, ok := wiremessage.ReadMsgFlags(wm)
		if !ok {
			t.Fatalf("wiremessage is too short to unmarshal")
		}
	loop:
		for {
			var stype wiremessage.SectionType
			stype, wm, ok = wiremessage.ReadMsgSectionType(wm)
			if !ok {
				t.Fatalf("wiremessage is too short to unmarshal")
			}
			switch stype {
			case wiremessage.DocumentSequence:
				_, _, wm, ok = wiremessage.ReadMsgSectionDocumentSequence(wm)
				if !ok {
					t.Fatalf("wiremessage is too short to unmarshal")
				}
			case wiremessage.SingleDocument:
				actualPayload, _, ok = wiremessage.ReadMsgSectionSingleDocument(wm)
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

type testAuthenticator struct{}

func (a *testAuthenticator) Auth(context.Context, *driver.AuthConfig) error {
	return fmt.Errorf("test error")
}

func (a *testAuthenticator) Reauth(context.Context, *driver.AuthConfig) error {
	return nil
}

func TestPerformAuthentication(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                   string
		authenticateToAnything bool
		require                func(*testing.T, error)
	}{
		{
			name:                   "positive",
			authenticateToAnything: true,
			require: func(t *testing.T, err error) {
				require.EqualError(t, err, "auth error: test error")
			},
		},
		{
			name:                   "negative",
			authenticateToAnything: false,
			require: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	mnetconn := mnet.NewConnection(&drivertest.ChannelConn{})
	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handshaker := auth.Handshaker(nil, &auth.HandshakeOptions{
				Authenticator: &testAuthenticator{},
				PerformAuthentication: func(description.Server) bool {
					return tc.authenticateToAnything
				},
			})

			err := handshaker.FinishHandshake(context.Background(), mnetconn)
			tc.require(t, err)
		})
	}
}
