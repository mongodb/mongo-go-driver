// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package randutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCryptoSeed(t *testing.T) {
	seeds := make(map[uint64]bool)
	for i := 1; i < 1000000; i++ {
		s := cryptoSeed()
		require.False(t, seeds[s], "cryptoSeed returned a duplicate value %d", s)
		seeds[s] = true
	}
}
