// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
//
// Based on gopkg.in/mgo.v2/bson by Gustavo Niemeyer
// See THIRD-PARTY-NOTICES for original license terms.

package primitive

import (
	"bytes"
	"encoding/json"

	"github.com/google/uuid"
)

// UUID is the BSON UUID type.
type UUID [16]byte

// NilUUID is the zero value for UUID.
var NilUUID UUID

// NewUUIDV1 returns a Version 1 UUID or panics. UUID is based on the current
// NodeID and clock sequence, and the current time.
func NewUUIDV1() UUID {
	return UUID(uuid.Must(uuid.NewUUID()))
}

// NewUUIDV4 returns a Version 4 UUID or panics. UUID is is equivalent to
// the expression - uuid.Must(uuid.NewRandom())
func NewUUIDV4() UUID {
	return UUID(uuid.New())
}

// NewUUID returns a Version 4 UUID or panics.
// In most cases this should be used
func NewUUID() UUID {
	return NewUUIDV4()
}

// String returns the string form of uuid, xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
// or "" if uuid is invalid.
func (id UUID) String() string {
	return uuid.UUID(id).String()
}

// Parse decodes s into a UUID or returns an error.  Both the standard UUID
// forms of xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx and
// urn:uuid:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx are decoded as well as the
// Microsoft encoding {xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx} and the raw hex
// encoding: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.
func Parse(s string) (UUID, error) {
	u, err := uuid.Parse(s)
	return UUID(u), err
}

// MustParse is like Parse but panics if the string cannot be parsed.
// It simplifies safe initialization of global variables holding compiled UUIDs.
func MustParse(s string) UUID {
	return UUID(uuid.MustParse(s))
}

// ParseBytes is like Parse, except it parses a byte slice instead of a string.
func ParseBytes(b []byte) (UUID, error) {
	u, err := uuid.ParseBytes(b)
	return UUID(u), err
}

// FromBytes creates a new UUID from a byte slice. Returns an error if the slice
// does not have a length of 16. The bytes are copied from the slice.
func FromBytes(b []byte) (id UUID, err error) {
	u, err := uuid.FromBytes(b)
	return UUID(u), err
}

// Must returns uuid if err is nil and panics otherwise.
func Must(id UUID, err error) UUID {
	return UUID(uuid.Must(uuid.UUID(id), err))
}

// MarshalJSON returns the UUID as a string
func (id UUID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.String())
}

// UnmarshalJSON populates the byte slice with the UUID.
func (id UUID) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, uuid.Must(uuid.ParseBytes(b)))
}

// IsZero returns true if id is the empty ObjectID.
func (id UUID) IsZero() bool {
	return bytes.Equal(id[:], NilUUID[:])
}

// Equal returns true if two UUIDs are equal.
func (id *UUID) Equal(b UUID) bool {
	return bytes.Equal([]byte(id[:]), []byte(b[:]))
}
