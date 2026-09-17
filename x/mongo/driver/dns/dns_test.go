// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dns

import (
	"net"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/require"
)

func TestNormalizeAllowedHostsSuffix(t *testing.T) {
	successCases := []struct {
		name  string
		value string
		want  string
	}{
		{"leading and trailing dot stripped", ".mydomain.net.", ".mydomain.net"},
		{"uppercase folded to lowercase", ".BUILD.10GEN.CC", ".build.10gen.cc"},
		{"leading dot prepended when absent", "10gen.cc", ".10gen.cc"},
		{"not a public suffix even though its parent is", "10gen.cc", ".10gen.cc"},
		{"underscore is accepted", ".my_domain.net", ".my_domain.net"},
		{"reserved single label", "localhost", ".localhost"},
		{"reserved single label with dots", ".test.", ".test"},
		{"reserved single label is case-folded", "LOCAL", ".local"},
		{"reserved single label not officially reserved", "corp", ".corp"},
	}
	for _, c := range successCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got, err := NormalizeAllowedHostsSuffix(c.value)
			require.NoError(t, err)
			require.Equal(t, c.want, got)
		})
	}

	errorCases := []struct {
		name  string
		value string
	}{
		{"empty after stripping dots", "."},
		{"single label", "net"},
		{"single label uppercase", "COM"},
		{"single label with leading dot", ".cc"},
		{"a public suffix", "cc"},
		{"a public suffix, case-insensitive", "CC"},
		{"a multi-label public suffix", "co.uk"},
	}
	for _, c := range errorCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			_, err := NormalizeAllowedHostsSuffix(c.value)
			require.Error(t, err)
		})
	}

	t.Run("normalization is idempotent", func(t *testing.T) {
		once, err := NormalizeAllowedHostsSuffix("mydomain.net")
		require.NoError(t, err)

		twice, err := NormalizeAllowedHostsSuffix(once)
		require.NoError(t, err)
		require.Equal(t, once, twice)
	})
}

func TestPublicSuffixProse(t *testing.T) {
	t.Run("1. A multi-label ordinary rule", func(t *testing.T) {
		require.True(t, isPublicSuffix("com.ac"))
		require.False(t, isPublicSuffix("foo.com.ac"))
	})
	t.Run("2. A shorter wildcard rule", func(t *testing.T) {
		require.True(t, isPublicSuffix("b.ck"))
		require.False(t, isPublicSuffix("a.b.ck"))
	})
	t.Run("3. A longer wildcard rule", func(t *testing.T) {
		require.True(t, isPublicSuffix("abc.nom.br"))
		require.False(t, isPublicSuffix("x.abc.nom.br"))
	})
	t.Run("4. An exception rule", func(t *testing.T) {
		require.True(t, isPublicSuffix("ck"))
		require.False(t, isPublicSuffix("www.ck"))
	})
	t.Run("5. No rule matches", func(t *testing.T) {
		require.True(t, isPublicSuffix("nosuchtld"))
		require.False(t, isPublicSuffix("foo.nosuchtld"))
	})
	t.Run("6. An internationalized rule", func(t *testing.T) {
		require.True(t, isPublicSuffix("xn--p1ai"))
		require.False(t, isPublicSuffix("example.xn--p1ai"))
		require.True(t, isPublicSuffix("xn--55qx5d.cn"))
	})
}

// TestParseHostsNormalizesReturnedHostnames pins the requirement that the seedlist is populated
// with normalized hostnames rather than the hostnames exactly as returned by DNS.
func TestParseHostsNormalizesReturnedHostnames(t *testing.T) {
	newResolver := func(target string) *Resolver {
		return &Resolver{
			LookupSRV: func(_, _, _ string) (string, []*net.SRV, error) {
				return "", []*net.SRV{{Target: target, Port: 27017}}, nil
			},
			LookupTXT: func(string) ([]string, error) { return nil, nil },
		}
	}

	t.Run("srvAllowedHostsSuffix", func(t *testing.T) {
		hosts, err := newResolver("CLUSTER.MONGODB.COM.").
			ParseHostsWithOptions("blogs.mongodb.com", ParseHostsOptions{
				AllowedHostsSuffix: ".mongodb.com",
				StopOnErr:          true,
			})
		require.NoError(t, err)
		require.Equal(t, []string{"cluster.mongodb.com:27017"}, hosts)
	})

	t.Run("default domain verification", func(t *testing.T) {
		hosts, err := newResolver("Cluster.Mongodb.COM.").
			ParseHostsWithOptions("blogs.mongodb.com", ParseHostsOptions{StopOnErr: true})
		require.NoError(t, err)
		require.Equal(t, []string{"cluster.mongodb.com:27017"}, hosts)
	})
}

// TestValidateSRVResultNormalizesQueryHostname tests that a Unicode query hostname is
// equivalent to its A-label form.
func TestValidateSRVResultNormalizesQueryHostname(t *testing.T) {
	require.NoError(t, validateSRVResult("host.b.xn--p1ai", "a.b.рф", ""))
	require.NoError(t, validateSRVResult("host.b.xn--p1ai", "a.b.xn--p1ai", ""))
	require.NoError(t, validateSRVResult("host.b.xn--p1ai", "a.b.xn--p1ai.", ""))
}
