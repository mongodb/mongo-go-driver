// Copyright (C) MongoDB, Inc. 2018-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"context"
	"fmt"

	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/wiremessage"
	"github.com/xdg/scram"
	"github.com/xdg/stringprep"
)

// SCRAMSHA256 is the mechanism name for SCRAM-SHA-256.
const SCRAMSHA256 = "SCRAM-SHA-256"

func newScramSHA256Authenticator(cred *Cred) (Authenticator, error) {
	passprep, err := stringprep.SASLprep.Prepare(cred.Password)
	if err != nil {
		return nil, newAuthError(fmt.Sprintf("error SASLprepping password '%s'", cred.Password), err)
	}
	client, err := scram.SHA256.NewClientUnprepped(cred.Username, passprep, "")
	if err != nil {
		return nil, newAuthError("error initializing SCRAM-SHA-256 client", err)
	}
	client.WithMinIterations(4096)
	return &ScramSHA256Authenticator{
		DB:     cred.Source,
		client: client,
	}, nil
}

// ScramSHA256Authenticator uses the SCRAM-SHA-256 algorithm over SASL to authenticate a connection.
type ScramSHA256Authenticator struct {
	DB     string
	client *scram.Client
}

// Auth authenticates the connection.
func (a *ScramSHA256Authenticator) Auth(ctx context.Context, desc description.Server, rw wiremessage.ReadWriter) error {
	adapter := &scramSaslAdapter{conversation: a.client.NewConversation()}
	err := ConductSaslConversation(ctx, desc, rw, a.DB, adapter)
	if err != nil {
		return newAuthError("sasl conversation error", err)
	}
	return nil
}

type scramSaslAdapter struct {
	conversation *scram.ClientConversation
}

func (a *scramSaslAdapter) Start() (string, []byte, error) {
	step, err := a.conversation.Step("")
	if err != nil {
		return SCRAMSHA256, nil, err
	}
	return SCRAMSHA256, []byte(step), nil
}

func (a *scramSaslAdapter) Next(challenge []byte) ([]byte, error) {
	step, err := a.conversation.Step(string(challenge))
	if err != nil {
		return nil, err
	}
	return []byte(step), nil
}

func (a *scramSaslAdapter) Completed() bool {
	return a.conversation.Done()
}
