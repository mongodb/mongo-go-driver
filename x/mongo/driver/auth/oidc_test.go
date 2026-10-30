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
	"regexp"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/mongo/address"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/mnet"
)

// oidcTestAddressConn is a drivertest.ChannelConn with a configurable address.
type oidcTestAddressConn struct {
	*drivertest.ChannelConn
	addr address.Address
}

func (c *oidcTestAddressConn) Address() address.Address { return c.addr }

func TestOIDCAllowedHosts(t *testing.T) {
	// A saslStart reply completing a one-step conversation.
	saslDoneResponse := bsoncore.BuildDocumentFromElements(nil,
		bsoncore.AppendInt32Element(nil, "conversationId", 1),
		bsoncore.AppendBinaryElement(nil, "payload", 0x00, []byte{}),
		bsoncore.AppendBooleanElement(nil, "done", true),
		bsoncore.AppendInt32Element(nil, "ok", 1),
	)

	newHumanAuthenticator := func(t *testing.T) *OIDCAuthenticator {
		t.Helper()

		humanCallback := func(context.Context, *OIDCArgs) (*OIDCCredential, error) {
			t.Error("human callback should not be invoked")
			return nil, errors.New("human callback should not be invoked")
		}
		authenticator, err := CreateAuthenticator(
			MongoDBOIDC,
			&Cred{Source: "$external", OIDCHumanCallback: humanCallback},
			&http.Client{})
		assert.Nil(t, err, "CreateAuthenticator error: %v", err)
		oa, ok := authenticator.(*OIDCAuthenticator)
		assert.True(t, ok, "expected *OIDCAuthenticator, got %T", authenticator)
		return oa
	}

	t.Run("cached access token is not sent to a disallowed host", func(t *testing.T) {
		oa := newHumanAuthenticator(t)
		oa.SetAccessToken("accessToken")

		responses := make(chan []byte, 1)
		writeReplies(responses, saslDoneResponse)
		chanconn := &drivertest.ChannelConn{
			Written:  make(chan []byte, 1),
			ReadResp: responses,
			Desc: description.Server{
				WireVersion: &description.VersionRange{Max: 21},
			},
		}
		conn := mnet.NewConnection(&oidcTestAddressConn{chanconn, address.Address("evil.invalid:27017")})

		err := oa.Auth(context.Background(), &driver.AuthConfig{Connection: conn})
		assert.NotNil(t, err, "expected Auth error for a host not in %q", AllowedHostsProp)
		assert.ErrorContains(t, err, AllowedHostsProp)
		assert.Equal(t, 0, len(chanconn.Written),
			"expected no wire messages to be sent to a disallowed host, got %d", len(chanconn.Written))
	})

	t.Run("speculative authentication is skipped for a disallowed host", func(t *testing.T) {
		oa := newHumanAuthenticator(t)
		oa.SetAccessToken("accessToken")

		handshaker := Handshaker(nil, &HandshakeOptions{Authenticator: oa})
		responses := make(chan []byte, 1)
		writeReplies(responses, bsoncore.BuildDocumentFromElements(nil, handshakeHelloElements...))
		chanconn := &drivertest.ChannelConn{
			Written:  make(chan []byte, 2),
			ReadResp: responses,
		}
		conn := mnet.NewConnection(&oidcTestAddressConn{chanconn, address.Address("evil.invalid:27017")})

		info, err := handshaker.GetHandshakeInformation(context.Background(), address.Address("evil.invalid:27017"), conn)
		assert.Nil(t, err, "GetHandshakeInformation error: %v", err)

		helloCmd, err := drivertest.GetCommandFromQueryWireMessage(<-chanconn.Written)
		assert.Nil(t, err, "error parsing hello command: %v", err)
		_, lookupErr := helloCmd.LookupErr("speculativeAuthenticate")
		assert.NotNil(t, lookupErr, "expected hello to a disallowed host to not contain 'speculativeAuthenticate'")

		// The connection then fails client-side before a credential is sent.
		chanconn.Desc = info.Description
		finishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = handshaker.FinishHandshake(finishCtx, conn)
		assert.NotNil(t, err, "expected FinishHandshake error for a host not in %q", AllowedHostsProp)
		assert.ErrorContains(t, err, AllowedHostsProp)
		assert.Equal(t, 0, len(chanconn.Written),
			"expected no wire messages to be sent to a disallowed host after the hello, got %d", len(chanconn.Written))
	})

	t.Run("cached access token is sent to an allowed host", func(t *testing.T) {
		oa := newHumanAuthenticator(t)
		oa.SetAccessToken("accessToken")

		handshaker := Handshaker(nil, &HandshakeOptions{Authenticator: oa})
		responses := make(chan []byte, 2)
		writeReplies(responses, bsoncore.BuildDocumentFromElements(nil, handshakeHelloElements...), saslDoneResponse)
		chanconn := &drivertest.ChannelConn{
			Written:  make(chan []byte, 2),
			ReadResp: responses,
		}
		conn := mnet.NewConnection(&oidcTestAddressConn{chanconn, address.Address("cluster0.mongodb.net:27017")})

		info, err := handshaker.GetHandshakeInformation(context.Background(), address.Address("cluster0.mongodb.net:27017"), conn)
		assert.Nil(t, err, "GetHandshakeInformation error: %v", err)

		helloCmd, err := drivertest.GetCommandFromQueryWireMessage(<-chanconn.Written)
		assert.Nil(t, err, "error parsing hello command: %v", err)
		_, lookupErr := helloCmd.LookupErr("speculativeAuthenticate")
		assert.Nil(t, lookupErr, "expected hello to an allowed host to contain 'speculativeAuthenticate'")

		chanconn.Desc = info.Description
		err = handshaker.FinishHandshake(context.Background(), conn)
		assert.Nil(t, err, "FinishHandshake error: %v", err)

		saslStartCmd, err := drivertest.GetCommandFromMsgWireMessage(<-chanconn.Written)
		assert.Nil(t, err, "error parsing saslStart command: %v", err)
		cmdName := saslStartCmd.Index(0).Key()
		assert.Equal(t, "saslStart", cmdName, "cmd name mismatch; expected 'saslStart', got %v", cmdName)
	})
}

func TestCreatePatternsForGlobs(t *testing.T) {
	t.Run("transform allowedHosts patterns", func(t *testing.T) {
		hosts := []string{
			"*.mongodb.net",
			"*.mongodb-qa.net",
			"*.mongodb-dev.net",
			"*.mongodbgov.net",
			"localhost",
			"127.0.0.1",
			"::1",
			"*.mongo.com",
		}

		check, err := createPatternsForGlobs(hosts)
		assert.NoError(t, err)
		assert.Equal(t,
			[]*regexp.Regexp{
				regexp.MustCompile(`^.*[.]mongodb[.]net(:\d+)?$`),
				regexp.MustCompile(`^.*[.]mongodb-qa[.]net(:\d+)?$`),
				regexp.MustCompile(`^.*[.]mongodb-dev[.]net(:\d+)?$`),
				regexp.MustCompile(`^.*[.]mongodbgov[.]net(:\d+)?$`),
				regexp.MustCompile(`^localhost(:\d+)?$`),
				regexp.MustCompile(`^127[.]0[.]0[.]1(:\d+)?$`),
				regexp.MustCompile(`^::1(:\d+)?$`),
				regexp.MustCompile(`^.*[.]mongo[.]com(:\d+)?$`),
			},
			check,
		)
	})
}
