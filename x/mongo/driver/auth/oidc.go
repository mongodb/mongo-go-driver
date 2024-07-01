// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
)

// MongoDBOIDC is the string constant for the MONGODB-OIDC authentication mechanism.
const MongoDBOIDC = "MONGODB-OIDC"

// TODO GODRIVER-2728: Automatic token acquisition for Azure Identity Provider
// const tokenResourceProp = "TOKEN_RESOURCE"
const environmentProp = "ENVIRONMENT"

const resourceProp = "TOKEN_RESOURCE"

// GODRIVER-3249	OIDC: Handle all possible OIDC configuration errors
//const allowedHostsProp = "ALLOWED_HOSTS"

const azureEnvironmentValue = "azure"
const gcpEnvironmentValue = "gcp"
const testEnvironmentValue = "test"

const apiVersion = 1
const invalidateSleepTimeout = 100 * time.Millisecond
const machineCallbackTimeout = 60 * time.Second

//GODRIVER-3246	OIDC: Implement Human Callback Mechanism
//var defaultAllowedHosts = []string{
//	"*.mongodb.net",
//	"*.mongodb-qa.net",
//	"*.mongodb-dev.net",
//	"*.mongodbgov.net",
//	"localhost",
//	"127.0.0.1",
//	"::1",
//}

// OIDCCallback is a function that takes a context and OIDCArgs and returns an OIDCCredential.
type OIDCCallback = driver.OIDCCallback

// OIDCArgs contains the arguments for the OIDC callback.
type OIDCArgs = driver.OIDCArgs

// OIDCCredential contains the access token and refresh token.
type OIDCCredential = driver.OIDCCredential

// IDPInfo contains the information needed to perform OIDC authentication with an Identity Provider.
type IDPInfo = driver.IDPInfo

var _ driver.Authenticator = (*OIDCAuthenticator)(nil)
var _ SpeculativeAuthenticator = (*OIDCAuthenticator)(nil)
var _ SaslClient = (*oidcOneStep)(nil)

// OIDCAuthenticator is synchronized and handles caching of the access token, refreshToken,
// and IDPInfo. It also provides a mechanism to refresh the access token, but this functionality
// is only for the OIDC Human flow.
type OIDCAuthenticator struct {
	mu sync.Mutex // Guards all of the info in the OIDCAuthenticator struct.

	AuthMechanismProperties map[string]string
	OIDCMachineCallback     OIDCCallback
	OIDCHumanCallback       OIDCCallback

	userName     string
	httpClient   *http.Client
	accessToken  string
	refreshToken *string
	idpInfo      *IDPInfo
	tokenGenID   uint64
}

// SetAccessToken allows for manually setting the access token for the OIDCAuthenticator, this is
// only for testing purposes.
func (oa *OIDCAuthenticator) SetAccessToken(accessToken string) {
	oa.mu.Lock()
	defer oa.mu.Unlock()
	oa.accessToken = accessToken
}

func newOIDCAuthenticator(cred *Cred, httpClient *http.Client) (Authenticator, error) {
	if cred.Password != "" {
		return nil, fmt.Errorf("password cannot be specified for %q", MongoDBOIDC)
	}
	if cred.Props != nil {
		if env, ok := cred.Props[environmentProp]; ok {
			switch strings.ToLower(env) {
			case azureEnvironmentValue:
				fallthrough
			case gcpEnvironmentValue:
				if _, ok := cred.Props[resourceProp]; !ok {
					return nil, fmt.Errorf("%q must be specified for %q %q", resourceProp, env, environmentProp)
				}
				fallthrough
			case testEnvironmentValue:
				if cred.OIDCMachineCallback != nil || cred.OIDCHumanCallback != nil {
					return nil, fmt.Errorf("OIDC callbacks are not allowed for %q %q", env, environmentProp)
				}
			}
		}
	}
	oa := &OIDCAuthenticator{
		userName:                cred.Username,
		httpClient:              httpClient,
		AuthMechanismProperties: cred.Props,
		OIDCMachineCallback:     cred.OIDCMachineCallback,
		OIDCHumanCallback:       cred.OIDCHumanCallback,
	}
	return oa, nil
}

type oidcOneStep struct {
	userName    string
	accessToken string
}

func jwtStepRequest(accessToken string) []byte {
	return bsoncore.NewDocumentBuilder().
		AppendString("jwt", accessToken).
		Build()
}

// TODO GODRIVER-3246: Implement OIDC human flow
//func principalStepRequest(principal string) []byte {
//	doc := bsoncore.NewDocumentBuilder()
//	if principal != "" {
//		doc.AppendString("n", principal)
//	}
//	return doc.Build()
//}

func (oos *oidcOneStep) Start() (string, []byte, error) {
	return MongoDBOIDC, jwtStepRequest(oos.accessToken), nil
}

func (oos *oidcOneStep) Next([]byte) ([]byte, error) {
	return nil, newAuthError("unexpected step in OIDC authentication", nil)
}

func (*oidcOneStep) Completed() bool {
	return true
}

func (oa *OIDCAuthenticator) providerCallback() (OIDCCallback, error) {
	env, ok := oa.AuthMechanismProperties[environmentProp]
	if !ok {
		return nil, nil
	}

	switch env {
	// TODO GODRIVER-2728: Automatic token acquisition for Azure Identity Provider
	// TODO GODRIVER-2806: Automatic token acquisition for GCP Identity Provider
	// This is here just to pass the linter, it will be fixed in one of the above tickets.
	case azureEnvironmentValue, gcpEnvironmentValue:
		return func(ctx context.Context, args *OIDCArgs) (*OIDCCredential, error) {
			return nil, fmt.Errorf("automatic token acquisition for %q not implemented yet", env)
		}, fmt.Errorf("automatic token acquisition for %q not implemented yet", env)
	}

	return nil, fmt.Errorf("%q %q not supported for MONGODB-OIDC", environmentProp, env)
}

