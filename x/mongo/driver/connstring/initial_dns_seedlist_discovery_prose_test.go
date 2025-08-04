// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package connstring

import (
	"fmt"
	"net"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/dns"
)

func TestInitialDNSSeedlistDiscoveryProse(t *testing.T) {
	newTestParser := func(record string) *parser {
		return &parser{&dns.Resolver{
			LookupSRV: func(_, _, _ string) (string, []*net.SRV, error) {
				return "", []*net.SRV{
					{
						Target: record,
						Port:   27017,
					},
				}, nil
			},
			LookupTXT: func(string) ([]string, error) {
				return nil, nil
			},
		}}
	}

	t.Run("1. Allow SRVs with fewer than 3 . separated parts", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			record string
			uri    string
		}{
			{"test_1.localhost", "mongodb+srv://localhost"},
			{"test_1.mongo.local", "mongodb+srv://mongo.local"},
		}
		for _, c := range cases {
			c := c
			t.Run(c.uri, func(t *testing.T) {
				t.Parallel()

				_, err := newTestParser(c.record).parse(c.uri)
				assert.NoError(t, err, "expected no URI parsing error, got %v", err)
			})
		}
	})
	t.Run("2. Throw when return address does not end with SRV domain", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			record string
			uri    string
		}{
			{"localhost.mongodb", "mongodb+srv://localhost"},
			{"test_1.evil.local", "mongodb+srv://mongo.local"},
			{"blogs.evil.com", "mongodb+srv://blogs.mongodb.com"},
		}
		for _, c := range cases {
			c := c
			t.Run(c.uri, func(t *testing.T) {
				t.Parallel()

				_, err := newTestParser(c.record).parse(c.uri)
				assert.ErrorContains(t, err, "Domain suffix from SRV record not matched input domain")
			})
		}
	})
	t.Run("3. Throw when return address is identical to SRV hostname", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			record string
			uri    string
			labels int
		}{
			{"localhost", "mongodb+srv://localhost", 1},
			{"mongo.local", "mongodb+srv://mongo.local", 2},
		}
		for _, c := range cases {
			c := c
			t.Run(c.uri, func(t *testing.T) {
				t.Parallel()

				_, err := newTestParser(c.record).parse(c.uri)
				expected := fmt.Sprintf(
					"Server record (%d levels) should have more domain levels than parent URI (%d levels)",
					c.labels, c.labels,
				)
				assert.ErrorContains(t, err, expected)
			})
		}
	})
	t.Run("4. Throw when return address does not contain . separating shared part of domain", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			record string
			uri    string
		}{
			{"test_1.cluster_1localhost", "mongodb+srv://localhost"},
			{"test_1.my_hostmongo.local", "mongodb+srv://mongo.local"},
			{"cluster.testmongodb.com", "mongodb+srv://blogs.mongodb.com"},
		}
		for _, c := range cases {
			c := c
			t.Run(c.uri, func(t *testing.T) {
				t.Parallel()

				_, err := newTestParser(c.record).parse(c.uri)
				assert.ErrorContains(t, err, "Domain suffix from SRV record not matched input domain")
			})
		}
	})
}
