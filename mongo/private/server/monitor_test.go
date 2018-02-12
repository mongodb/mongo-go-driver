// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package server_test

import (
	"sync"
	"testing"

	"github.com/mongodb/mongo-go-driver/mongo/internal/servertest"
	"github.com/mongodb/mongo-go-driver/mongo/model"
	"github.com/stretchr/testify/require"
)

func TestMonitor_Close_should_close_all_update_channels(t *testing.T) {
	t.Parallel()

	fm := servertest.NewFakeMonitor(model.Standalone, model.Addr("localhost:27017"))

	updates1, _, _ := fm.Subscribe()
	done1 := false

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for range updates1 {
		}

		done1 = true
	}()

	updates2, _, _ := fm.Subscribe()
	done2 := false
	go func() {
		defer wg.Done()
		for range updates2 {
		}

		done2 = true
	}()

	fm.Stop()
	wg.Wait()

	require.True(t, done1)
	require.True(t, done2)
}

func TestMonitor_Subscribe_after_close_should_return_an_error(t *testing.T) {
	t.Parallel()

	fm := servertest.NewFakeMonitor(model.Standalone, model.Addr("localhost:27017"))
	updates, _, _ := fm.Subscribe()
	fm.Stop()

	// Ensure that the monitor is closed.
	for range updates {
	}

	_, _, err := fm.Subscribe()
	require.Error(t, err)
}
