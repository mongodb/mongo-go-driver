// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"io"
	"math/rand"
	"strconv"

	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
	"golang.org/x/crypto/pbkdf2"

	"strings"

	"encoding/base64"
)

// SCRAMSHA1 is the mechanism name for SCRAM-SHA-1.
const SCRAMSHA1 = "SCRAM-SHA-1"
const scramSHA1NonceLen = 24

var usernameSanitizer = strings.NewReplacer("=", "=3D", ",", "=2D")

func newScramSHA1Authenticator(cred *Cred) (Authenticator, error) {
	return &ScramSHA1Authenticator{
		DB:       cred.Source,
		Username: cred.Username,
		Password: cred.Password,
	}, nil
}

// ScramSHA1Authenticator uses the SCRAM-SHA-1 algorithm over SASL to authenticate a connection.
type ScramSHA1Authenticator struct {
	DB        string
	Username  string
	Password  string
	clientKey []byte

	NonceGenerator func([]byte) error
}

// Auth authenticates the connection.
func (a *ScramSHA1Authenticator) Auth(ctx context.Context, desc description.Server, rw wiremessage.ReadWriter) error {
	client := &scramSaslClient{
		username:       a.Username,
		password:       a.Password,
		nonceGenerator: a.NonceGenerator,
		clientKey:      a.clientKey,
	}

	err := ConductSaslConversation(ctx, desc, rw, a.DB, client)
	if err != nil {
		return newAuthError("sasl conversation error", err)
	}

	a.clientKey = client.clientKey

	return nil
}

type scramSaslClient struct {
	username       string
	password       string
	nonceGenerator func([]byte) error
	clientKey      []byte

	step                   uint8
	clientNonce            []byte
	clientFirstMessageBare string
	serverSignature        []byte
}

func (c *scramSaslClient) Start() (string, []byte, error) {
	if err := c.generateClientNonce(scramSHA1NonceLen); err != nil {
		return SCRAMSHA1, nil, newAuthError("generate nonce error", err)
	}

	c.clientFirstMessageBare = "n=" + usernameSanitizer.Replace(c.username) + ",r=" + string(c.clientNonce)

	return SCRAMSHA1, []byte("n,," + c.clientFirstMessageBare), nil
}

func (c *scramSaslClient) Next(challenge []byte) ([]byte, error) {
	c.step++
	switch c.step {
	case 1:
		return c.step1(challenge)
	case 2:
		return c.step2(challenge)
	default:
		return nil, newAuthError("unexpected server challenge", nil)
	}
}

func (c *scramSaslClient) Completed() bool {
	return c.step >= 2
}

func (c *scramSaslClient) step1(challenge []byte) ([]byte, error) {
	fields := bytes.Split(challenge, []byte{','})
	if len(fields) != 3 {
		return nil, newAuthError("invalid server response", nil)
	}

	if !bytes.HasPrefix(fields[0], []byte("r=")) || len(fields[0]) < 2 {
		return nil, newAuthError("invalid nonce", nil)
	}
	r := fields[0][2:]
	if !bytes.HasPrefix(r, c.clientNonce) {
		return nil, newAuthError("invalid nonce", nil)
	}

	if !bytes.HasPrefix(fields[1], []byte("s=")) || len(fields[1]) < 6 {
		return nil, newAuthError("invalid salt", nil)
	}
	s := make([]byte, base64.StdEncoding.DecodedLen(len(fields[1][2:])))
	n, err := base64.StdEncoding.Decode(s, fields[1][2:])
	if err != nil {
		return nil, newAuthError("invalid salt", nil)
	}
	s = s[:n]

	if !bytes.HasPrefix(fields[2], []byte("i=")) || len(fields[2]) < 3 {
		return nil, newAuthError("invalid iteration count", nil)
	}
	i, err := strconv.Atoi(string(fields[2][2:]))
	if err != nil {
		return nil, newAuthError("invalid iteration count", nil)
	}

	clientFinalMessageWithoutProof := "c=biws,r=" + string(r)
	authMessage := c.clientFirstMessageBare + "," + string(challenge) + "," + clientFinalMessageWithoutProof

	saltedPassword := pbkdf2.Key([]byte(mongoPasswordDigest(c.username, c.password)), s, i, 20, sha1.New)

	if c.clientKey == nil {
		c.clientKey = c.hmac(saltedPassword, "Client Key")
	}

	storedKey := c.h(c.clientKey)
	clientSignature := c.hmac(storedKey, authMessage)
	clientProof := c.xor(c.clientKey, clientSignature)
	serverKey := c.hmac(saltedPassword, "Server Key")
	c.serverSignature = c.hmac(serverKey, authMessage)

	proof := "p=" + base64.StdEncoding.EncodeToString(clientProof)
	clientFinalMessage := clientFinalMessageWithoutProof + "," + proof

	return []byte(clientFinalMessage), nil
}

func (c *scramSaslClient) step2(challenge []byte) ([]byte, error) {
	var hasV, hasE bool
	fields := bytes.Split(challenge, []byte{','})
	if len(fields) == 1 {
		hasV = bytes.HasPrefix(fields[0], []byte("v="))
		hasE = bytes.HasPrefix(fields[0], []byte("e="))
	}
	if hasE {
		return nil, newAuthError(string(fields[0][2:]), nil)
	}
	if !hasV {
		return nil, newAuthError("invalid final message", nil)
	}

	v := make([]byte, base64.StdEncoding.DecodedLen(len(fields[0][2:])))
	n, err := base64.StdEncoding.Decode(v, fields[0][2:])
	if err != nil {
		return nil, newAuthError("invalid server verification", nil)
	}
	v = v[:n]

	if !bytes.Equal(c.serverSignature, v) {
		return nil, newAuthError("invalid server signature", nil)
	}

	return nil, nil
}

func (c *scramSaslClient) generateClientNonce(n uint8) error {
	if c.nonceGenerator != nil {
		c.clientNonce = make([]byte, n)
		return c.nonceGenerator(c.clientNonce)
	}

	buf := make([]byte, n)
	rand.Read(buf)

	c.clientNonce = make([]byte, base64.StdEncoding.EncodedLen(int(n)))
	base64.StdEncoding.Encode(c.clientNonce, buf)
	return nil
}

func (c *scramSaslClient) h(data []byte) []byte {
	h := sha1.New()
	_, _ = h.Write(data)
	return h.Sum(nil)
}

func (c *scramSaslClient) hmac(data []byte, key string) []byte {
	h := hmac.New(sha1.New, data)
	_, _ = io.WriteString(h, key)
	return h.Sum(nil)
}

func (c *scramSaslClient) xor(a []byte, b []byte) []byte {
	result := make([]byte, len(a))
	for i := 0; i < len(a); i++ {
		result[i] = a[i] ^ b[i]
	}
	return result
}
