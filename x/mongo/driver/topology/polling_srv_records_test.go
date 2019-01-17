// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"net"
	"sort"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/x/mongo/driver/dns"
	"go.mongodb.org/mongo-driver/x/network/address"
	"go.mongodb.org/mongo-driver/x/network/connstring"
)

type mockResolver struct {
	recordsToAdd    []*net.SRV
	recordsToRemove []*net.SRV
	lookupFail      bool
	lookupTimeout   bool
}

func (r *mockResolver) LookupSRV(ctx context.Context, service, proto, name string) (string, []*net.SRV, error) {
	if r.lookupFail {
		return "", nil, &net.DNSError{}
	}
	if r.lookupTimeout {
		return "", nil, &net.DNSError{IsTimeout: true}
	}
	str, addresses, err := net.LookupSRV("mongodb", "tcp", name)
	if err != nil {
		return str, addresses, err
	}

	// Add/remove records to mimic changing the DNS records.
	if r.recordsToAdd != nil {
		addresses = append(addresses, r.recordsToAdd...)
	}
	if r.recordsToRemove != nil {
		for _, removeAddr := range r.recordsToRemove {
			for j, addr := range addresses {
				if removeAddr.Target == addr.Target && removeAddr.Port == addr.Port {
					addresses = append(addresses[:j], addresses[j+1:]...)
				}
			}
		}
	}
	return str, addresses, err
}

func (r *mockResolver) LookupTXT(ctx context.Context, name string) ([]string, error) { return nil, nil }

var srvPollingTests = []struct {
	name            string
	recordsToAdd    []*net.SRV
	recordsToRemove []*net.SRV
	lookupFail      bool
	lookupTimeout   bool
	expectedHosts   []string
}{
	{"Add new record", []*net.SRV{{"localhost.test.build.10gen.cc.", 27019, 0, 0}}, nil, false, false, []string{"localhost.test.build.10gen.cc:27017", "localhost.test.build.10gen.cc:27018", "localhost.test.build.10gen.cc:27019"}},
	{"Remove existing record", nil, []*net.SRV{{"localhost.test.build.10gen.cc.", 27018, 0, 0}}, false, false, []string{"localhost.test.build.10gen.cc:27017"}},
	{"Replace existing record", []*net.SRV{{"localhost.test.build.10gen.cc.", 27019, 0, 0}}, []*net.SRV{{"localhost.test.build.10gen.cc.", 27018, 0, 0}}, false, false, []string{"localhost.test.build.10gen.cc:27017", "localhost.test.build.10gen.cc:27019"}},
	{"Replace both with one new", []*net.SRV{{"localhost.test.build.10gen.cc.", 27019, 0, 0}}, []*net.SRV{{"localhost.test.build.10gen.cc.", 27017, 0, 0}, {"localhost.test.build.10gen.cc.", 27018, 0, 0}}, false, false, []string{"localhost.test.build.10gen.cc:27019"}},
	{"Replace both with two new", []*net.SRV{{"localhost.test.build.10gen.cc.", 27019, 0, 0}, {"localhost.test.build.10gen.cc.", 27020, 0, 0}}, []*net.SRV{{"localhost.test.build.10gen.cc.", 27017, 0, 0}, {"localhost.test.build.10gen.cc.", 27018, 0, 0}}, false, false, []string{"localhost.test.build.10gen.cc:27019", "localhost.test.build.10gen.cc:27020"}},
	{"DNS lookup timeout", nil, nil, false, true, []string{"localhost.test.build.10gen.cc:27017", "localhost.test.build.10gen.cc:27018"}},
	{"DNS lookup failure", nil, nil, true, false, []string{"localhost.test.build.10gen.cc:27017", "localhost.test.build.10gen.cc:27018"}},
	{"Remove all", nil, []*net.SRV{{"localhost.test.build.10gen.cc.", 27017, 0, 0}, {"localhost.test.build.10gen.cc.", 27018, 0, 0}}, false, false, []string{"localhost.test.build.10gen.cc:27017", "localhost.test.build.10gen.cc:27018"}},
}

func compareHostLists(received []string, expected []string, t *testing.T) {
	if len(received) != len(expected) {
		t.Fatalf("Number of elems in t.cfg.seedList does not match expected value. Got %v; want %v.", len(received), len(expected))
	}

	sort.Strings(received)
	sort.Strings(expected)

	for i := range received {
		if received[i] != expected[i] {
			t.Errorf("Hosts in t.cfg.seedList differ from expected values. Got %v; want %v.",
				received[i], expected[i])
		}
	}
}

func TestPollingSRVRecordsSpec(t *testing.T) {
	for _, tt := range srvPollingTests {
		t.Run(tt.name, func(t *testing.T) {
			cs, err := connstring.Parse("mongodb+srv://test1.test.build.10gen.cc")
			if err != nil {
				t.Fatalf("Problem parsing the uri: %v", err)
			}
			topo, err := New(WithConnString(func(connstring.ConnString) connstring.ConnString { return cs }))
			if err != nil {
				t.Fatalf("Could not create the topology: %v", err)
			}
			mockRes := mockResolver{tt.recordsToAdd, tt.recordsToRemove, tt.lookupFail, tt.lookupTimeout}
			topo.DNSResolver = dns.Resolver{&mockRes}
			topo.rescanSRVInterval = time.Second
			err = topo.Connect(context.Background())
			if err != nil {
				t.Fatalf("Could not connect to the topology: %v", err)
			}

			time.Sleep(2 * time.Second)
			compareHostLists(topo.cfg.seedList, tt.expectedHosts, t)
			for _, e := range tt.expectedHosts {
				addr := address.Address(e).Canonicalize()
				if _, ok := topo.servers[addr]; !ok {
					t.Errorf("Topology server list did not contain expected value %v", e)
				}
			}
			_ = topo.Disconnect(context.Background())
		})
	}
}
