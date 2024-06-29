// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
//
// Based on gopkg.in/mgo.v2/bson by Gustavo Niemeyer
// See THIRD-PARTY-NOTICES for original license terms.

package bson

import (
	"crypto/rand"
	"encoding"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"time"
)

// ErrInvalidHex indicates that a hex string cannot be converted to an ObjectID.
var ErrInvalidHex = errors.New("the provided hex string is not a valid ObjectID")

// ObjectID is the BSON ObjectID type.
type ObjectID [12]byte

// NilObjectID is the zero value for ObjectID.
var NilObjectID ObjectID

var objectIDCounter = readRandomUint32()
var processUnique = processUniqueBytes()

var _ encoding.TextMarshaler = ObjectID{}
var _ encoding.TextUnmarshaler = &ObjectID{}

// NewObjectID generates a new ObjectID.
func NewObjectID() ObjectID {
	return NewObjectIDFromTimestamp(time.Now())
}

// NewObjectIDFromTimestamp generates a new ObjectID based on the given time.
func NewObjectIDFromTimestamp(timestamp time.Time) ObjectID {
	var b [12]byte

	binary.BigEndian.PutUint32(b[0:4], uint32(timestamp.Unix()))
	copy(b[4:9], processUnique[:])
	putUint24(b[9:12], atomic.AddUint32(&objectIDCounter, 1))

	return b
}

// Timestamp extracts the time part of the ObjectId.
func (id ObjectID) Timestamp() time.Time {
	unixSecs := binary.BigEndian.Uint32(id[0:4])
	return time.Unix(int64(unixSecs), 0).UTC()
}

// Hex returns the hex encoding of the ObjectID as a string.
func (id ObjectID) Hex() string {
	var buf [24]byte
	hex.Encode(buf[:], id[:])
	return string(buf[:])
}

func (id ObjectID) String() string {
	return `ObjectID("` + id.Hex() + `")`
}

// IsZero returns true if id is the empty ObjectID.
func (id ObjectID) IsZero() bool {
	return id == NilObjectID
}

// ObjectIDFromHex creates a new ObjectID from a hex string. It returns an error if the hex string is not a
// valid ObjectID.
func ObjectIDFromHex(s string) (ObjectID, error) {
	if len(s) != 24 {
		return NilObjectID, ErrInvalidHex
	}

	var oid [12]byte
	_, err := hex.Decode(oid[:], []byte(s))
	if err != nil {
		return NilObjectID, err
	}

	return oid, nil
}

// MarshalText returns the ObjectID as UTF-8-encoded text. Implementing this allows us to use ObjectID
// as a map key when marshalling JSON. See https://pkg.go.dev/encoding#TextMarshaler
func (id ObjectID) MarshalText() ([]byte, error) {
	var buf [24]byte
	hex.Encode(buf[:], id[:])
	return buf[:], nil
}

// UnmarshalText populates the byte slice with the ObjectID. If the byte slice
// is 24 bytes long, it will be populated with the hex representation of the
// ObjectID. If the byte slice is 12 bytes long, it will be populated with the
// BSON representation of the ObjectID. This method also accepts empty strings
// and decodes them as NilObjectID.
//
// For any other inputs, an error will be returned.
//
// Implementing this allows us to use ObjectID as a map key when unmarshalling
// JSON. See https://pkg.go.dev/encoding#TextUnmarshaler
func (id *ObjectID) UnmarshalText(b []byte) error {
	// NB(charlie): The json package will use UnmarshalText instead of
	// UnmarshalJSON if the value is a string.

	// An empty string is not a valid ObjectID, but we treat it as a
	// special value that decodes as NilObjectID.
	if len(b) == 0 {
		return nil
	}
	oid, err := ObjectIDFromHex(string(b))
	if err != nil {
		return err
	}
	*id = oid
	return nil
}

// MarshalJSON returns the ObjectID as a string
func (id ObjectID) MarshalJSON() ([]byte, error) {
	var buf [26]byte
	buf[0] = '"'
	hex.Encode(buf[1:25], id[:])
	buf[25] = '"'
	return buf[:], nil
}

// UnmarshalJSON populates the byte slice with the ObjectID. If the byte slice
// is 24 bytes long, it will be populated with the hex representation of the
// ObjectID. If the byte slice is 12 bytes long, it will be populated with the
// BSON representation of the ObjectID. This method also accepts empty strings
// and decodes them as NilObjectID.
//
// As a special case UnmarshalJSON will decode a JSON object with key "$oid"
// that stores a hex encoded ObjectID: {"$oid": "65b3f7edd9bfca00daa6e3b31"}.
//
// For any other inputs, an error will be returned.
func (id *ObjectID) UnmarshalJSON(b []byte) error {
	// Ignore "null" to keep parity with the standard library. Decoding a JSON
	// null into a non-pointer ObjectID field will leave the field unchanged.
	// For pointer values, encoding/json will set the pointer to nil and will
	// not enter the UnmarshalJSON hook.
	if string(b) == "null" {
		return nil
	}

	// Handle string
	if len(b) >= 2 && b[0] == '"' {
		// TODO: fails because of error
		return id.UnmarshalText(b[1 : len(b)-1])
	}
	if len(b) == 12 {
		copy(id[:], b)
		return nil
	}
	var v struct {
		OID *string `json:"$oid"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return fmt.Errorf("failed to parse extended JSON ObjectID: %w", err)
	}
	if v.OID == nil {
		return errors.New("not an extended JSON ObjectID")
	}
	i, err := ObjectIDFromHex(*v.OID)
	if err != nil {
		return err
	}
	*id = i
	return nil
}

func processUniqueBytes() [5]byte {
	var b [5]byte
	_, err := io.ReadFull(rand.Reader, b[:])
	if err != nil {
		panic(fmt.Errorf("cannot initialize objectid package with crypto.rand.Reader: %w", err))
	}

	return b
}

func readRandomUint32() uint32 {
	var b [4]byte
	_, err := io.ReadFull(rand.Reader, b[:])
	if err != nil {
		panic(fmt.Errorf("cannot initialize objectid package with crypto.rand.Reader: %w", err))
	}

	return (uint32(b[0]) << 0) | (uint32(b[1]) << 8) | (uint32(b[2]) << 16) | (uint32(b[3]) << 24)
}

func putUint24(b []byte, v uint32) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}
