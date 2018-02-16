// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package conn_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo/model"
	. "github.com/mongodb/mongo-go-driver/mongo/private/conn"
	"github.com/mongodb/mongo-go-driver/mongo/private/msg"
	"github.com/stretchr/testify/require"
)

func createIntegrationTestConn(opts ...Option) (Connection, error) {
	opts = append(opts, WithAppName("mongo-go-driver-test"))

	caFile := os.Getenv("MONGO_GO_DRIVER_CA_FILE")
	if len(caFile) != 0 {
		tls := NewTLSConfig()
		err := tls.AddCaCertFromFile(caFile)
		if err != nil {
			return nil, err
		}

		tls.SetInsecure(true)

		opts = append(opts, WithTLSConfig(tls))
	}

	c, err := New(context.Background(), model.Addr(*host), opts...)

	if err != nil {
		return nil, fmt.Errorf("failed dialing mongodb server - ensure that one is running at %s: %v", *host, err)
	}
	return c, nil
}

func TestConn_Initialize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConn()
	require.NoError(t, err)

	require.True(t, subject.Alive())
	require.False(t, subject.Expired())
}

func TestConn_ReadWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConn()
	require.NoError(t, err)

	isMasterRequest := msg.NewCommand(
		msg.NextRequestID(),
		"admin",
		true,
		bson.NewDocument(bson.EC.Int32("ismaster", 1)),
	)

	ctx, cancel := context.WithCancel(context.Background())
	err = subject.Write(ctx, isMasterRequest)
	require.NoError(t, err)
	_, err = subject.Read(ctx, isMasterRequest.RequestID())
	require.NoError(t, err)
	cancel()

	require.NotEmpty(t, subject.Model())
}

func TestConn_Expired_due_to_idle_time(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConn(WithIdleTimeout(2 * time.Second))
	require.NoError(t, err)

	require.False(t, subject.Expired())
	time.Sleep(4 * time.Second)
	require.True(t, subject.Expired())
}

func TestConn_Expired_due_to_life_time(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConn(WithLifeTimeout(2 * time.Second))
	require.NoError(t, err)

	require.False(t, subject.Expired())
	time.Sleep(4 * time.Second)
	require.True(t, subject.Expired())
}

func TestConnection_Write_cancel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConn()
	require.NoError(t, err)

	isMasterRequest := msg.NewCommand(
		msg.NextRequestID(),
		"admin",
		true,
		bson.NewDocument(bson.EC.Int32("ismaster", 1)),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = subject.Write(ctx, isMasterRequest)
	require.Error(t, err)

	require.True(t, subject.Alive())
}

func TestConnection_Write_timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConn()
	require.NoError(t, err)

	isMasterRequest := msg.NewCommand(
		msg.NextRequestID(),
		"admin",
		true,
		bson.NewDocument(bson.EC.Int32("ismaster", 1)),
	)

	err = subject.Write(&timeoutContext{}, isMasterRequest)
	require.Error(t, err)
	require.False(t, subject.Alive())
}

func TestConnection_Write_after_connection_is_dead(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConn()
	require.NoError(t, err)

	isMasterRequest := msg.NewCommand(
		msg.NextRequestID(),
		"admin",
		true,
		bson.NewDocument(bson.EC.Int32("ismaster", 1)),
	)

	err = subject.Write(&timeoutContext{}, isMasterRequest)
	require.Error(t, err)
	require.False(t, subject.Alive())
	err = subject.Write(context.Background(), isMasterRequest)
	require.Error(t, err)
}

func TestConnection_Read_cancel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConn()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = subject.Read(ctx, 0)
	require.Error(t, err)

	require.False(t, subject.Alive())
}

func TestConnection_Read_timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConn()
	require.NoError(t, err)

	_, err = subject.Read(&timeoutContext{}, 0)
	require.Error(t, err)

	require.False(t, subject.Alive())
}

func TestConnection_LocalAddr(t *testing.T) {
	t.Parallel()
	subject, err := createIntegrationTestConn()
	require.NoError(t, err)
	localAddr := subject.LocalAddr()
	require.NotEqual(t, localAddr, &net.TCPAddr{})
}

func TestConnection_Read_after_connection_is_dead(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConn()
	require.NoError(t, err)

	_, err = subject.Read(&timeoutContext{}, 0)
	require.Error(t, err)
	require.False(t, subject.Alive())
	_, err = subject.Read(context.Background(), 0)
	require.Error(t, err)
}

type timeoutContext struct {
	context.Context
}

func (c *timeoutContext) Deadline() (time.Time, bool) {
	return time.Now(), true
}

func (c *timeoutContext) Done() <-chan struct{} {
	return make(chan struct{})
}

func (c *timeoutContext) Err() error {
	return nil
}

func (c *timeoutContext) Value(key interface{}) interface{} {
	return nil
}
