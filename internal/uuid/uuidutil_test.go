// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package uuid

import (
	"sync"
	"testing"

	"go.mongodb.org/mongo-driver/internal/randutil"
	"go.mongodb.org/mongo-driver/internal/uuid/uuid"

	"github.com/stretchr/testify/require"
)

// GODRIVER-2349
// Test that initializing many package-global UUID sources concurrently never leads to any duplicate
// UUIDs being generated.
func TestGlobalSource(t *testing.T) {
	t.Run("exp rand 1 UUID x 1,000,000 goroutines", func(t *testing.T) {
		// Read a UUID from each of 1,000,000 goroutines and assert that there is never a duplicate value.
		uuid.SetRand(randutil.GlobalRand)
		uuids := new(sync.Map)
		var wg sync.WaitGroup
		for i := 1; i < 1000000; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				uuid, err := uuid.NewRandom()
				require.NoError(t, err, "new() error")
				_, ok := uuids.Load(uuid)
				require.Falsef(t, ok, "New returned a duplicate UUID on iteration %d: %v", i, uuid)
				uuids.Store(uuid, true)
			}(i)
		}
		wg.Wait()
	})
	t.Run("exp rand 1,000 UUIDs x 1,000 goroutines", func(t *testing.T) {
		// Read 1,000 UUIDs from each goroutine and assert that there is never a duplicate value, either
		// from the same goroutine or from separate goroutines.
		const iterations = 1000
		uuid.SetRand(randutil.GlobalRand)
		var wg sync.WaitGroup
		uuids := new(sync.Map)
		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					uuid, err := uuid.NewRandom()
					require.NoError(t, err, "new() error")
					_, ok := uuids.Load(uuid)
					require.Falsef(t, ok, "goroutine %d returned a duplicate UUID on iteration %d: %v", i, j, uuid)
					uuids.Store(uuid, true)
				}
			}(i)
		}
		wg.Wait()
	})
	t.Run("pooled xrand 1 UUID x 1,000,000 goroutines", func(t *testing.T) {
		// Read a UUID from each of 1,000,000 goroutines and assert that there is never a duplicate value.
		uuid.SetRand(randutil.GlobalRand)
		uuid.EnableRandPool()
		defer uuid.DisableRandPool()
		uuids := new(sync.Map)
		var wg sync.WaitGroup
		for i := 1; i < 1000000; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				uuid, err := uuid.NewRandom()
				require.NoError(t, err, "new() error")
				_, ok := uuids.Load(uuid)
				require.Falsef(t, ok, "New returned a duplicate UUID on iteration %d: %v", i, uuid)
				uuids.Store(uuid, true)
			}(i)
		}
		wg.Wait()
	})
	t.Run("pooled xrand 1,000 UUIDs x 1,000 goroutines", func(t *testing.T) {
		// Read 1,000 UUIDs from each goroutine and assert that there is never a duplicate value, either
		// from the same goroutine or from separate goroutines.
		const iterations = 1000
		uuid.SetRand(randutil.GlobalRand)
		uuid.EnableRandPool()
		defer uuid.DisableRandPool()
		var wg sync.WaitGroup
		uuids := new(sync.Map)
		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					uuid, err := uuid.NewRandom()
					require.NoError(t, err, "new() error")
					_, ok := uuids.Load(uuid)
					require.Falsef(t, ok, "source %d returned a duplicate UUID on iteration %d: %v", i, j, uuid)
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
