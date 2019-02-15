// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package primitive

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUUIDNewUUID(t *testing.T) {
	NewUUID()
}

func TestUUIDNewUUIDV1(t *testing.T) {
	NewUUIDV1()
}

func TestUUIDNewUUIDV4(t *testing.T) {
	NewUUIDV4()
}

func TestUUIDString(t *testing.T) {
	id := NewUUIDV4()
	require.Equal(t, len(id.String()), 36)
}

func TestUUIDParse(t *testing.T) {
	testCases := []struct {
		Hex      string
		Expected string
	}{
		{
			"1239af32-282c-4200-b373-81c3ab8f8c61",
			"1239af32-282c-4200-b373-81c3ab8f8c61",
		},
		{
			"urn:uuid:1239af32-282c-4200-b373-81c3ab8f8c61",
			"1239af32-282c-4200-b373-81c3ab8f8c61",
		},
		{
			"{1239af32-282c-4200-b373-81c3ab8f8c61}",
			"1239af32-282c-4200-b373-81c3ab8f8c61",
		},
		{
			"1239af32282c4200b37381c3ab8f8c61",
			"1239af32-282c-4200-b373-81c3ab8f8c61",
		},
	}

	for _, testcase := range testCases {
		id, err := Parse(testcase.Hex)
		require.NoError(t, err)

		require.Equal(t, testcase.Expected, id.String())
	}
}
