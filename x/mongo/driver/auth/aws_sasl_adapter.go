// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	v4signer "go.mongodb.org/mongo-driver/v2/internal/aws/signer/v4"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

type clientState int

const (
	clientStarting clientState = iota
	clientFirst
	clientFinal
	clientDone
)

type awsSaslAdapter struct {
	signer *v4signer.Signer

	state clientState
	nonce []byte
}

var _ SaslClient = (*awsSaslAdapter)(nil)

func (a *awsSaslAdapter) Start() (string, []byte, error) {
	step, err := a.step(nil)
	if err != nil {
		return MongoDBAWS, nil, err
	}
	return MongoDBAWS, step, nil
}

func (a *awsSaslAdapter) Next(_ context.Context, challenge []byte) ([]byte, error) {
	step, err := a.step(challenge)
	if err != nil {
		return nil, err
	}
	return step, nil
}

func (a *awsSaslAdapter) Completed() bool {
	return a.state == clientDone
}

const (
	amzDateFormat       = "20060102T150405Z"
	defaultRegion       = "us-east-1"
	maxHostLength       = 255
	responseNonceLength = 64
)

// step takes a string provided from a server (or just an empty string for the
// very first conversation step) and attempts to move the authentication
// conversation forward.  It returns a string to be sent to the server or an
// error if the server message is invalid.  Calling Step after a conversation
// completes is also an error.
func (a *awsSaslAdapter) step(challenge []byte) (response []byte, err error) {
	switch a.state {
	case clientStarting:
		a.state = clientFirst
		response = a.firstMsg()
	case clientFirst:
		a.state = clientFinal
		response, err = a.finalMsg(challenge)
	case clientFinal:
		a.state = clientDone
	default:
		response, err = nil, errors.New("conversation already completed")
	}
	return
}

func getRegion(host string) (string, error) {
	region := defaultRegion

	if len(host) == 0 {
		return "", errors.New("invalid STS host: empty")
	}
	if len(host) > maxHostLength {
		return "", errors.New("invalid STS host: too large")
	}
	// The implicit region for sts.amazonaws.com is us-east-1
	if host == "sts.amazonaws.com" {
		return region, nil
	}
	if strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") || strings.Contains(host, "..") {
		return "", errors.New("invalid STS host: empty part")
	}

	// If the host has multiple parts, the second part is the region
	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		region = parts[1]
	}

	return region, nil
}

func (a *awsSaslAdapter) firstMsg() []byte {
	// Values are cached for use in final message parameters
	a.nonce = make([]byte, 32)
	_, _ = rand.Read(a.nonce)

	idx, msg := bsoncore.AppendDocumentStart(nil)
	msg = bsoncore.AppendInt32Element(msg, "p", 110)
	msg = bsoncore.AppendBinaryElement(msg, "r", 0x00, a.nonce)
	msg, _ = bsoncore.AppendDocumentEnd(msg, idx)
	return msg
}

func (a *awsSaslAdapter) finalMsg(s1 []byte) ([]byte, error) {
	var sm struct {
		Nonce bson.Binary `bson:"s"`
		Host  string      `bson:"h"`
	}
	err := bson.Unmarshal(s1, &sm)
	if err != nil {
		return nil, err
	}

	// Check nonce prefix
	if sm.Nonce.Subtype != 0x00 {
		return nil, errors.New("server reply contained unexpected binary subtype")
	}
	if len(sm.Nonce.Data) != responseNonceLength {
		return nil, fmt.Errorf("server reply nonce was not %v bytes", responseNonceLength)
	}
	if !bytes.HasPrefix(sm.Nonce.Data, a.nonce) {
		return nil, errors.New("server nonce did not extend client nonce")
	}

	region, err := getRegion(sm.Host)
	if err != nil {
		return nil, err
	}

	currentTime := time.Now().UTC()
	body := "Action=GetCallerIdentity&Version=2011-06-15"

	// Create http.Request
	req, err := http.NewRequest("POST", "/", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Host = sm.Host
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Length", "43")
	req.Header.Set("X-Amz-Date", currentTime.Format(amzDateFormat))
	req.Header.Set("X-MongoDB-Server-Nonce", base64.StdEncoding.EncodeToString(sm.Nonce.Data))
	req.Header.Set("X-MongoDB-GS2-CB-Flag", "n")

	// Get signed header
	_, err = a.signer.Sign(req, strings.NewReader(body), "sts", region, currentTime)
	if err != nil {
		return nil, err
	}

	// create message
	idx, msg := bsoncore.AppendDocumentStart(nil)
	msg = bsoncore.AppendStringElement(msg, "a", req.Header.Get("Authorization"))
	msg = bsoncore.AppendStringElement(msg, "d", req.Header.Get("X-Amz-Date"))
	if sessionToken := req.Header.Get("X-Amz-Security-Token"); len(sessionToken) > 0 {
		msg = bsoncore.AppendStringElement(msg, "t", sessionToken)
	}
	msg, _ = bsoncore.AppendDocumentEnd(msg, idx)

	return msg, nil
}
