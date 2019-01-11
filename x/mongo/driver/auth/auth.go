// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package auth

import (
	"context"
	"fmt"

	"github.com/mongodb/mongo-go-driver/x/network/address"
	"github.com/mongodb/mongo-go-driver/x/network/command"
	"github.com/mongodb/mongo-go-driver/x/network/connection"
	"github.com/mongodb/mongo-go-driver/x/network/description"
	"github.com/mongodb/mongo-go-driver/x/network/wiremessage"
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
	AppName       string
	Authenticator Authenticator
	Compressors   []string
	DBUser        string
}

// Handshaker creates a connection handshaker for the given authenticator.
func Handshaker(h connection.Handshaker, options *HandshakeOptions) connection.Handshaker {
	return connection.HandshakerFunc(func(ctx context.Context, addr address.Address, rw wiremessage.ReadWriter) (description.Server, error) {
		desc, err := (&command.Handshake{
			Client:             command.ClientDoc(options.AppName),
			Compressors:        options.Compressors,
			SaslSupportedMechs: options.DBUser,
		}).Handshake(ctx, addr, rw)

		if err != nil {
			return description.Server{}, newAuthError("handshake failure", err)
		}

		err = options.Authenticator.Auth(ctx, desc, rw)
		if err != nil {
			return description.Server{}, newAuthError("auth error", err)
		}
		if h == nil {
			return desc, nil
		}
		return h.Handshake(ctx, addr, rw)
	})
}

// Authenticator handles authenticating a connection.
type Authenticator interface {
	// Auth authenticates the connection.
	Auth(context.Context, description.Server, wiremessage.ReadWriter) error
}

func newAuthError(msg string, inner error) error {
	return &Error{
		message: msg,
		inner:   inner,
	}
}

func newError(err error, mech string) error {
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

// Message returns the message.
func (e *Error) Message() string {
	return e.message
}
