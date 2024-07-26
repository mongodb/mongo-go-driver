// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package writeconcern_test

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

func TestWriteConcern(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	testCases := []struct {
		name             string
		wc               *writeconcern.WriteConcern
		wantAcknowledged bool
		wantIsValid      bool
	}{
		{
			name:             "Unacknowledged",
			wc:               writeconcern.Unacknowledged(),
			wantAcknowledged: false,
			wantIsValid:      true,
		},
		{
			name:             "W1",
			wc:               writeconcern.W1(),
			wantAcknowledged: true,
			wantIsValid:      true,
		},
		{
			name:             "Journaled",
			wc:               writeconcern.Journaled(),
			wantAcknowledged: true,
			wantIsValid:      true,
		},
		{
			name:             "Majority",
			wc:               writeconcern.Majority(),
			wantAcknowledged: true,
			wantIsValid:      true,
		},
		{
			name: "{w: 0, j: true}",
			wc: &writeconcern.WriteConcern{
				W:       0,
				Journal: boolPtr(true),
			},
			wantAcknowledged: true,
			wantIsValid:      false,
		},
		{
			name:             "{w: custom}",
			wc:               &writeconcern.WriteConcern{W: "custom"},
			wantAcknowledged: true,
			wantIsValid:      true,
		},
		{
			name:             "nil",
			wc:               nil,
			wantAcknowledged: true,
			wantIsValid:      true,
		},
		{
			name: "invalid type",
			wc: &writeconcern.WriteConcern{
				W: struct{ Field string }{},
			},
			wantAcknowledged: true,
			wantIsValid:      false,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t,
				tc.wantAcknowledged,
				tc.wc.Acknowledged(),
				"expected and actual Acknowledged value are different")
			assert.Equal(t,
				tc.wantIsValid,
				tc.wc.IsValid(),
				"expected and actual IsValid value are different")
		})
	}
}
