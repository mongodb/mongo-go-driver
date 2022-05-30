// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
)

func TestOptionsSetting(t *testing.T) {
	conf := &config{}
	ssts := time.Minute
	name := "test"
	assert.Zero(t, conf.cs)

	opt := WithConnString(func(connstring.ConnString) connstring.ConnString {
		return connstring.ConnString{
			ServerSelectionTimeout:    ssts,
			ServerSelectionTimeoutSet: true,
			AppName:                   name,
		}

	})

	assert.NoError(t, opt(conf))

	assert.Equal(t, ssts, conf.serverSelectionTimeout)

	serverConf := newServerConfig(conf.serverOpts...)
	assert.Equal(t, name, serverConf.appname, "expected appname to be: %v, got: %v", name, serverConf.appname)
}

func TestDirectConnectionFromConnString(t *testing.T) {
	singleConnect := connstring.ConnString{
		Connect:    connstring.SingleConnect,
		ConnectSet: true,
	}
	autoConnect := connstring.ConnString{
		Connect:    connstring.AutoConnect,
		ConnectSet: true,
	}
	directConnectionTrue := connstring.ConnString{
		DirectConnection:    true,
		DirectConnectionSet: true,
	}
	directConnectionFalse := connstring.ConnString{
		DirectConnection:    false,
		DirectConnectionSet: true,
	}
	defaultConnString := connstring.ConnString{}

	testCases := []struct {
		name string
		cs   connstring.ConnString
		mode MonitorMode
	}{
		{"connect=direct", singleConnect, SingleMode},
		{"connect=automatic", autoConnect, AutomaticMode},
		{"directConnection=true", directConnectionTrue, SingleMode},
		{"directConnection=false", directConnectionFalse, AutomaticMode},
		{"default", defaultConnString, AutomaticMode},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			topo, err := New(WithConnString(func(connstring.ConnString) connstring.ConnString { return tc.cs }))
			assert.Nil(t, err, "topology.New error: %v", err)
			assert.Equal(t, tc.mode, topo.cfg.mode, "expected mode %v, got %v", tc.mode, topo.cfg.mode)
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
			cs, err := connstring.ParseAndValidate(uri)
			assert.Nil(t, err, "connstring.ParseAndValidate error: %v", err)

			topo, err := New(WithConnString(func(connstring.ConnString) connstring.ConnString { return cs }))
			assert.Nil(t, err, "topology.New error: %v", err)
			assert.Equal(t, tc.loadBalanced, topo.cfg.loadBalanced, "expected loadBalanced %v, got %v", tc.loadBalanced, topo.cfg.loadBalanced)

			srvr := NewServer("", topo.id, topo.cfg.serverOpts...)
			assert.Equal(t, tc.loadBalanced, srvr.cfg.loadBalanced, "expected loadBalanced %v, got %v", tc.loadBalanced, srvr.cfg.loadBalanced)

			conn := newConnection("", srvr.cfg.connectionOpts...)
			assert.Equal(t, tc.loadBalanced, conn.config.loadBalanced, "expected loadBalanced %v, got %v", tc.loadBalanced, conn.config.loadBalanced)
		})
	}
}
