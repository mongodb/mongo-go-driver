// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"net"
	"os"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
)

func TestClientOptions_CustomDialer(t *testing.T) {
	td := &testDialer{d: &net.Dialer{}}
	cs := testutil.ConnString(t)
	opts := options.Client().ApplyURI(cs.String()).SetDialer(td)
	if os.Getenv("REQUIRE_API_VERSION") == "true" {
		opts.SetServerAPIOptions(options.ServerAPI().SetServerAPIVersion(driver.LatestServerAPIVersion))
	}
	client, err := NewClient(opts)
	require.NoError(t, err)
	err = client.Connect(context.Background())
	require.NoError(t, err)
	_, err = client.ListDatabases(context.Background(), bsonx.Doc{})
	require.NoError(t, err)
	got := atomic.LoadInt32(&td.called)
	if got < 1 {
		t.Errorf("Custom dialer was not used when dialing new connections")
	}
}

type testDialer struct {
	called int32
	d      Dialer
}

func (td *testDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	atomic.AddInt32(&td.called, 1)
	return td.d.DialContext(ctx, network, address)
}
