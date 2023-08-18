// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package logger_test

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/logger"
)

func TestContext_WithOperationName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		ctx    context.Context
		opName string
		ok     bool
	}{
		{
			name:   "simple",
			ctx:    context.Background(),
			opName: "foo",
			ok:     true,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture the range variable.

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := logger.WithOperationName(tt.ctx, tt.opName)

			opName, ok := logger.OperationName(ctx)
			assert.Equal(t, tt.ok, ok)

			if ok {
				assert.Equal(t, tt.opName, opName)
			}
		})
	}
}

func TestContext_OperationName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		ctx    context.Context
		opName interface{}
		ok     bool
	}{
		{
			name:   "nil",
			ctx:    context.Background(),
			opName: nil,
			ok:     false,
		},
		{
			name:   "string type",
			ctx:    context.Background(),
			opName: "foo",
			ok:     true,
		},
		{
			name:   "non-string type",
			ctx:    context.Background(),
			opName: int32(1),
			ok:     false,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture the range variable.

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			if opNameStr, ok := tt.opName.(string); ok {
				ctx = logger.WithOperationName(tt.ctx, opNameStr)
			}

			opName, ok := logger.OperationName(ctx)
			assert.Equal(t, tt.ok, ok)

			if ok {
				assert.Equal(t, tt.opName, opName)
			}
		})
	}
}

func TestContext_WithOperationID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  context.Context
		opID int32
		ok   bool
	}{
		{
			name: "non-zero",
			ctx:  context.Background(),
			opID: 1,
			ok:   true,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture the range variable.

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := logger.WithOperationID(tt.ctx, tt.opID)

			opID, ok := logger.OperationID(ctx)
			assert.Equal(t, tt.ok, ok)

			if ok {
				assert.Equal(t, tt.opID, opID)
			}
		})
	}
}

func TestContext_OperationID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  context.Context
		opID interface{}
		ok   bool
	}{
		{
			name: "nil",
			ctx:  context.Background(),
			opID: nil,
			ok:   false,
		},
		{
			name: "i32 type",
			ctx:  context.Background(),
			opID: int32(1),
			ok:   true,
		},
		{
			name: "non-i32 type",
			ctx:  context.Background(),
			opID: "foo",
			ok:   false,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture the range variable.

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			if opIDI32, ok := tt.opID.(int32); ok {
				ctx = logger.WithOperationID(tt.ctx, opIDI32)
			}

			opName, ok := logger.OperationID(ctx)
			assert.Equal(t, tt.ok, ok)

			if ok {
				assert.Equal(t, tt.opID, opName)
			}
		})
	}
}
