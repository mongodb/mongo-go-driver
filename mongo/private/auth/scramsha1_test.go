// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth_test

import (
	"context"
	"strings"
	"testing"

	"reflect"

	"encoding/base64"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/internal/conntest"
	"github.com/mongodb/mongo-go-driver/mongo/internal/msgtest"
	. "github.com/mongodb/mongo-go-driver/mongo/private/auth"
	"github.com/mongodb/mongo-go-driver/mongo/private/msg"
	"github.com/stretchr/testify/require"
)

func TestScramSHA1Authenticator_Fails(t *testing.T) {
	t.Parallel()

	authenticator := ScramSHA1Authenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
	}

	require.True(t, authenticator.IsClientKeyNil())

	saslStartReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", []byte{}),
		bson.EC.Int32("code", 143),
		bson.EC.Boolean("done", true)))

	conn := &conntest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply},
	}

	err := authenticator.Auth(context.Background(), conn)
	require.Error(t, err)

	errPrefix := "unable to authenticate using mechanism \"SCRAM-SHA-1\""
	require.True(t, strings.HasPrefix(err.Error(), errPrefix))
	require.True(t, authenticator.IsClientKeyNil())
}

func TestScramSHA1Authenticator_Missing_challenge_fields(t *testing.T) {
	t.Parallel()

	authenticator := ScramSHA1Authenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
		NonceGenerator: func(dst []byte) error {
			copy(dst, []byte("fyko+d2lbbFgONRv9qkxdawL"))
			return nil
		},
	}

	require.True(t, authenticator.IsClientKeyNil())

	payload, _ := base64.StdEncoding.DecodeString("cz1yUTlaWTNNbnRCZXVQM0UxVERWQzR3PT0saT0xMDAwMA===")
	saslStartReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)))

	conn := &conntest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply},
	}

	err := authenticator.Auth(context.Background(), conn)
	require.Error(t, err)

	errPrefix := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid server response"
	require.True(t, strings.HasPrefix(err.Error(), errPrefix))

	require.True(t, authenticator.IsClientKeyNil())
}

func TestScramSHA1Authenticator_Invalid_server_nonce1(t *testing.T) {
	t.Parallel()

	authenticator := ScramSHA1Authenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
		NonceGenerator: func(dst []byte) error {
			copy(dst, []byte("fyko+d2lbbFgONRv9qkxdawL"))
			return nil
		},
	}

	require.True(t, authenticator.IsClientKeyNil())

	payload, _ := base64.StdEncoding.DecodeString("bD0yMzJnLHM9clE5WlkzTW50QmV1UDNFMVREVkM0dz09LGk9MTAwMDA=")
	saslStartReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)))

	conn := &conntest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply},
	}

	err := authenticator.Auth(context.Background(), conn)

	errPrefix := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid nonce"
	require.True(t, strings.HasPrefix(err.Error(), errPrefix))

	require.True(t, authenticator.IsClientKeyNil())
}

func TestScramSHA1Authenticator_Invalid_server_nonce2(t *testing.T) {
	t.Parallel()

	authenticator := ScramSHA1Authenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
		NonceGenerator: func(dst []byte) error {
			copy(dst, []byte("fyko+d2lbbFgONRv9qkxdawL"))
			return nil
		},
	}

	require.True(t, authenticator.IsClientKeyNil())

	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvLWQybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPTEwMDAw")
	saslStartReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)))

	conn := &conntest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply},
	}

	err := authenticator.Auth(context.Background(), conn)
	require.Error(t, err)

	errPrefix := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid nonce"
	require.True(t, strings.HasPrefix(err.Error(), errPrefix))

	require.True(t, authenticator.IsClientKeyNil())
}

