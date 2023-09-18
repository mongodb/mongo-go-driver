// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestDirectConnectionFromConnString(t *testing.T) {
	type testCaseQuery [][2]string

	testCases := []struct {
		name  string
		mode  MonitorMode
		query *testCaseQuery
	}{
		{"connect=direct", SingleMode, &testCaseQuery{{"connect", "direct"}}},
		{"connect=automatic", AutomaticMode, &testCaseQuery{{"connect", "automatic"}}},
		{"directConnection=true", SingleMode, &testCaseQuery{{"directconnection", "true"}}},
		{"directconnection=false", AutomaticMode, &testCaseQuery{{"directconnection", "false"}}},
		{"default", AutomaticMode, nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dns, err := url.Parse("mongodb://localhost/")
			assert.Nil(t, err, "error parsing uri: %v", err)

			if tc.query != nil {
				query := dns.Query()
				for _, q := range *tc.query {
					query.Set(q[0], q[1])
				}
				dns.RawQuery = query.Encode()
			}

			cfg, err := NewConfig(options.Client().ApplyURI(dns.String()), nil)
			assert.Nil(t, err, "error constructing topology config: %v", err)

			topo, err := New(cfg)
			assert.Nil(t, err, "error constructing topology: %v", err)
			assert.Equal(t, tc.mode, topo.cfg.Mode, "expected mode %v, got %v", tc.mode, topo.cfg.Mode)
		})
	}
}

func TestLoadBalancedFromConnString(t *testing.T) {
	testCases := []struct {
		name         string
		uriOptions   string
		loadBalanced bool
	}{
		{"loadBalanced=true", "loadBalanced=true", true},
		{"loadBalanced=false", "loadBalanced=false", false},
		{"loadBalanced unset", "", false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			uri := fmt.Sprintf("mongodb://localhost/?%s", tc.uriOptions)
			cfg, err := NewConfig(options.Client().ApplyURI(uri), nil)
			assert.Nil(t, err, "error constructing topology config: %v", err)

			topo, err := New(cfg)
			assert.Nil(t, err, "topology.New error: %v", err)
			assert.Equal(t, tc.loadBalanced, topo.cfg.LoadBalanced, "expected loadBalanced %v, got %v", tc.loadBalanced, topo.cfg.LoadBalanced)

			srvr := NewServer("", topo.id, topo.cfg.ServerOpts...)
			assert.Equal(t, tc.loadBalanced, srvr.cfg.loadBalanced, "expected loadBalanced %v, got %v", tc.loadBalanced, srvr.cfg.loadBalanced)

			conn := newConnection("", srvr.cfg.connectionOpts...)
			assert.Equal(t, tc.loadBalanced, conn.config.loadBalanced, "expected loadBalanced %v, got %v", tc.loadBalanced, conn.config.loadBalanced)
		})
	}
}

func TestTopologyNewConfig(t *testing.T) {
	t.Run("default ServerSelectionTimeout", func(t *testing.T) {
		cfg, err := NewConfig(options.Client(), nil)
		assert.Nil(t, err, "error constructing topology config: %v", err)
		assert.Equal(t, defaultServerSelectionTimeout, cfg.ServerSelectionTimeout)
	})
	t.Run("non-default ServerSelectionTimeout", func(t *testing.T) {
		cfg, err := NewConfig(options.Client().SetServerSelectionTimeout(1), nil)
		assert.Nil(t, err, "error constructing topology config: %v", err)
		assert.Equal(t, time.Duration(1), cfg.ServerSelectionTimeout)
	})
	t.Run("default SeedList", func(t *testing.T) {
		cfg, err := NewConfig(options.Client(), nil)
		assert.Nil(t, err, "error constructing topology config: %v", err)
		assert.Equal(t, []string{"localhost:27017"}, cfg.SeedList)
	})
	t.Run("non-default SeedList", func(t *testing.T) {
		cfg, err := NewConfig(options.Client().ApplyURI("mongodb://localhost:27018"), nil)
		assert.Nil(t, err, "error constructing topology config: %v", err)
		assert.Equal(t, []string{"localhost:27018"}, cfg.SeedList)
	})
}
