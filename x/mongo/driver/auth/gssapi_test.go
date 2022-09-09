// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build gssapi
// +build gssapi

package auth

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/description"
)

func TestGSSAPIAuthenticator(t *testing.T) {
	t.Run("PropsError", func(t *testing.T) {
		// Cannot specify both CANONICALIZE_HOST_NAME and SERVICE_HOST

		authenticator := &GSSAPIAuthenticator{
			Username:    "foo",
			Password:    "bar",
			PasswordSet: true,
			Props: map[string]string{
				"CANONICALIZE_HOST_NAME": "true",
				"SERVICE_HOST":           "localhost",
			},
		}
		desc := description.Server{
			WireVersion: &description.VersionRange{
				Max: 6,
			},
			Addr: address.Address("foo:27017"),
		}
		err := authenticator.Auth(context.Background(), &Config{Description: desc})
		if err == nil {
			t.Fatalf("expected err, got nil")
		}
	})

}