func TestScramSHA1Authenticator_No_salt(t *testing.T) {
	t.Parallel()

	authenticator := ScramSHA1Authenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
		NonceGenerator: func(dst []byte) error {
			copy(dst, []byte("fyko+d2lbbFgONRv9qkxdawL"))
			return nil
		},
	}

	require.True(t, authenticator.IsClientKeyNil())

	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxrPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPTEwMDAw======")
	saslStartReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)))

	conn := &conntest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply},
	}

	err := authenticator.Auth(context.Background(), conn)
	require.Error(t, err)

	errPrefix := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid salt"
	require.True(t, strings.HasPrefix(err.Error(), errPrefix))

	require.True(t, authenticator.IsClientKeyNil())
}

func TestScramSHA1Authenticator_No_iteration_count(t *testing.T) {
	t.Parallel()

	authenticator := ScramSHA1Authenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
		NonceGenerator: func(dst []byte) error {
			copy(dst, []byte("fyko+d2lbbFgONRv9qkxdawL"))
			return nil
		},
	}

	require.True(t, authenticator.IsClientKeyNil())

	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxrPXNkZg======")
	saslStartReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)))

	conn := &conntest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply},
	}

	err := authenticator.Auth(context.Background(), conn)
	require.Error(t, err)

	errPrefix := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid iteration count"
	require.True(t, strings.HasPrefix(err.Error(), errPrefix))

	require.True(t, authenticator.IsClientKeyNil())
}

func TestScramSHA1Authenticator_Invalid_iteration_count(t *testing.T) {
	t.Parallel()

	authenticator := ScramSHA1Authenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
		NonceGenerator: func(dst []byte) error {
			copy(dst, []byte("fyko+d2lbbFgONRv9qkxdawL"))
			return nil
		},
	}

	require.True(t, authenticator.IsClientKeyNil())

	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPWFiYw====")
	saslStartReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)))

	conn := &conntest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply},
	}

	err := authenticator.Auth(context.Background(), conn)
	require.Error(t, err)

	errPrefix := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid iteration count"
	require.True(t, strings.HasPrefix(err.Error(), errPrefix))

	require.True(t, authenticator.IsClientKeyNil())
}

func TestScramSHA1Authenticator_Invalid_server_signature(t *testing.T) {
	t.Parallel()

	authenticator := ScramSHA1Authenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
		NonceGenerator: func(dst []byte) error {
			copy(dst, []byte("fyko+d2lbbFgONRv9qkxdawL"))
			return nil
		},
	}

	require.True(t, authenticator.IsClientKeyNil())

	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPTEwMDAw")
	saslStartReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)))
	payload, _ = base64.StdEncoding.DecodeString("dj1VTVdlSTI1SkQxeU5ZWlJNcFo0Vkh2aFo5ZTBh")
	saslContinueReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)))

	conn := &conntest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply, saslContinueReply},
	}

	err := authenticator.Auth(context.Background(), conn)
	require.Error(t, err)

	errPrefix := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid server signature"
	require.True(t, strings.HasPrefix(err.Error(), errPrefix))

	require.True(t, authenticator.IsClientKeyNil())
}

func TestScramSHA1Authenticator_Server_provided_error(t *testing.T) {
	t.Parallel()

	authenticator := ScramSHA1Authenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
		NonceGenerator: func(dst []byte) error {
			copy(dst, []byte("fyko+d2lbbFgONRv9qkxdawL"))
			return nil
		},
	}

	require.True(t, authenticator.IsClientKeyNil())

	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPTEwMDAw")
	saslStartReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)))
	payload, _ = base64.StdEncoding.DecodeString("ZT1zZXJ2ZXIgcGFzc2VkIGVycm9y")
	saslContinueReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)))

	conn := &conntest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply, saslContinueReply},
	}

	err := authenticator.Auth(context.Background(), conn)
	require.Error(t, err)

	errPrefix := "unable to authenticate using mechanism \"SCRAM-SHA-1\": server passed error"
	require.True(t, strings.HasPrefix(err.Error(), errPrefix))

	require.True(t, authenticator.IsClientKeyNil())
}

