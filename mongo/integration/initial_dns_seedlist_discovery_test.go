// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

const (
	seedlistDiscoveryTestsBaseDir = "../../data/initial-dns-seedlist-discovery"
)

type seedlistTest struct {
	URI      string   `bson:"uri"`
	Seeds    []string `bson:"seeds"`
	NumSeeds *int     `bson:"numSeeds"`
	Hosts    []string `bson:"hosts"`
	NumHosts *int     `bson:"numHosts"`
	Error    bool     `bson:"error"`
	Options  bson.Raw `bson:"options"`
}

func TestInitialDNSSeedlistDiscoverySpec(t *testing.T) {
	mt := mtest.New(t, noClientOpts)
	defer mt.Close()

	mt.RunOpts("replica set", mtest.NewOptions().Topologies(mtest.ReplicaSet).CreateClient(false), func(mt *mtest.T) {
		runSeedlistDiscoveryDirectory(mt, "replica-set")
	})
	mt.RunOpts("sharded", mtest.NewOptions().Topologies(mtest.Sharded).CreateClient(false), func(mt *mtest.T) {
		runSeedlistDiscoveryDirectory(mt, "sharded")
	})
	mt.RunOpts("load balanced", mtest.NewOptions().Topologies(mtest.LoadBalanced).CreateClient(false), func(mt *mtest.T) {
		runSeedlistDiscoveryDirectory(mt, "load-balanced")
	})
}

func runSeedlistDiscoveryDirectory(mt *mtest.T, subdirectory string) {
	directoryPath := path.Join(seedlistDiscoveryTestsBaseDir, subdirectory)
	for _, file := range jsonFilesInDir(mt, directoryPath) {
		mt.RunOpts(file, noClientOpts, func(mt *mtest.T) {
			runSeedlistDiscoveryTest(mt, path.Join(directoryPath, file))
		})
	}
}

func runSeedlistDiscoveryTest(mt *mtest.T, file string) {
	content, err := ioutil.ReadFile(file)
	assert.Nil(mt, err, "ReadFile error for %v: %v", file, err)

	var test seedlistTest
	err = bson.UnmarshalExtJSONWithRegistry(specTestRegistry, content, false, &test)
	assert.Nil(mt, err, "UnmarshalExtJSONWithRegistry error: %v", err)

	if runtime.GOOS == "windows" && strings.HasSuffix(file, "/two-txt-records.json") {
		mt.Skip("skipping to avoid windows multiple TXT record lookup bug")
	}
	if strings.HasPrefix(runtime.Version(), "go1.11") && strings.HasSuffix(file, "/one-txt-record-multiple-strings.json") {
		mt.Skip("skipping to avoid go1.11 problem with multiple strings in one TXT record")
	}

	cs, err := connstring.ParseAndValidate(test.URI)
	if test.Error {
		assert.NotNil(mt, err, "expected URI parsing error, got nil")
		return
	}

	assert.Nil(mt, err, "ParseAndValidate error: %v", err)
	assert.Equal(mt, connstring.SchemeMongoDBSRV, cs.Scheme,
		"expected scheme %v, got %v", connstring.SchemeMongoDBSRV, cs.Scheme)

	// DNS records may be out of order from the test file's ordering.
	actualSeedlist := buildSet(cs.Hosts)
	// If NumSeeds is set, check number of seeds in seedlist.
	if test.NumSeeds != nil {
		assert.Equal(mt, len(actualSeedlist), *test.NumSeeds,
			"expected %v seeds, got %v", *test.NumSeeds, len(actualSeedlist))
	}
	// If Seeds is set, check contents of seedlist.
	if test.Seeds != nil {
		expectedSeedlist := buildSet(test.Seeds)
		assert.Equal(mt, expectedSeedlist, actualSeedlist, "expected seedlist %v, got %v", expectedSeedlist, actualSeedlist)
	}
	verifyConnstringOptions(mt, test.Options, cs)
	setSSLSettings(mt, &cs, test)

	// Make a topology from the options.
	topo, err := topology.New(topology.WithConnString(func(connstring.ConnString) connstring.ConnString { return cs }))
	assert.Nil(mt, err, "topology.New error: %v", err)
	err = topo.Connect()
	assert.Nil(mt, err, "topology.Connect error: %v", err)
	defer func() { _ = topo.Disconnect(context.Background()) }()

	// If NumHosts is set, check number of hosts currently stored on the Topology.
	if test.NumHosts != nil {
		actualNumHosts := len(topo.Description().Servers)
		assert.Equal(mt, *test.NumHosts, actualNumHosts, "expected to find %v hosts, found %v",
			*test.NumHosts, actualNumHosts)
	}
	for _, host := range test.Hosts {
		_, err := getServerByAddress(host, topo)
		assert.Nil(mt, err, "error finding host %q: %v", host, err)
	}
}

