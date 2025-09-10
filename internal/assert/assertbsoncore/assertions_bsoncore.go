// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package assertbsoncore

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

func HandshakeClientMetadata(t testing.TB, expectedWM, actualWM []byte) bool {
	command := bsoncore.Document(actualWM)

	// Lookup the "client" field in the command document.
	clientVal, err := command.LookupErr("client")
	if err != nil {
		return assert.Fail(t, "expected command to contain the 'client' field")
	}

	GotCommand, ok := clientVal.DocumentOK()
	if !ok {
		return assert.Fail(t, "expected client field to be a document, got %s", clientVal.Type)
	}

	wantCommand := bsoncore.Document(expectedWM)
	return assert.Equal(t, wantCommand, GotCommand, "want: %v, got: %v", wantCommand, GotCommand)
}