func TestScramSHA1Authenticator_Invalid_final_message(t *testing.T) {
	t.Parallel()

	authenticator := ScramSHA1Authenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
		NonceGenerator: func(dst []byte) error {
			copy(dst, []byte("fyko+d2lbbFgONRv9qkxdawL"))
			return nil
		},
	}

	require.True(t, authenticator.IsClientKeyNil())

	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPTEwMDAw")
	saslStartReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)))
	payload, _ = base64.StdEncoding.DecodeString("Zj1VTVdlSTI1SkQxeU5ZWlJNcFo0Vkh2aFo5ZTBh")
	saslContinueReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)))

	conn := &conntest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply, saslContinueReply},
	}

	err := authenticator.Auth(context.Background(), conn)
	require.Error(t, err)

	errPrefix := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid final message"
	require.True(t, strings.HasPrefix(err.Error(), errPrefix))

	require.True(t, authenticator.IsClientKeyNil())
}

func TestScramSHA1Authenticator_Extra_message(t *testing.T) {
	t.Parallel()

	authenticator := ScramSHA1Authenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
		NonceGenerator: func(dst []byte) error {
			copy(dst, []byte("fyko+d2lbbFgONRv9qkxdawL"))
			return nil
		},
	}

	require.True(t, authenticator.IsClientKeyNil())

	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPTEwMDAw")
	saslStartReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)))
	payload, _ = base64.StdEncoding.DecodeString("dj1VTVdlSTI1SkQxeU5ZWlJNcFo0Vkh2aFo5ZTA9")
	saslContinueReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)))
	saslContinueReply2 := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", []byte{}),
		bson.EC.Boolean("done", false)))

	conn := &conntest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply, saslContinueReply, saslContinueReply2},
	}

	err := authenticator.Auth(context.Background(), conn)
	require.Error(t, err)

	errPrefix := "unable to authenticate using mechanism \"SCRAM-SHA-1\": unexpected server challenge"
	require.True(t, strings.HasPrefix(err.Error(), errPrefix))

	require.True(t, authenticator.IsClientKeyNil())
}

func TestScramSHA1Authenticator_Succeeds(t *testing.T) {
	t.Parallel()

	authenticator := ScramSHA1Authenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
		NonceGenerator: func(dst []byte) error {
			copy(dst, []byte("fyko+d2lbbFgONRv9qkxdawL"))
			return nil
		},
	}

	require.True(t, authenticator.IsClientKeyNil())

	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPTEwMDAw")
	saslStartReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)))
	payload, _ = base64.StdEncoding.DecodeString("dj1VTVdlSTI1SkQxeU5ZWlJNcFo0Vkh2aFo5ZTA9")
	saslContinueReply := msgtest.CreateCommandReply(bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", true)))

	conn := &conntest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply, saslContinueReply},
	}

	err := authenticator.Auth(context.Background(), conn)
	require.NoError(t, err)

	require.Len(t, conn.Sent, 2)

	saslStartRequest := conn.Sent[0].(*msg.Query)
	payload, _ = base64.RawStdEncoding.DecodeString("biwsbj11c2VyLHI9ZnlrbytkMmxiYkZnT05Sdjlxa3hkYXdM")
	expectedCmd := bson.NewDocument(
		bson.EC.Int32("saslStart", 1),
		bson.EC.String("mechanism", "SCRAM-SHA-1"),
		bson.EC.Binary("payload", payload))

	require.True(t, reflect.DeepEqual(saslStartRequest.Query, expectedCmd))

	saslContinueRequest := conn.Sent[1].(*msg.Query)
	payload, _ = base64.RawStdEncoding.DecodeString("Yz1iaXdzLHI9ZnlrbytkMmxiYkZnT05Sdjlxa3hkYXdMSG8rVmdrN3F2VU9LVXd1V0xJV2c0bC85U3JhR01IRUUscD1NQzJUOEJ2Ym1XUmNrRHc4b1dsNUlWZ2h3Q1k9")
	expectedCmd = bson.NewDocument(
		bson.EC.Int32("saslContinue", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload))

	require.True(t, reflect.DeepEqual(saslContinueRequest.Query, expectedCmd))

	require.False(t, authenticator.IsClientKeyNil())
}
