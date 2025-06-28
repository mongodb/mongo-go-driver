// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"context"
	"fmt"
	"net/http"

	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
)

func newDefaultAuthenticator(cred *Cred, httpClient *http.Client) (Authenticator, error) {
	scram, err := newScramSHA256Authenticator(cred, httpClient)
	if err != nil {
		return nil, newAuthError("failed to create internal authenticator", err)
	}
	speculative, ok := scram.(SpeculativeAuthenticator)
	if !ok {
		typeErr := fmt.Errorf("expected SCRAM authenticator to be SpeculativeAuthenticator but got %T", scram)
		return nil, newAuthError("failed to create internal authenticator", typeErr)
	}

	return &DefaultAuthenticator{
		Cred:                     cred,
		speculativeAuthenticator: speculative,
		httpClient:               httpClient,
	}, nil
}

// DefaultAuthenticator uses SCRAM-SHA-1 or SCRAM-SHA-256, depending on the
// server's SASL supported mechanisms.
type DefaultAuthenticator struct {
	Cred *Cred

	// The authenticator to use for speculative authentication. Because the correct auth mechanism is unknown when doing
	// the initial hello, SCRAM-SHA-256 is used for the speculative attempt.
	speculativeAuthenticator SpeculativeAuthenticator

	httpClient *http.Client
}

var _ SpeculativeAuthenticator = (*DefaultAuthenticator)(nil)

// CreateSpeculativeConversation creates a speculative conversation for SCRAM authentication.
func (a *DefaultAuthenticator) CreateSpeculativeConversation() (SpeculativeConversation, error) {
	return a.speculativeAuthenticator.CreateSpeculativeConversation()
}

// Auth authenticates the connection.
func (a *DefaultAuthenticator) Auth(ctx context.Context, cfg *driver.AuthConfig) error {
	actual, err := func() (Authenticator, error) {
		// If a server provides a list of supported mechanisms, we choose
		// SCRAM-SHA-256 if it exists or else MUST use SCRAM-SHA-1.
		// Otherwise, we decide based on what is supported.
		if saslSupportedMechs := cfg.HandshakeInfo.SaslSupportedMechs; saslSupportedMechs != nil {
			for _, v := range saslSupportedMechs {
				if v == SCRAMSHA256 {
					return newScramSHA256Authenticator(a.Cred, a.httpClient)
				}
			}
		}

		return newScramSHA1Authenticator(a.Cred, a.httpClient)
	}()
	if err != nil {
		return newAuthError("error creating authenticator", err)
	}

	return actual.Auth(ctx, cfg)
}

// Reauth reauthenticates the connection.
func (a *DefaultAuthenticator) Reauth(_ context.Context, _ *driver.AuthConfig) error {
	return newAuthError("DefaultAuthenticator does not support reauthentication", nil)
}
