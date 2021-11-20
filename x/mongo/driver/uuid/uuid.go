// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package uuid // import "go.mongodb.org/mongo-driver/x/mongo/driver/uuid"

import (
	"io"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/internal/randutil"
)

// UUID represents a UUID.
type UUID [16]byte

// random is a package-global pseudo-random number generator.
var random = randutil.NewLockedRand(rand.NewSource(time.Now().UnixNano()))

// New returns a random UUIDv4. It uses a "math/rand" pseudo-random number generator seeded with the
// package initialization time.
//
// New should not be used to generate cryptographically-secure random UUIDs.
func New() (UUID, error) {
	var uuid [16]byte

	_, err := io.ReadFull(random, uuid[:])
	if err != nil {
		return [16]byte{}, err
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10

	return uuid, nil
}

// Equal returns true if two UUIDs are equal.
func Equal(a, b UUID) bool {
	return a == b
}
