// Copyright (C) MongoDB, Inc. 2020-present.

// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package readpref

import (
	"testing"

	"go.mongodb.org/mongo-driver/internal/testutil/assert"
)

func TestMode_String(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		mode Mode
	}{
		{"primary", PrimaryMode},
		{"primaryPreferred", PrimaryPreferredMode},
		{"secondary", SecondaryMode},
		{"secondaryPreferred", SecondaryPreferredMode},
		{"nearest", NearestMode},
		{"unknown", Mode(42)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.name, tc.mode.String(), "expected %q, got %q", tc.name, tc.mode.String())
		})
	}
}
