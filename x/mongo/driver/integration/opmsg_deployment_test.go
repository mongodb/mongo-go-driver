// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
)

func TestOPMSGMockDeployment(t *testing.T) {
	var md *MockDeployment

	t.Run("NewMockDeployment", func(t *testing.T) {
		md = NewMockDeployment()
		require.NotNil(t, md, "unexpected error from NewMockDeployment")
	})
	t.Run("AddResponses with one", func(t *testing.T) {
		res := bson.D{{"ok", 1}}
		md.AddResponses(res)
		assert.NotNil(t, md.conn.responses, "expected non-nil responses")
		assert.Len(t, md.conn.responses, 1, "expected 1 response, got %v", len(md.conn.responses))
	})
	t.Run("AddResponses with multiple", func(t *testing.T) {
		res1 := bson.D{{"ok", 1}}
		res2 := bson.D{{"ok", 2}}
		res3 := bson.D{{"ok", 3}}
		md.AddResponses(res1, res2, res3)
		assert.NotNil(t, md.conn.responses, "expected non-nil responses")
		assert.Len(t, md.conn.responses, 4, "expected 4 responses, got %v", len(md.conn.responses))
	})
	t.Run("ClearResponses", func(t *testing.T) {
		md.ClearResponses()
		assert.NotNil(t, md.conn.responses, "expected non-nil responses")
		assert.Len(t, md.conn.responses, 0, "expected 0 responses, got %v", len(md.conn.responses))
	})
}
