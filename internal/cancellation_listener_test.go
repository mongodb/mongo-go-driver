// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCancellationListenerDataRace makes sure that there is no data race
// when cancellation listener is used concurrently.
func TestCancellationListenerDataRace(t *testing.T) {
	l := NewCancellationListener()
	ctx := context.Background()
	go l.Listen(ctx, func() {})
	go l.Listen(ctx, func() {})
	require.False(t, l.StopListening())
}
