// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package assertbsoncore

import (
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/handshake"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

// HandshakeClientMetadata compares the client metadata in two wire messages. It
// extracts the client metadata document from each wire message and compares
// them. If the document is not found, it assumes the wire message is just the
// value of the client metadata document itself.
func HandshakeClientMetadata(t testing.TB, expectedWM, actualWM []byte) bool {
	gotCommand, err := handshake.ParseClientMetadata(actualWM)
	if err != nil {
		if errors.Is(err, bsoncore.ErrElementNotFound) {
			// If the element is not found, the actual wire message may just be the
			// client metadata document itself.
			gotCommand = bsoncore.Document(actualWM)
		} else {
			return assert.Fail(t, "error parsing actual wire message: %v", err)
		}
	}

	wantCommand, err := handshake.ParseClientMetadata(expectedWM)
	if err != nil {
		// If the element is not found, the expected wire message may just be the
		// client metadata document itself.
		if errors.Is(err, bsoncore.ErrElementNotFound) {
			wantCommand = bsoncore.Document(expectedWM)
		} else {
			return assert.Fail(t, "error parsing expected wire message: %v", err)
		}
	}

	return assert.Equal(t, wantCommand, gotCommand,
		"expected: %v, got: %v", bsoncore.Document(wantCommand), bsoncore.Document(gotCommand))
}
