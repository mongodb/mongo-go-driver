// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/x/network/address"
	"go.mongodb.org/mongo-driver/x/network/command"
	"go.mongodb.org/mongo-driver/x/network/connection"
)

func TestPool(t *testing.T) {
	noerr := func(t *testing.T, err error) {
		if err != nil {
			t.Helper()
			t.Errorf("Unepexted error: %v", err)
			t.FailNow()
		}
	}
	opts := []connection.Option{connection.WithAppName(func(string) string { return "mongo-go-driver-test" })}
	opts = append(opts, connection.WithHandshaker(func(connection.Handshaker) connection.Handshaker {
		return &command.Handshake{Client: command.ClientDoc("mongo-go-driver-test")}
	}))

	caFile := os.Getenv("MONGO_GO_DRIVER_CA_FILE")
	if len(caFile) != 0 {
		config := connection.NewTLSConfig()
		err := config.AddCACertFromFile(caFile)
		if err != nil {
			t.Errorf("Unexpected error while adding ca file to config: %v", err)
			t.FailNow()
		}

		config.SetInsecure(true)

		opts = append(opts, connection.WithTLSConfig(func(*connection.TLSConfig) *connection.TLSConfig { return config }))
	}
	t.Run("Cannot Create Pool With Size Larger Than Capacity", func(t *testing.T) {
		_, err := connection.NewPool(address.Address(""), 4, 2, opts...)
		if err != connection.ErrSizeLargerThanCapacity {
			t.Errorf("Should not be able to create a pool with size larger than capacity. got %v; want %v", err, connection.ErrSizeLargerThanCapacity)
		}
	})
	t.Run("Reuses Connections", func(t *testing.T) {
		// TODO(skriptble): make this a table test.
		p, err := connection.NewPool(address.Address(*host), 2, 4, opts...)
		if err != nil {
			t.Errorf("Unexpected error while creating pool: %v", err)
		}
		err = p.Connect(context.TODO())
		noerr(t, err)

		c1, _, err := p.Get(context.Background())
		noerr(t, err)
		first := c1.ID()
		err = c1.Close()
		noerr(t, err)
		c2, _, err := p.Get(context.Background())
		noerr(t, err)
		second := c2.ID()
		if first != second {
			t.Errorf("Pool does not reuse connections. The connection ids differ. first %s; second %s", first, second)
		}
	})
	t.Run("Expired Connections Aren't Returned", func(t *testing.T) {
		p, err := connection.NewPool(address.Address(*host), 2, 4,
			append(opts, connection.WithIdleTimeout(func(time.Duration) time.Duration { return 10 * time.Millisecond }))...,
		)
		if err != nil {
			t.Errorf("Unexpected error while creating pool: %v", err)
		}
		err = p.Connect(context.TODO())
		noerr(t, err)
		c1, _, err := p.Get(context.Background())
		noerr(t, err)
		first := c1.ID()
		err = c1.Close()
		noerr(t, err)
		time.Sleep(400 * time.Millisecond)
		c2, _, err := p.Get(context.Background())
		noerr(t, err)
		second := c2.ID()
		if c1 == c2 {
			t.Errorf("Pool does not expire connections. IDs of first and second connection match. first %s; second %s", first, second)
		}
	})
	t.Run("Get With Done Context", func(t *testing.T) {
		p, err := connection.NewPool(address.Address(*host), 2, 4, opts...)
		if err != nil {
			t.Errorf("Unexpected error while creating pool: %v", err)
		}
		err = p.Connect(context.TODO())
		noerr(t, err)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, _, err = p.Get(ctx)
		if !strings.Contains(err.Error(), "context canceled") {
			t.Errorf("Expected context called error, but got: %v", err)
		}
	})
	t.Run("Get Returns Error From Creating A Connection", func(t *testing.T) {
		p, err := connection.NewPool(address.Address("localhost:0"), 2, 4, opts...)
		if err != nil {
			t.Errorf("Unexpected error while creating pool: %v", err)
		}
		err = p.Connect(context.TODO())
		noerr(t, err)
		_, _, err = p.Get(context.Background())
		if !strings.Contains(err.Error(), "dial tcp") {
			t.Errorf("Expected context called error, but got: %v", err)
		}
	})
	t.Run("Get Returns An Error After Pool Is Closed", func(t *testing.T) {
		p, err := connection.NewPool(address.Address(*host), 2, 4, opts...)
		if err != nil {
			t.Errorf("Unexpected error while creating pool: %v", err)
		}
		err = p.Connect(context.TODO())
		noerr(t, err)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		err = p.Disconnect(ctx)
		noerr(t, err)
		_, _, err = p.Get(context.Background())
		if err != connection.ErrPoolClosed {
			t.Errorf("Did not get expected error. got %v; want %v", err, connection.ErrPoolClosed)
		}
	})
	t.Run("Connection Close Does Not Error After Pool Is Closed", func(t *testing.T) {
		p, err := connection.NewPool(address.Address(*host), 2, 4, opts...)
		if err != nil {
			t.Errorf("Unexpected error while creating pool: %v", err)
		}
		err = p.Connect(context.TODO())
		noerr(t, err)
		c1, _, err := p.Get(context.Background())
		noerr(t, err)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		err = p.Disconnect(ctx)
		noerr(t, err)
		err = c1.Close()
		if err != nil {
			t.Errorf("Connection Close should not error after Pool is closed, but got error: %v", err)
		}
	})
	t.Run("Connection Close Does Not Close Underlying Connection If Not Expired", func(t *testing.T) {
		// Implement this once there is a more testable Dialer.
		t.Skip()
	})
	t.Run("Connection Close Does Close Underlying Connection If Expired", func(t *testing.T) {
		// Implement this once there is a more testable Dialer.
		t.Skip()
	})
	t.Run("Connection Close Closes Underlying Connection When Size Is Exceeded", func(t *testing.T) {
		// Implement this once there is a more testable Dialer.
		t.Skip()
	})
	t.Run("Drain Expires Existing Checked Out Connections", func(t *testing.T) {
		p, err := connection.NewPool(address.Address(*host), 2, 4, opts...)
		if err != nil {
			t.Errorf("Unexpected error while creating pool: %v", err)
		}
		err = p.Connect(context.TODO())
		noerr(t, err)
		c1, _, err := p.Get(context.Background())
		noerr(t, err)
		if c1.Expired() != false {
			t.Errorf("Newly retrieved connection should not be expired.")
		}
		err = p.Drain()
		noerr(t, err)
		if c1.Expired() != true {
			t.Errorf("Existing checkout out connections should be expired once pool is drained.")
		}
	})
	t.Run("Drain Expires Idle Connections", func(t *testing.T) {
		// Implement this once there is a more testable Dialer.
		t.Skip()
	})
	t.Run("Pool Close Closes All Connections In A Pool", func(t *testing.T) {
		// Implement this once there is a more testable Dialer.
		t.Skip()
	})
}
