// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package uuid // import "go.mongodb.org/mongo-driver/x/mongo/driver/uuid"

import (
	"io"
	"math/rand"

	"go.mongodb.org/mongo-driver/internal/randutil"
)

// UUID represents a UUID.
type UUID [16]byte

// A source is a UUID generator that reads random values from a randutil.LockedRand.
// It is safe to use from multiple goroutines.
type source struct {
	random *randutil.LockedRand
}

// new returns a random UUIDv4 with bytes read from the source's random number generator.
func (s *source) new() (UUID, error) {
	var uuid [16]byte

	_, err := io.ReadFull(s.random, uuid[:])
	if err != nil {
		return [16]byte{}, err
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10

	return uuid, nil
}

// newGlobalSource returns a source that uses a "math/rand" pseudo-random number generator seeded
// with a cryptographically-secure random number. It is intended to be used to initialize the
// package-global UUID generator.
func newGlobalSource() *source {
	return &source{
		random: randutil.NewLockedRand(rand.NewSource(randutil.CryptoSeed())),
	}
}

// globalSource is a package-global pseudo-random UUID generator.
var globalSource = newGlobalSource()

// New returns a random UUIDv4. It uses a "math/rand" pseudo-random number generator seeded with a
// cryptographically-secure random number at package initialization.
//
// New should not be used to generate cryptographically-secure random UUIDs.
func New() (UUID, error) {
	return globalSource.new()
}

// Equal returns true if two UUIDs are equal.
func Equal(a, b UUID) bool {
	return a == b
}
