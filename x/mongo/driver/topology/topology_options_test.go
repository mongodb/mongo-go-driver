// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/mnet"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/xoptions"
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

			srvr := NewServer("", topo.id, defaultConnectionTimeout, topo.cfg.ServerOpts...)
			assert.Equal(t, tc.loadBalanced, srvr.cfg.loadBalanced, "expected loadBalanced %v, got %v", tc.loadBalanced, srvr.cfg.loadBalanced)

			conn := newConnection("", srvr.cfg.connectionOpts...)
			assert.Equal(t, tc.loadBalanced, conn.config.loadBalanced, "expected loadBalanced %v, got %v", tc.loadBalanced, conn.config.loadBalanced)
		})
	}
}

type testAuthenticator struct{}

var _ driver.Authenticator = &testAuthenticator{}

func (a *testAuthenticator) Auth(context.Context, *driver.AuthConfig) error {
	return fmt.Errorf("test error")
}

func (a *testAuthenticator) Reauth(context.Context, *driver.AuthConfig) error {
	return nil
}

func TestAuthenticateToAnything(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		set    func(*options.ClientOptions) error
		assert func(*testing.T, error)
	}{
		{
			name: "default",
			set:  func(*options.ClientOptions) error { return nil },
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "positive",
			set: func(opt *options.ClientOptions) error {
				return xoptions.SetInternalClientOptions(opt, "authenticateToAnything", true)
			},
			assert: func(t *testing.T, err error) {
				require.EqualError(t, err, "auth error: test error")
			},
		},
		{
			name: "negative",
			set: func(opt *options.ClientOptions) error {
				return xoptions.SetInternalClientOptions(opt, "authenticateToAnything", false)
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}

	describer := &drivertest.ChannelConn{
		Desc: description.Server{Kind: description.ServerKindRSArbiter},
	}
	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opt := options.Client().SetAuth(options.Credential{Username: "foo", Password: "bar"})
			err := tc.set(opt)
			require.NoError(t, err, "error setting authenticateToAnything: %v", err)
			cfg, err := NewConfigFromOptionsWithAuthenticator(opt, nil, &testAuthenticator{})
			require.NoError(t, err, "error constructing topology config: %v", err)

			srvrCfg := newServerConfig(defaultConnectionTimeout, cfg.ServerOpts...)
			connCfg := newConnectionConfig(srvrCfg.connectionOpts...)
			err = connCfg.handshaker.FinishHandshake(context.TODO(), &mnet.Connection{Describer: describer})
			tc.assert(t, err)
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

// Test that convertOIDCArgs exhaustively copies all fields of a driver.OIDCArgs
// into an options.OIDCArgs.
func TestConvertOIDCArgs(t *testing.T) {
	t.Parallel()
	refreshToken := "test refresh token"

	testCases := []struct {
		desc string
		args *driver.OIDCArgs
	}{
		{
			desc: "populated args",
			args: &driver.OIDCArgs{
				Version: 9,
				IDPInfo: &driver.IDPInfo{
					Issuer:        "test issuer",
					ClientID:      "test client ID",
					RequestScopes: []string{"test scope 1", "test scope 2"},
				},
				RefreshToken: &refreshToken,
			},
		},
		{
			desc: "nil",
			args: nil,
		},
		{
			desc: "nil IDPInfo and RefreshToken",
			args: &driver.OIDCArgs{
				Version:      9,
				IDPInfo:      nil,
				RefreshToken: nil,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			got := convertOIDCArgs(tc.args)

			if tc.args == nil {
				assert.Nil(t, got, "expected nil when input is nil")
				return
			}

			require.Equal(t,
				3,
				reflect.ValueOf(*tc.args).NumField(),
				"expected the driver.OIDCArgs struct to have exactly 3 fields")
			require.Equal(t,
				3,
				reflect.ValueOf(*got).NumField(),
				"expected the options.OIDCArgs struct to have exactly 3 fields")

			assert.Equal(t,
				tc.args.Version,
				got.Version,
				"expected Version field to be equal")
			assert.EqualValues(t,
				tc.args.IDPInfo,
				got.IDPInfo,
				"expected IDPInfo field to be convertible to equal values")
			assert.Equal(t,
				tc.args.RefreshToken,
				got.RefreshToken,
				"expected RefreshToken field to be equal")
		})
	}
}
