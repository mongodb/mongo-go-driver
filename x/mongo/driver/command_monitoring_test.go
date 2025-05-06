// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/handshake"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

func TestCommandMonitoring(t *testing.T) {
	t.Run("redactCommand", func(t *testing.T) {
		emptyDoc := bsoncore.BuildDocumentFromElements(nil)
		legacyHello := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendInt32Element(nil, handshake.LegacyHello, 1),
		)
		legacyHelloLowercase := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendInt32Element(nil, handshake.LegacyHelloLowercase, 1),
		)
		legacyHelloSpeculative := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendInt32Element(nil, handshake.LegacyHello, 1),
			bsoncore.AppendDocumentElement(nil, "speculativeAuthenticate", emptyDoc),
		)
		legacyHelloSpeculativeLowercase := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendInt32Element(nil, handshake.LegacyHelloLowercase, 1),
			bsoncore.AppendDocumentElement(nil, "speculativeAuthenticate", emptyDoc),
		)

		testCases := []struct {
			name        string
			commandName string
			command     bsoncore.Document
			redacted    bool
		}{
			{"legacy hello", handshake.LegacyHello, legacyHello, false},
			{"legacy hello lowercase", handshake.LegacyHelloLowercase, legacyHelloLowercase, false},
			{"legacy hello speculative auth", handshake.LegacyHello, legacyHelloSpeculative, true},
			{"legacy hello speculative auth lowercase", handshake.LegacyHello, legacyHelloSpeculativeLowercase, true},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				canMonitor := (&Operation{}).redactCommand(tc.commandName, tc.command)
				assert.Equal(t, tc.redacted, canMonitor, "expected redacted %v, got %v", tc.redacted, canMonitor)
			})
		}
	})
}
