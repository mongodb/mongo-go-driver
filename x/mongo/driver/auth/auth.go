// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
)

// AuthenticatorFactory constructs an authenticator.
type AuthenticatorFactory func(cred *Cred) (Authenticator, error)

var authFactories = make(map[string]AuthenticatorFactory)

func init() {
	RegisterAuthenticatorFactory("", newDefaultAuthenticator)
	RegisterAuthenticatorFactory(SCRAMSHA1, newScramSHA1Authenticator)
	RegisterAuthenticatorFactory(SCRAMSHA256, newScramSHA256Authenticator)
	RegisterAuthenticatorFactory(MONGODBCR, newMongoDBCRAuthenticator)
	RegisterAuthenticatorFactory(PLAIN, newPlainAuthenticator)
	RegisterAuthenticatorFactory(GSSAPI, newGSSAPIAuthenticator)
	RegisterAuthenticatorFactory(MongoDBX509, newMongoDBX509Authenticator)
	RegisterAuthenticatorFactory(MongoDBAWS, newMongoDBAWSAuthenticator)
	RegisterAuthenticatorFactory(MongoDBOIDC, newOidcAuthenticator)
}

// CreateAuthenticator creates an authenticator.
func CreateAuthenticator(name string, cred *Cred) (Authenticator, error) {
	if f, ok := authFactories[name]; ok {
		return f(cred)
	}

	return nil, newAuthError(fmt.Sprintf("unknown authenticator: %s", name), nil)
}

// RegisterAuthenticatorFactory registers the authenticator factory.
func RegisterAuthenticatorFactory(name string, factory AuthenticatorFactory) {
	authFactories[name] = factory
}

// HandshakeOptions packages options that can be passed to the Handshaker()
// function.  DBUser is optional but must be of the form <dbname.username>;
// if non-empty, then the connection will do SASL mechanism negotiation.
type HandshakeOptions struct {
	AppName               string
	Authenticator         Authenticator
	Compressors           []string
	DBUser                string
	PerformAuthentication func(description.Server) bool
	ClusterClock          *session.ClusterClock
	ServerAPI             *driver.ServerAPIOptions
	LoadBalanced          bool
	HTTPClient            *http.Client
}

type authHandshaker struct {
	wrapped driver.Handshaker
	options *HandshakeOptions

	handshakeInfo driver.HandshakeInformation
	conversation  SpeculativeConversation
}

var _ driver.Handshaker = (*authHandshaker)(nil)

// GetHandshakeInformation performs the initial MongoDB handshake to retrieve the required information for the provided
// connection.
func (ah *authHandshaker) GetHandshakeInformation(ctx context.Context, addr address.Address, conn driver.Connection) (driver.HandshakeInformation, error) {
	if ah.wrapped != nil {
		return ah.wrapped.GetHandshakeInformation(ctx, addr, conn)
	}

	op := operation.NewHello().
		AppName(ah.options.AppName).
		Compressors(ah.options.Compressors).
		SASLSupportedMechs(ah.options.DBUser).
		ClusterClock(ah.options.ClusterClock).
		ServerAPI(ah.options.ServerAPI).
		LoadBalanced(ah.options.LoadBalanced)

	if ah.options.Authenticator != nil {
		if speculativeAuth, ok := ah.options.Authenticator.(SpeculativeAuthenticator); ok {
			ah.conversation = speculativeAuth.CreateSpeculativeConversation()

			firstMsg, err := ah.conversation.FirstMessage(addr)
			if err != nil {
				return driver.HandshakeInformation{}, newAuthError("failed to create speculative authentication message", err)
			}

			op = op.SpeculativeAuthenticate(firstMsg)
		}
	}

	var err error
	ah.handshakeInfo, err = op.GetHandshakeInformation(ctx, addr, conn)
	if err != nil {
		return driver.HandshakeInformation{}, newAuthError("handshake failure", err)
	}
	return ah.handshakeInfo, nil
}

// FinishHandshake performs authentication for conn if necessary.
func (ah *authHandshaker) FinishHandshake(ctx context.Context, conn driver.Connection) error {
	performAuth := ah.options.PerformAuthentication
	if performAuth == nil {
		performAuth = func(serv description.Server) bool {
			// Authentication is possible against all server types except arbiters
			return serv.Kind != description.RSArbiter
		}
	}

	if performAuth(conn.Description()) && ah.options.Authenticator != nil {
		if err := ah.authenticate(ctx, ah.newConfig(conn)); err != nil {
			return newAuthError("auth error", err)
		}
	}

	if ah.wrapped == nil {
		return nil
	}
	return ah.wrapped.FinishHandshake(ctx, conn)
}

func (ah *authHandshaker) newConfig(conn driver.Connection) *Config {
	return &Config{
		Description:   conn.Description(),
		Connection:    conn,
		ClusterClock:  ah.options.ClusterClock,
		HandshakeInfo: ah.handshakeInfo,
		ServerAPI:     ah.options.ServerAPI,
		HTTPClient:    ah.options.HTTPClient,
	}
}

func (ah *authHandshaker) authenticate(ctx context.Context, cfg *Config) error {
	speculativeResponse := ah.handshakeInfo.SpeculativeAuthenticate
	// Conduct the remainder of the conversation for speculative authentication attempt.
	if ah.conversation != nil {
		err := ah.conversation.Finish(ctx, cfg, speculativeResponse)
		// Finish the conversation if the initial hello reply included a response.
		if speculativeResponse != nil {
			return err
		}
	} else if speculativeResponse != nil {
		// Defensively ensure that the server did not include a response if speculative auth was not attempted.
		return errors.New("speculative auth was not attempted but the server included a response")
	}

	// If the server does not support speculative authentication or the first attempt was not successful, we need to
	// perform authentication from scratch.
	return ah.options.Authenticator.Auth(ctx, cfg)
}

// Reauth re-authenticates the connection.
func (ah *authHandshaker) Reauth(ctx context.Context, conn driver.Connection) error {
	return ah.options.Authenticator.Auth(ctx, ah.newConfig(conn))
}

// Handshaker creates a connection handshaker for the given authenticator.
func Handshaker(h driver.Handshaker, options *HandshakeOptions) driver.Handshaker {
	return &authHandshaker{
		wrapped: h,
		options: options,
	}
}

// Config holds the information necessary to perform an authentication attempt.
type Config struct {
	Description   description.Server
	Connection    driver.Connection
	ClusterClock  *session.ClusterClock
	HandshakeInfo driver.HandshakeInformation
	ServerAPI     *driver.ServerAPIOptions
	HTTPClient    *http.Client
}

// Authenticator handles authenticating a connection.
type Authenticator interface {
	// Auth authenticates the connection.
	Auth(context.Context, *Config) error
}

// Reauthenticator re-authenticates a connection.
type Reauthenticator interface {
	Reauth(context.Context, driver.Connection) error
}

func newAuthError(msg string, inner error) *Error {
	return &Error{
		message: msg,
		inner:   inner,
	}
}

func newError(err error, mech string) *Error {
	return &Error{
		message: fmt.Sprintf("unable to authenticate using mechanism \"%s\"", mech),
		inner:   err,
	}
}

// Error is an error that occurred during authentication.
type Error struct {
	message string
	inner   error
}

func (e *Error) Error() string {
	if e.inner == nil {
		return e.message
	}
	return fmt.Sprintf("%s: %s", e.message, e.inner)
}

// Inner returns the wrapped error.
func (e *Error) Inner() error {
	return e.inner
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.inner
}

// Message returns the message.
func (e *Error) Message() string {
	return e.message
}