func (oa *OIDCAuthenticator) getAccessToken(
	ctx context.Context,
	conn driver.Connection,
	args *OIDCArgs,
	callback OIDCCallback,
) (string, error) {
	oa.mu.Lock()
	defer oa.mu.Unlock()

	if oa.accessToken != "" {
		return oa.accessToken, nil
	}

	cred, err := callback(ctx, args)
	if err != nil {
		return "", err
	}

	oa.accessToken = cred.AccessToken
	oa.tokenGenID++
	conn.SetOIDCTokenGenID(oa.tokenGenID)
	if cred.RefreshToken != nil {
		oa.refreshToken = cred.RefreshToken
	}
	return cred.AccessToken, nil
}

// TODO GODRIVER-3246: Implement OIDC human flow
// This should only be called with the Mutex held.
//func (oa *OIDCAuthenticator) getAccessTokenWithRefresh(
//	ctx context.Context,
//	callback OIDCCallback,
//	refreshToken string,
//) (string, error) {
//
//	cred, err := callback(ctx, &OIDCArgs{
//		Version:      apiVersion,
//		IDPInfo:      oa.idpInfo,
//		RefreshToken: &refreshToken,
//	})
//	if err != nil {
//		return "", err
//	}
//
//	oa.accessToken = cred.AccessToken
//	oa.tokenGenID++
//	oa.cfg.Connection.SetOIDCTokenGenID(oa.tokenGenID)
//	return cred.AccessToken, nil
//}

// invalidateAccessToken invalidates the access token, if the force flag is set to true (which is
// only on a Reauth call) or if the tokenGenID of the connection is greater than or equal to the
// tokenGenID of the OIDCAuthenticator. It should never actually be greater than, but only equal,
// but this is a safety check, since extra invalidation is only a performance impact, not a
// correctness impact.
func (oa *OIDCAuthenticator) invalidateAccessToken(conn driver.Connection) {
	oa.mu.Lock()
	defer oa.mu.Unlock()
	tokenGenID := conn.OIDCTokenGenID()
	// If the connection used in a Reauth is a new connection it will not have a correct tokenGenID,
	// it will instead be set to 0. In the absence of information, the only safe thing to do is to
	// invalidate the cached accessToken.
	if tokenGenID == 0 || tokenGenID >= oa.tokenGenID {
		oa.accessToken = ""
		conn.SetOIDCTokenGenID(0)
	}
}

// Reauth reauthenticates the connection when the server returns a 391 code. Reauth is part of the
// driver.Authenticator interface.
func (oa *OIDCAuthenticator) Reauth(ctx context.Context, cfg *Config) error {
	oa.invalidateAccessToken(cfg.Connection)
	return oa.Auth(ctx, cfg)
}

// Auth authenticates the connection.
func (oa *OIDCAuthenticator) Auth(ctx context.Context, cfg *Config) error {
	var err error

	if cfg == nil {
		return newAuthError(fmt.Sprintf("config must be set for %q authentication", MongoDBOIDC), nil)
	}
	conn := cfg.Connection

	if oa.accessToken != "" {
		err = ConductSaslConversation(ctx, cfg, "$external", &oidcOneStep{
			userName:    oa.userName,
			accessToken: oa.accessToken,
		})
		if err == nil {
			return nil
		}
		oa.invalidateAccessToken(conn)
		time.Sleep(invalidateSleepTimeout)
	}

	if oa.OIDCHumanCallback != nil {
		return oa.doAuthHuman(ctx, cfg, oa.OIDCHumanCallback)
	}

	// Handle user provided or automatic provider machine callback.
	var machineCallback OIDCCallback
	if oa.OIDCMachineCallback != nil {
		machineCallback = oa.OIDCMachineCallback
	} else {
		machineCallback, err = oa.providerCallback()
		if err != nil {
			return fmt.Errorf("error getting built-in OIDC provider: %w", err)
		}
	}

	if machineCallback != nil {
		return oa.doAuthMachine(ctx, cfg, machineCallback)
	}
	return newAuthError("no OIDC callback provided", nil)
}

func (oa *OIDCAuthenticator) doAuthHuman(_ context.Context, _ *Config, _ OIDCCallback) error {
	// TODO GODRIVER-3246: Implement OIDC human flow
	// Println is for linter
	fmt.Println("OIDC human flow not implemented yet", oa.idpInfo)
	return newAuthError("OIDC human flow not implemented yet", nil)
}

func (oa *OIDCAuthenticator) doAuthMachine(ctx context.Context, cfg *Config, machineCallback OIDCCallback) error {
	subCtx, cancel := context.WithTimeout(ctx, machineCallbackTimeout)
	accessToken, err := oa.getAccessToken(subCtx,
		cfg.Connection,
		&OIDCArgs{
			Version: apiVersion,
			// idpInfo is nil for machine callbacks in the current spec.
			IDPInfo:      nil,
			RefreshToken: nil,
		},
		machineCallback)
	cancel()
	if err != nil {
		return err
	}
	return runSaslConversation(ctx,
		cfg,
		newSaslConversation(&oidcOneStep{accessToken: accessToken}, "$external", false),
	)
}

// CreateSpeculativeConversation creates a speculative conversation for SCRAM authentication.
func (oa *OIDCAuthenticator) CreateSpeculativeConversation() (SpeculativeConversation, error) {
	oa.mu.Lock()
	defer oa.mu.Unlock()
	accessToken := oa.accessToken
	if accessToken == "" {
		return nil, nil // Skip speculative auth.
	}

	return newSaslConversation(&oidcOneStep{accessToken: accessToken}, "$external", true), nil
}
