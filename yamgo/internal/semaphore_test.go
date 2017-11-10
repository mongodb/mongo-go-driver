// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package internal_test

import (
	"context"
	"testing"
	"time"

	. "github.com/10gen/mongo-go-driver/yamgo/internal"
	"github.com/stretchr/testify/require"
)

func TestSemaphore_Wait(t *testing.T) {
	s := NewSemaphore(3)
	err := s.Wait(context.Background())
	require.NoError(t, err)
	err = s.Wait(context.Background())
	require.NoError(t, err)
	err = s.Wait(context.Background())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err = s.Wait(ctx)
		require.Error(t, err)
	}()

	time.Sleep(1 * time.Second)
	cancel()
}

func TestSemaphore_Release(t *testing.T) {
	s := NewSemaphore(3)
	err := s.Wait(context.Background())
	err = s.Wait(context.Background())
	err = s.Wait(context.Background())

	go func() {
		err = s.Wait(context.Background())
		require.NoError(t, err)
	}()

	time.Sleep(1 * time.Second)
	s.Release()
}
