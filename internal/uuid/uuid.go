// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package uuid

import (
	"io"

	"go.mongodb.org/mongo-driver/internal/randutil"
)

// UUID represents a UUID.
type UUID [16]byte

// globalRandom is a package-global pseudo-random number generator.
// It should be safe to use from multiple goroutines.
var globalRandom = randutil.NewLockedRand()

// New returns a random UUIDv4. It uses a global pseudo-random number generator in randutil
// at package initialization.
//
// New should not be used to generate cryptographically-secure random UUIDs.
func New() (UUID, error) {
	var uuid UUID
	_, err := io.ReadFull(globalRandom, uuid[:])
	if err != nil {
		return UUID{}, err
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10
	return uuid, nil
}
