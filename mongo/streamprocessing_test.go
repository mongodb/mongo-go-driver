// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"crypto/tls"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestApplyStreamProcessingDefaults_EnablesTLS(t *testing.T) {
	opts := options.Client()
	applyStreamProcessingDefaults(opts)
	require.NotNil(t, opts.TLSConfig)
}

func TestApplyStreamProcessingDefaults_PreservesExistingTLS(t *testing.T) {
	cfg := &tls.Config{ServerName: "example"}
	opts := options.Client().SetTLSConfig(cfg)
	applyStreamProcessingDefaults(opts)
	require.NotNil(t, opts.TLSConfig)
	assert.Equal(t, "example", opts.TLSConfig.ServerName)
}

func TestApplyStreamProcessingDefaults_DefaultsAuthSourceToAdmin(t *testing.T) {
	opts := options.Client().SetAuth(options.Credential{Username: "u", Password: "p"})
	applyStreamProcessingDefaults(opts)
	require.NotNil(t, opts.Auth)
	assert.Equal(t, "admin", opts.Auth.AuthSource)
}

func TestApplyStreamProcessingDefaults_PreservesExplicitAuthSource(t *testing.T) {
	opts := options.Client().SetAuth(options.Credential{Username: "u", Password: "p", AuthSource: "elsewhere"})
	applyStreamProcessingDefaults(opts)
	require.NotNil(t, opts.Auth)
	assert.Equal(t, "elsewhere", opts.Auth.AuthSource)
}

func TestApplyStreamProcessingDefaults_NoAuthLeavesAuthNil(t *testing.T) {
	opts := options.Client()
	applyStreamProcessingDefaults(opts)
	assert.Nil(t, opts.Auth)
}

func TestIsStreamProcessingHost(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{"atlas-stream-699c842ef433fe6001480b17-etif1.virginia-usa.a.query.mongodb.net", true},
		{"atlas-stream-x.us-west.a.query.mongodb.net", true},
		{"cluster0.mongodb.net", false},
		{"localhost", false},
		{"atlas-stream-not-the-right-suffix.mongodb.net", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.host, func(t *testing.T) {
			assert.Equal(t, tc.want, IsStreamProcessingHost(tc.host))
		})
	}
}
