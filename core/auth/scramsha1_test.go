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
	. "github.com/mongodb/mongo-go-driver/core/auth"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
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

	resps := make(chan wiremessage.WireMessage, 1)
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", []byte{}),
		bson.EC.Int32("code", 143),
		bson.EC.Boolean("done", true)),
	)

	c := &conn{written: make(chan wiremessage.WireMessage, 1), readResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{}, c)
	require.Error(t, err)

	errSubstring := "unable to authenticate using mechanism \"SCRAM-SHA-1\""
	require.True(t, strings.Contains(err.Error(), errSubstring))
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

	resps := make(chan wiremessage.WireMessage, 1)
	payload, _ := base64.StdEncoding.DecodeString("cz1yUTlaWTNNbnRCZXVQM0UxVERWQzR3PT0saT0xMDAwMA===")
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)),
	)

	c := &conn{written: make(chan wiremessage.WireMessage, 1), readResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{}, c)
	require.Error(t, err)

	errSubstring := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid server response"
	require.True(t, strings.Contains(err.Error(), errSubstring))

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

	resps := make(chan wiremessage.WireMessage, 1)
	payload, _ := base64.StdEncoding.DecodeString("bD0yMzJnLHM9clE5WlkzTW50QmV1UDNFMVREVkM0dz09LGk9MTAwMDA=")
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)),
	)

	c := &conn{written: make(chan wiremessage.WireMessage, 1), readResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{}, c)

	errSubstring := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid nonce"
	require.True(t, strings.Contains(err.Error(), errSubstring))

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

	resps := make(chan wiremessage.WireMessage, 1)
	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvLWQybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPTEwMDAw")
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)),
	)

	c := &conn{written: make(chan wiremessage.WireMessage, 1), readResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{}, c)
	require.Error(t, err)

	errSubstring := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid nonce"
	require.True(t, strings.Contains(err.Error(), errSubstring))

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

	resps := make(chan wiremessage.WireMessage, 1)
	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxrPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPTEwMDAw======")
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)),
	)

	c := &conn{written: make(chan wiremessage.WireMessage, 1), readResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{}, c)
	require.Error(t, err)

	errSubstring := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid salt"
	require.True(t, strings.Contains(err.Error(), errSubstring))

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

	resps := make(chan wiremessage.WireMessage, 1)
	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxrPXNkZg======")
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)),
	)

	c := &conn{written: make(chan wiremessage.WireMessage, 1), readResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{}, c)
	require.Error(t, err)

	errSubstring := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid iteration count"
	require.True(t, strings.Contains(err.Error(), errSubstring))

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

	resps := make(chan wiremessage.WireMessage, 1)
	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPWFiYw====")
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)),
	)

	c := &conn{written: make(chan wiremessage.WireMessage, 1), readResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{}, c)
	require.Error(t, err)

	errSubstring := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid iteration count"
	require.True(t, strings.Contains(err.Error(), errSubstring))

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

	resps := make(chan wiremessage.WireMessage, 2)
	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPTEwMDAw")
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)),
	)
	payload, _ = base64.StdEncoding.DecodeString("dj1VTVdlSTI1SkQxeU5ZWlJNcFo0Vkh2aFo5ZTBh")
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)),
	)

	c := &conn{written: make(chan wiremessage.WireMessage, 2), readResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{}, c)
	require.Error(t, err)

	errSubstring := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid server signature"
	require.True(t, strings.Contains(err.Error(), errSubstring))

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

	resps := make(chan wiremessage.WireMessage, 2)
	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPTEwMDAw")
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)),
	)
	payload, _ = base64.StdEncoding.DecodeString("ZT1zZXJ2ZXIgcGFzc2VkIGVycm9y")
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)),
	)

	c := &conn{written: make(chan wiremessage.WireMessage, 2), readResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{}, c)
	require.Error(t, err)

	errSubstring := "unable to authenticate using mechanism \"SCRAM-SHA-1\": server passed error"
	require.True(t, strings.Contains(err.Error(), errSubstring))

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

	resps := make(chan wiremessage.WireMessage, 2)
	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPTEwMDAw")
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)),
	)
	payload, _ = base64.StdEncoding.DecodeString("Zj1VTVdlSTI1SkQxeU5ZWlJNcFo0Vkh2aFo5ZTBh")
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)),
	)

	c := &conn{written: make(chan wiremessage.WireMessage, 2), readResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{}, c)
	require.Error(t, err)

	errSubstring := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid final message"
	require.True(t, strings.Contains(err.Error(), errSubstring))

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

	resps := make(chan wiremessage.WireMessage, 3)
	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPTEwMDAw")
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)),
	)
	payload, _ = base64.StdEncoding.DecodeString("dj1VTVdlSTI1SkQxeU5ZWlJNcFo0Vkh2aFo5ZTA9")
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)),
	)
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", []byte{}),
		bson.EC.Boolean("done", false)),
	)

	c := &conn{written: make(chan wiremessage.WireMessage, 3), readResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{}, c)
	require.Error(t, err)

	errSubstring := "unable to authenticate using mechanism \"SCRAM-SHA-1\": unexpected server challenge"
	require.True(t, strings.Contains(err.Error(), errSubstring))

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

	resps := make(chan wiremessage.WireMessage, 2)
	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPTEwMDAw")
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", false)),
	)
	payload, _ = base64.StdEncoding.DecodeString("dj1VTVdlSTI1SkQxeU5ZWlJNcFo0Vkh2aFo5ZTA9")
	resps <- makeReply(t, bson.NewDocument(
		bson.EC.Int32("ok", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
		bson.EC.Boolean("done", true)),
	)

	c := &conn{written: make(chan wiremessage.WireMessage, 2), readResp: resps}

	err := authenticator.Auth(context.Background(), description.Server{}, c)
	require.NoError(t, err)

	require.Len(t, c.written, 2)

	saslStartRequest := (<-c.written).(wiremessage.Query)
	payload, _ = base64.RawStdEncoding.DecodeString("biwsbj11c2VyLHI9ZnlrbytkMmxiYkZnT05Sdjlxa3hkYXdM")
	var expectedCmd bson.Reader
	expectedCmd, err = bson.NewDocument(
		bson.EC.Int32("saslStart", 1),
		bson.EC.String("mechanism", "SCRAM-SHA-1"),
		bson.EC.Binary("payload", payload),
	).MarshalBSON()
	if err != nil {
		t.Fatalf("couldn't marshal bson: %v", err)
	}

	require.True(t, reflect.DeepEqual(saslStartRequest.Query, expectedCmd))

	saslContinueRequest := (<-c.written).(wiremessage.Query)
	payload, _ = base64.RawStdEncoding.DecodeString("Yz1iaXdzLHI9ZnlrbytkMmxiYkZnT05Sdjlxa3hkYXdMSG8rVmdrN3F2VU9LVXd1V0xJV2c0bC85U3JhR01IRUUscD1NQzJUOEJ2Ym1XUmNrRHc4b1dsNUlWZ2h3Q1k9")
	expectedCmd, err = bson.NewDocument(
		bson.EC.Int32("saslContinue", 1),
		bson.EC.Int32("conversationId", 1),
		bson.EC.Binary("payload", payload),
	).MarshalBSON()
	if err != nil {
		t.Fatalf("couldn't marshal bson: %v", err)
	}

	require.True(t, reflect.DeepEqual(saslContinueRequest.Query, expectedCmd))

	require.False(t, authenticator.IsClientKeyNil())
}
