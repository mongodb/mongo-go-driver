// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driver

import (
	"testing"

	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func TestCommandMonitoring(t *testing.T) {
	t.Run("redactCommand", func(t *testing.T) {
		emptyDoc := bsoncore.BuildDocumentFromElements(nil)
		isMaster := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendInt32Element(nil, "isMaster", 1),
		)
		isMasterLowercase := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendInt32Element(nil, "ismaster", 1),
		)
		isMasterSpeculative := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendInt32Element(nil, "isMaster", 1),
			bsoncore.AppendDocumentElement(nil, "speculativeAuthenticate", emptyDoc),
		)
		isMasterSpeculativeLowercase := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendInt32Element(nil, "ismaster", 1),
			bsoncore.AppendDocumentElement(nil, "speculativeAuthenticate", emptyDoc),
		)

		testCases := []struct {
			name        string
			commandName string
			command     bsoncore.Document
			redacted    bool
		}{
			{"isMaster", "isMaster", isMaster, false},
			{"isMaster lowercase", "ismaster", isMasterLowercase, false},
			{"isMaster speculative auth", "isMaster", isMasterSpeculative, true},
			{"isMaster speculative auth lowercase", "isMaster", isMasterSpeculativeLowercase, true},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				canMonitor := (&Operation{}).redactCommand(tc.commandName, tc.command)
				assert.Equal(t, tc.redacted, canMonitor, "expected redacted %v, got %v", tc.redacted, canMonitor)
			})
		}
	})
}
