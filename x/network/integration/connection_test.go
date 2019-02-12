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
	"go.mongodb.org/mongo-driver/x/network/wiremessage"
)

func TestConnection(t *testing.T) {
	isMaster, err := (&command.IsMaster{}).Encode()
	if err != nil {
		t.Errorf("Unexpected error while marshaling wire message: %v", err)
	}

	t.Run("New", func(t *testing.T) {
		// address := address.Address(*host)
		// _, err := connection.New(context.Background(), address)
		// if err != nil {
		// 	t.Errorf("Whoops: %s", err)
		// }
	})
	t.Run("Initialize", func(t *testing.T) {
		c := testConnection(t)
		defer c.Close()
		if c.Alive() != true {
			t.Errorf("Newly created connection should be alive.")
		}

		if c.Expired() != false {
			t.Errorf("Newly created connection should not be expired.")
		}
	})
	t.Run("ReadWrite", func(t *testing.T) {
		c := testConnection(t)
		defer c.Close()

		ctx, cancel := context.WithCancel(context.Background())
		err = c.WriteWireMessage(ctx, isMaster)
		if err != nil {
			t.Errorf("Unexpected error while writing wire message: %v", err)
		}
		wm, err := c.ReadWireMessage(ctx)
		if err != nil {
			t.Errorf("Unexpected error while reading wire message: %v", err)
		}
		cancel()

		if _, ok := wm.(wiremessage.Reply); !ok {
			t.Errorf("Unexpected return wiremessage: %v", wm)
		}
	})

	t.Run("Expired", func(t *testing.T) {
		t.Run("Idle Timeout", func(t *testing.T) {
			c := testConnection(t, connection.WithIdleTimeout(func(time.Duration) time.Duration { return 50 * time.Millisecond }))
			defer c.Close()
			if c.Expired() != false {
				t.Errorf("Newly created connection should not be expired.")
			}
			time.Sleep(100 * time.Millisecond)
			if c.Expired() != true {
				t.Errorf("Connection should be expired after idle timeout.")
			}
		})
		t.Run("Lifetime Timeout", func(t *testing.T) {
			c := testConnection(t, connection.WithLifeTimeout(func(time.Duration) time.Duration { return 50 * time.Millisecond }))
			defer c.Close()
			if c.Expired() != false {
				t.Errorf("Newly created connection should not be expired.")
			}
			time.Sleep(100 * time.Millisecond)
			if c.Expired() != true {
				t.Errorf("Connection should be expired after idle timeout.")
			}
		})
	})
	t.Run("WriteContext", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()
		testCases := []struct {
			name      string
			ctx       context.Context
			alive     bool
			errSubStr string
		}{
			{"Cancel", cancelledCtx, true, "context canceled"},
			{"Timeout", timeoutCtx{}, false, "i/o timeout"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				c := testConnection(t)
				err := c.WriteWireMessage(tc.ctx, isMaster)
				if !strings.Contains(err.Error(), tc.errSubStr) {
					t.Errorf("Expected context error message but got: %v", err)
				}
				alive := c.Alive()
				if alive != tc.alive {
					t.Errorf("Unexpected alive state. got %t; want %t", alive, tc.alive)
				}
			})
		}
	})
	t.Run("Write After Connection Dead", func(t *testing.T) {
		c := testConnection(t)
		c.Close()
		if c.Alive() != false {
			t.Errorf("Connection should be dead after calling Close.")
		}
		err := c.WriteWireMessage(context.Background(), isMaster)
		if !strings.Contains(err.Error(), "connection is dead") {
			t.Errorf("Writing to dead connection should return connection is dead error, but got: %v", err)
		}
	})
	t.Run("ReadContext", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()
		testCases := []struct {
			name      string
			ctx       context.Context
			alive     bool
			errSubStr string
		}{
			{"Cancel", cancelledCtx, false, "context canceled"},
			{"Timeout", timeoutCtx{}, false, "i/o timeout"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				c := testConnection(t)
				_, err := c.ReadWireMessage(tc.ctx)
				if !strings.Contains(err.Error(), tc.errSubStr) {
					t.Errorf("Expected context error message but got: %v", err)
				}
				alive := c.Alive()
				if alive != tc.alive {
					t.Errorf("Unexpected alive state. got %t; want %t", alive, tc.alive)
				}
			})
		}
	})
	t.Run("Read After Connection Dead", func(t *testing.T) {
		c := testConnection(t)
		c.Close()
		if c.Alive() != false {
			t.Errorf("Connection should be dead after calling Close.")
		}
		_, err := c.ReadWireMessage(context.Background())
		if !strings.Contains(err.Error(), "connection is dead") {
			t.Errorf("Writing to dead connection should return connection is dead error, but got: %v", err)
		}
	})
}

func testConnection(t *testing.T, opts ...connection.Option) connection.Connection {
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

	opts = append(opts, connection.WithHandshaker(func(connection.Handshaker) connection.Handshaker {
		return &command.Handshake{Client: command.ClientDoc("mongo-go-driver-test")}
	}))

	c, _, err := connection.New(context.Background(), address.Address(*host), opts...)
	if err != nil {
		t.Errorf("failed dialing mongodb server - ensure that one is running at %s: %v", *host, err)
		t.FailNow()
	}

	return c
}

type timeoutCtx struct{ context.Context }

func (tc timeoutCtx) Deadline() (time.Time, bool)       { return time.Now(), true }
func (tc timeoutCtx) Done() <-chan struct{}             { return make(chan struct{}) }
func (tc timeoutCtx) Err() error                        { return nil }
func (tc timeoutCtx) Value(key interface{}) interface{} { return nil }
