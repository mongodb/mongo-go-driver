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
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/x/mongo/driverlegacy/dns"
	"go.mongodb.org/mongo-driver/x/network/address"
	"go.mongodb.org/mongo-driver/x/network/connstring"
	"go.mongodb.org/mongo-driver/x/network/description"
)

type mockResolver struct {
	recordsToAdd    []*net.SRV
	recordsToRemove []*net.SRV
	lookupFail      bool
	lookupTimeout   bool
}

func (r *mockResolver) LookupSRV(service, proto, name string) (string, []*net.SRV, error) {
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

func (r *mockResolver) LookupTXT(name string) ([]string, error) { return nil, nil }

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

type serverSorter []description.Server

func (ss serverSorter) Len() int      { return len(ss) }
func (ss serverSorter) Swap(i, j int) { ss[i], ss[j] = ss[j], ss[i] }
func (ss serverSorter) Less(i, j int) bool {
	return strings.Compare(ss[i].Addr.String(), ss[j].Addr.String()) < 0
}

func compareHosts(t *testing.T, received []description.Server, expected []string) {
	if len(received) != len(expected) {
		t.Fatalf("Number of hosts in topology does not match expected value. Got %v; want %v.", len(received), len(expected))
	}

	actual := serverSorter(received)
	sort.Sort(actual)
	sort.Strings(expected)

	for i := range received {
		if received[i].Addr.String() != expected[i] {
			t.Errorf("Hosts in topology differ from expected values. Got %v; want %v.",
				received[i].Addr.String(), expected[i])
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
			topo.dnsResolver = dns.Resolver{mockRes.LookupSRV, mockRes.LookupTXT}
			topo.rescanSRVInterval = time.Second
			err = topo.Connect(context.Background())
			if err != nil {
				t.Fatalf("Could not connect to the topology: %v", err)
			}

			// wait for description to update
			sub, err := topo.Subscribe()
			if err != nil {
				t.Fatalf("Couldn't subscribe: %v", err)
			}
			for i := 0; i < 4; i++ {
				<-sub.C
			}
			actualHosts := topo.Description().Servers
			compareHosts(t, actualHosts, tt.expectedHosts)
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
