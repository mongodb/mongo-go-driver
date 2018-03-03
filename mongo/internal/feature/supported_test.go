// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package feature_test

import (
	"testing"

	. "github.com/mongodb/mongo-go-driver/mongo/internal/feature"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
	"github.com/stretchr/testify/require"
)

func TestMaxStaleness(t *testing.T) {
	wireVersionSupported := description.NewVersionRange(0, 5)
	wireVersionUnsupported := description.NewVersionRange(0, 4)

	tests := []struct {
		version  description.Version
		wire     *description.VersionRange
		expected bool
	}{
		{description.Version{Parts: []uint8{2, 4, 0}}, nil, false},
		{description.Version{Parts: []uint8{3, 3, 99}}, nil, false},
		{description.Version{Parts: []uint8{3, 4, 0}}, nil, true},
		{description.Version{Parts: []uint8{3, 4, 1}}, nil, true},
		{description.Version{Parts: []uint8{2, 4, 0}}, &wireVersionSupported, false},
		{description.Version{Parts: []uint8{2, 4, 0}}, &wireVersionUnsupported, false},
		{description.Version{Parts: []uint8{3, 4, 1}}, &wireVersionSupported, true},
		{description.Version{Parts: []uint8{3, 4, 1}}, &wireVersionUnsupported, false},
	}

	for _, test := range tests {
		t.Run(test.version.String(), func(t *testing.T) {
			t.Parallel()

			err := MaxStaleness(test.version, test.wire)
			if test.expected {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
