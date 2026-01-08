// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/aws/credentials"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/mnet"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/wiremessage"
)

func TestGetRegion(t *testing.T) {
	longHost := make([]rune, 256)
	emptyErr := errors.New("invalid STS host: empty")
	tooLongErr := errors.New("invalid STS host: too large")
	emptyPartErr := errors.New("invalid STS host: empty part")
	testCases := []struct {
		name   string
		host   string
		err    error
		region string
	}{
		{"success default", "sts.amazonaws.com", nil, "us-east-1"},
		{"success parse", "first.second", nil, "second"},
		{"success no region", "first", nil, "us-east-1"},
		{"error host too long", string(longHost), tooLongErr, ""},
		{"error host empty", "", emptyErr, ""},
		{"error empty middle part", "abc..def", emptyPartErr, ""},
		{"error empty part", "first.", emptyPartErr, ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reg, err := getRegion(tc.host)
			if tc.err == nil {
				assert.Nil(t, err, "error getting region: %v", err)
				assert.Equal(t, tc.region, reg, "expected %v, got %v", tc.region, reg)
				return
			}
			assert.NotNil(t, err, "expected error, got nil")
			assert.Equal(t, err, tc.err, "expected error: %v, got: %v", tc.err, err)
		})
	}

}

type awsCredentialsProvider struct {
	cnt int
}

func (a *awsCredentialsProvider) Retrieve(_ context.Context) (credentials.Value, error) {
	a.cnt++
	return credentials.Value{}, nil
}

func (*awsCredentialsProvider) IsExpired() bool {
	return false
}

func TestAWSCustomCredentialProvider(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "AWS_ACCESS_KEY_ID")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "AWS_SECRET_ACCESS_KEY")

	provider := &awsCredentialsProvider{}
	for _, tc := range []struct {
		name string
		cred *Cred
		cnt  int
	}{
		{
			name: "provider with cred",
			cred: &Cred{
				Username:               "user",
				Password:               "pass",
				Props:                  map[string]string{"AWS_SESSION_TOKEN": "token"},
				AWSCredentialsProvider: provider,
			},
			cnt: 0,
		},
		{
			name: "provider with empty cred",
			cred: &Cred{
				AWSCredentialsProvider: provider,
			},
			cnt: 1,
		},
	} {
		provider.cnt = 0
		t.Run(tc.name, func(t *testing.T) {
			authenticator, err := newMongoDBAWSAuthenticator(
				tc.cred,
				&http.Client{},
			)
			require.NoErrorf(t, err, "unexpected error %v", err)

			resps := make(chan []byte, 1)
			written := make(chan []byte, 1)
			var readErr chan error
			go func() {
				for {
					processWm(resps, written, readErr)
				}
			}()

			desc := description.Server{
				WireVersion: &description.VersionRange{
					Max: 6,
				},
			}
			c := &drivertest.ChannelConn{
				Written:  written,
				ReadResp: resps,
				ReadErr:  readErr,
				Desc:     desc,
			}

			mnetconn := mnet.NewConnection(c)

			err = authenticator.Auth(context.Background(), &driver.AuthConfig{Connection: mnetconn})
			assert.NoErrorf(t, err, "expected no error but got %v", err)
			assert.Equalf(t, tc.cnt, provider.cnt, "expected provider to be called %v times but got %v", tc.cnt, provider.cnt)
		})
	}
}

func processWm(resps, written chan []byte, errChan chan error) {
	buf := <-written
	buf, ok := extractPayload(buf)
	if !ok {
		errChan <- errors.New("could not extract payload from message")
	}
	var p struct {
		Payload bson.Binary `bson:"payload"`
	}
	err := bson.Unmarshal(buf, &p)
	if err != nil {
		errChan <- err
	}
	if p.Payload.Subtype != 0x00 {
		errChan <- errors.New("unexpected payload subtype")
	}
	var n struct {
		Nonce bson.Binary `bson:"r"`
	}
	err = bson.Unmarshal(p.Payload.Data, &n)
	if err != nil {
		errChan <- err
	}
	if n.Nonce.Subtype != 0x00 {
		errChan <- errors.New("unexpected nonce subtype")
	}
	nonce := make([]byte, 64)
	copy(nonce, n.Nonce.Data)

	writeReplies(resps,
		bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendInt32Element(nil, "ok", 1),
			bsoncore.AppendInt32Element(nil, "conversationId", 1),
			bsoncore.AppendBinaryElement(nil, "payload", 0x00, bsoncore.BuildDocumentFromElements(nil,
				bsoncore.AppendBinaryElement(nil, "s", n.Nonce.Subtype, nonce),
				bsoncore.AppendStringElement(nil, "h", "region"),
			)),
			bsoncore.AppendBooleanElement(nil, "done", true),
		),
	)
}

func extractPayload(wm []byte) (bsoncore.Document, bool) {
	_, _, _, opcode, wm, ok := wiremessage.ReadHeader(wm)
	if !ok {
		return nil, ok
	}
	if opcode != wiremessage.OpMsg {
		return nil, false
	}
	var actualPayload bsoncore.Document
	_, wm, ok = wiremessage.ReadMsgFlags(wm)
	if !ok {
		return nil, ok
	}
	for loop := true; loop; {
		var stype wiremessage.SectionType
		stype, wm, ok = wiremessage.ReadMsgSectionType(wm)
		if !ok {
			return nil, ok
		}
		switch stype {
		case wiremessage.DocumentSequence:
			_, _, wm, ok = wiremessage.ReadMsgSectionDocumentSequence(wm)
			if !ok {
				return nil, ok
			}
		case wiremessage.SingleDocument:
			actualPayload, _, ok = wiremessage.ReadMsgSectionSingleDocument(wm)
			if !ok {
				return nil, ok
			}
			loop = false
		}
	}
	return actualPayload, true
}
