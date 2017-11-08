// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package feature_test

import (
	"testing"

	. "github.com/10gen/mongo-go-driver/mongo/internal/feature"
	"github.com/10gen/mongo-go-driver/mongo/model"
	"github.com/stretchr/testify/require"
)

func TestMaxStaleness(t *testing.T) {
	tests := []struct {
		version  model.Version
		expected bool
	}{
		{model.Version{Parts: []uint8{2, 4, 0}}, false},
		{model.Version{Parts: []uint8{3, 3, 99}}, false},
		{model.Version{Parts: []uint8{3, 4, 0}}, true},
		{model.Version{Parts: []uint8{3, 4, 1}}, true},
	}

	for _, test := range tests {
		t.Run(test.version.String(), func(t *testing.T) {
			t.Parallel()

			err := MaxStaleness(test.version)
			if test.expected {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
