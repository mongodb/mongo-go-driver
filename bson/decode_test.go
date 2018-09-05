// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func requireErrEqual(t *testing.T, err1 error, err2 error) {
	switch e := err1.(type) {
	case ErrTooSmall:
		require.True(t, e.Equals(err2))

		return
	}

	switch e := err2.(type) {
	case ErrTooSmall:
		require.True(t, e.Equals(e))

		return
	}

	require.Equal(t, err1, err2)
}

func elementSliceEqual(t *testing.T, e1 []*Element, e2 []*Element) {
	require.Equal(t, len(e1), len(e2))

	for i := range e1 {
		require.True(t, readerElementComparer(e1[i], e2[i]))
	}
}
