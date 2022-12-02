// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package uuid

import (
	"sync"
	"testing"

	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/internal/testutil/israce"
)

// GODRIVER-2349
// Test that initializing many package-global UUID sources concurrently never leads to any duplicate
// UUIDs being generated.
func TestGlobalSource(t *testing.T) {
	t.Parallel()

	t.Run("exp rand 1 UUID x 1,000,000 goroutines using a global source", func(t *testing.T) {
		t.Parallel()

		if israce.Enabled {
			t.Skip("skipping as race detector is enabled and test exceeds 8128 goroutine limit")
		}

		// Read a UUID from each of 1,000,000 goroutines and assert that there is never a duplicate value.
		const iterations = 1e6
		uuids := new(sync.Map)
		var wg sync.WaitGroup
		wg.Add(iterations)
		for i := 0; i < iterations; i++ {
			go func(i int) {
				defer wg.Done()
				uuid, err := New()
				require.NoError(t, err, "new() error")
				_, ok := uuids.Load(uuid)
				require.Falsef(t, ok, "New returned a duplicate UUID on iteration %d: %v", i, uuid)
				uuids.Store(uuid, true)
			}(i)
		}
		wg.Wait()
	})
	t.Run("exp rand 1 UUID x 1,000,000 goroutines each initializing a new source", func(t *testing.T) {
		t.Parallel()

		if israce.Enabled {
			t.Skip("skipping as race detector is enabled and test exceeds 8128 goroutine limit")
		}

		// Read a UUID from each of 1,000,000 goroutines and assert that there is never a duplicate value.
		// The goal is to emulate many separate Go driver processes starting at the same time and
		// initializing the uuid package at the same time.
		const iterations = 1e6
		uuids := new(sync.Map)
		var wg sync.WaitGroup
		wg.Add(iterations)
		for i := 0; i < iterations; i++ {
			go func(i int) {
				defer wg.Done()
				s := newSource()
				uuid, err := s.new()
				require.NoError(t, err, "new() error")
				_, ok := uuids.Load(uuid)
				require.Falsef(t, ok, "New returned a duplicate UUID on iteration %d: %v", i, uuid)
				uuids.Store(uuid, true)
			}(i)
		}
		wg.Wait()
	})
	t.Run("exp rand 1,000 UUIDs x 1,000 goroutines each initializing a new source", func(t *testing.T) {
		t.Parallel()

		// Read 1,000 UUIDs from each goroutine and assert that there is never a duplicate value, either
		// from the same goroutine or from separate goroutines.
		const iterations = 1000
		uuids := new(sync.Map)
		var wg sync.WaitGroup
		wg.Add(iterations)
		for i := 0; i < iterations; i++ {
			go func(i int) {
				defer wg.Done()
				s := newSource()
				for j := 0; j < iterations; j++ {
					uuid, err := s.new()
					require.NoError(t, err, "new() error")
					_, ok := uuids.Load(uuid)
					require.Falsef(t, ok, "goroutine %d returned a duplicate UUID on iteration %d: %v", i, j, uuid)
					uuids.Store(uuid, true)
				}
			}(i)
		}
		wg.Wait()
	})
}

func BenchmarkUuidGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := New()
		if err != nil {
			panic(err)
		}
	}
}
