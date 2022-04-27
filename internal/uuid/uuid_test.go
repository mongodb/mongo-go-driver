// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package uuid

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	m := make(map[UUID]bool)
	for i := 1; i < 1000000; i++ {
		uuid, err := New()
		require.NoError(t, err, "New error")
		require.False(t, m[uuid], "New returned a duplicate UUID %v", uuid)
		m[uuid] = true
	}
}

// GODRIVER-2349
// Test that initializing many package-global UUID sources concurrently never leads to any duplicate
// UUIDs being generated.
func TestGlobalSource(t *testing.T) {
	// Create a slice of 1,000 sources and initialize them in 1,000 separate goroutines. The goal is
	// to emulate many separate Go driver processes starting at the same time and initializing the
	// uuid package at the same time.
	sources := make([]*source, 1000)
	var wg sync.WaitGroup
	for i := range sources {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sources[i] = newGlobalSource()
		}(i)
	}
	wg.Wait()

	// Read 1,000 UUIDs from each source and assert that there is never a duplicate value, either
	// from the same source or from separate sources.
	const iterations = 1000
	uuids := make(map[UUID]bool, len(sources)*iterations)
	for i := 0; i < iterations; i++ {
		for j, s := range sources {
			uuid, err := s.new()
			require.NoError(t, err, "new() error")
			require.Falsef(t, uuids[uuid], "source %d returned a duplicate UUID on iteration %d: %v", j, i, uuid)
			uuids[uuid] = true
		}
	}
}
