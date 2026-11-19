// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/integtest"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestAuthProse(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))

	mt.Run("1. Driver rejects a PLAIN authentication credential with NUL character", func(mt *mtest.T) {
		testCases := []struct {
			name     string
			username string
			password string
		}{
			{name: "Case 1: NUL character in username", username: "user\x00name", password: "password"},
			{name: "Case 2: NUL character in password", username: "username", password: "pass\x00word"},
		}

		for _, tc := range testCases {
			mt.Run(tc.name, func(mt *mtest.T) {
				clientOpts := options.Client().
					ApplyURI(mtest.ClusterURI()).
					SetAuth(options.Credential{
						AuthMechanism: "PLAIN",
						Username:      tc.username,
						Password:      tc.password,
					})
				integtest.AddTestServerAPIVersion(clientOpts)

				// The spec allows the error to be raised when the credential is
				// constructed, when the client is constructed, or when the client
				// is first used (as long as no authentication command is sent).
				// The Go Driver validates credentials in ClientOptions.Validate,
				// which Connect calls before creating any connections, so assert
				// that Connect returns the error.
				client, err := mongo.Connect(clientOpts)
				if err == nil {
					_ = client.Disconnect(context.Background())
				}
				require.ErrorContains(mt, err, "cannot contain NUL characters")
			})
		}
	})
}
