// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package conn_test

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/mongo/internal/conntest"
	"github.com/mongodb/mongo-go-driver/mongo/internal/testutil/helpers"
	. "github.com/mongodb/mongo-go-driver/mongo/private/conn"
	"github.com/stretchr/testify/require"
)

func TestTracked_Inc(t *testing.T) {
	t.Parallel()

	c := &conntest.MockConnection{}
	require.True(t, c.Alive())

	tracked := Tracked(c)
	require.True(t, tracked.Alive())
	require.True(t, c.Alive())

	tracked.Inc()
	require.True(t, tracked.Alive())
	require.True(t, c.Alive())

	testhelpers.RequireNoErrorOnClose(t, tracked)
	require.True(t, tracked.Alive())
	require.True(t, c.Alive())

	testhelpers.RequireNoErrorOnClose(t, tracked)
	require.False(t, tracked.Alive())
	require.False(t, c.Alive())
}
