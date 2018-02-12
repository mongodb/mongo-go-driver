// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//+build gssapi
//+build windows linux darwin

package auth

import (
	"context"
	"fmt"

	"github.com/mongodb/mongo-go-driver/mongo/internal/auth/gssapi"
	"github.com/mongodb/mongo-go-driver/mongo/private/conn"
)

// GSSAPI is the mechanism name for GSSAPI.
const GSSAPI = "GSSAPI"

func newGSSAPIAuthenticator(cred *Cred) (Authenticator, error) {
	if cred.Source != "" && cred.Source != "$external" {
		return nil, fmt.Errorf("GSSAPI source must be empty or $external")
	}

	return &GSSAPIAuthenticator{
		Username:    cred.Username,
		Password:    cred.Password,
		PasswordSet: cred.PasswordSet,
		Props:       cred.Props,
	}, nil
}

// GSSAPIAuthenticator uses the GSSAPI algorithm over SASL to authenticate a connection.
type GSSAPIAuthenticator struct {
	Username    string
	Password    string
	PasswordSet bool
	Props       map[string]string
}

// Auth authenticates the connection.
func (a *GSSAPIAuthenticator) Auth(ctx context.Context, c conn.Connection) error {
	client, err := gssapi.New(c.Model().Addr.String(), a.Username, a.Password, a.PasswordSet, a.Props)

	if err != nil {
		return err
	}
	return ConductSaslConversation(ctx, c, "$external", client)
}