func buildSet(list []string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, s := range list {
		set[s] = struct{}{}
	}
	return set
}

func verifyConnstringOptions(mt *mtest.T, expected bson.Raw, cs connstring.ConnString) {
	mt.Helper()

	elems, _ := expected.Elements()
	for _, elem := range elems {
		key := elem.Key()
		opt := elem.Value()

		switch key {
		case "replicaSet":
			rs := opt.StringValue()
			assert.Equal(mt, rs, cs.ReplicaSet, "expected replicaSet value %v, got %v", rs, cs.ReplicaSet)
		case "ssl":
			ssl := opt.Boolean()
			assert.Equal(mt, ssl, cs.SSL, "expected ssl value %v, got %v", ssl, cs.SSL)
		case "authSource":
			source := opt.StringValue()
			assert.Equal(mt, source, cs.AuthSource, "expected auth source value %v, got %v", source, cs.AuthSource)
		case "directConnection":
			dc := opt.Boolean()
			assert.True(mt, cs.DirectConnectionSet, "expected cs.DirectConnectionSet to be true, got false")
			assert.Equal(mt, dc, cs.DirectConnection, "expected cs.DirectConnection to be %v, got %v", dc, cs.DirectConnection)
		case "loadBalanced":
			lb := opt.Boolean()
			assert.True(mt, cs.LoadBalancedSet, "expected cs.LoadBalancedSet set to be true, got false")
			assert.Equal(mt, lb, cs.LoadBalanced, "expected cs.LoadBalanced to be %v, got %v", lb, cs.LoadBalanced)
		case "srvMaxHosts":
			srvMaxHosts := opt.Int32()
			assert.Equal(mt, srvMaxHosts, int32(cs.SRVMaxHosts), "expected cs.SRVMaxHosts to be %v, got %v", srvMaxHosts, cs.SRVMaxHosts)
		case "srvServiceName":
			srvName := opt.StringValue()
			assert.Equal(mt, srvName, cs.SRVServiceName, "expected cs.SRVServiceName to be %q, got %q", srvName, cs.SRVServiceName)
		default:
			mt.Fatalf("unrecognized connstring option %v", key)
		}
	}
}

// Because the Go driver tests can be run either against a server with SSL enabled or without, a
// number of configurations have to be checked to ensure that the SRV tests are run properly.
//
// First, the "ssl" option in the JSON test description has to be checked. If this option is not
// present, we assume that the test will assert an error, so we proceed with the test as normal.
// If the option is false, then we skip the test if the server is running with SSL enabled.
// If the option is true, then we skip the test if the server is running without SSL enabled; if
// the server is running with SSL enabled, then we manually set the necessary SSL options in the
// connection string.
func setSSLSettings(mt *mtest.T, cs *connstring.ConnString, test seedlistTest) {
	ssl, err := test.Options.LookupErr("ssl")
	if err != nil {
		// No "ssl" option is specified
		return
	}
	testCaseExpectsSSL := ssl.Boolean()
	envSSL := os.Getenv("SSL") == "ssl"

	// Skip non-SSL tests if the server is running with SSL.
	if !testCaseExpectsSSL && envSSL {
		mt.Skip("skipping test that does not expect ssl in an ssl environment")
	}

	// Skip SSL tests if the server is running without SSL.
	if testCaseExpectsSSL && !envSSL {
		mt.Skip("skipping test that expects ssl in a non-ssl environment")
	}

	// If SSL tests are running, set the CA file.
	if testCaseExpectsSSL && envSSL {
		cs.SSLInsecure = true
	}
}

func getServerByAddress(address string, topo *topology.Topology) (description.Server, error) {
	selectByName := description.ServerSelectorFunc(func(_ description.Topology, servers []description.Server) ([]description.Server, error) {
		for _, s := range servers {
			if s.Addr.String() == address {
				return []description.Server{s}, nil
			}
		}
		return []description.Server{}, nil
	})

	selectedServer, err := topo.SelectServer(context.Background(), selectByName)
	if err != nil {
		return description.Server{}, err
	}
	selectedServerConnection, err := selectedServer.Connection(context.Background())
	if err != nil {
		return description.Server{}, err
	}
	defer selectedServerConnection.Close()
	return selectedServerConnection.Description(), nil
}
