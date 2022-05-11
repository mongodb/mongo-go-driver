// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package uuid

import (
	"go.mongodb.org/mongo-driver/internal/randutil"
	"go.mongodb.org/mongo-driver/internal/uuid/uuid"
)

// UUID exposes uuid.UUID.
type UUID uuid.UUID

// A source is a wrapper of google/uuid.
type source struct{}

// new returns a random UUIDv4 from uuid.NewRandom.
func (s *source) new() (UUID, error) {
	uuid, err := uuid.NewRandom()
	return (UUID)(uuid), err
}

// globalSource is a package-global pseudo-random UUID generator.
var globalSource = func() *source {
	uuid.SetRand(randutil.GlobalRand)
	uuid.EnableRandPool()
	return new(source)
}()

// New returns a random UUIDv4. It uses a global pseudo-random number generator in randutil
// at package initialization.
//
// New should not be used to generate cryptographically-secure random UUIDs.
func New() (UUID, error) {
	return globalSource.new()
}

// Equal returns true if two UUIDs are equal.
func Equal(a, b UUID) bool {
	return a == b
}
