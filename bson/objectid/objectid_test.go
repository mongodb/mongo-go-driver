// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package objectid

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	// Ensure that objectid.New() doesn't panic.
	New()
}

func TestString(t *testing.T) {
	id := New()
	require.Contains(t, id.String(), id.Hex())
}

func TestFromHex_RoundTrip(t *testing.T) {
	before := New()
	after, err := FromHex(before.Hex())
	require.NoError(t, err)

	require.Equal(t, before, after)
}

func TestFromHex_InvalidHex(t *testing.T) {
	_, err := FromHex("this is not a valid hex string!")
	require.Error(t, err)
}

func TestFromHex_WrongLength(t *testing.T) {
	_, err := FromHex("deadbeef")
	require.Equal(t, ErrInvalidHex, err)
}
