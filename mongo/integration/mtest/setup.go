// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mtest

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"go.mongodb.org/mongo-driver/x/mongo/driver/ocsp"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

const (
	// TestDb specifies the name of default test database.
	TestDb = "test"
)

// testContext holds the global context for the integration tests. The testContext members should only be initialized
// once during the global setup in TestMain. These variables should only be accessed indirectly through MongoTest
// instances.
var testContext struct {
	connString connstring.ConnString
	topo       *topology.Topology
	topoKind   TopologyKind
	// shardedReplicaSet will be true if we're connected to a sharded cluster and each shard is backed by a replica set.
	// We track this as a separate boolean rather than setting topoKind to ShardedReplicaSet because a general
	// "Sharded" constraint in a test should match both Sharded and ShardedReplicaSet.
	shardedReplicaSet           bool
	client                      *mongo.Client // client used for setup and teardown
	serverVersion               string
	authEnabled                 bool
	sslEnabled                  bool
	enterpriseServer            bool
	dataLake                    bool
	requireAPIVersion           bool
	serverParameters            bson.Raw
	singleMongosLoadBalancerURI string
	multiMongosLoadBalancerURI  string
}

func setupClient(cs connstring.ConnString, opts *options.ClientOptions) (*mongo.Client, error) {
	wcMajority := writeconcern.New(writeconcern.WMajority())
	// set ServerAPIOptions to latest version if required
	if opts.ServerAPIOptions == nil && testContext.requireAPIVersion {
		opts.SetServerAPIOptions(options.ServerAPI(driver.TestServerAPIVersion))
	}
	// for sharded clusters, pin to one host. Due to how the cache is implemented on 4.0 and 4.2, behavior
	// can be inconsistent when multiple mongoses are used
	return mongo.Connect(Background, opts.ApplyURI(cs.Original).SetWriteConcern(wcMajority).SetHosts(cs.Hosts[:1]))
}

// Setup initializes the current testing context.
// This function must only be called one time and must be called before any tests run.
func Setup(setupOpts ...*SetupOptions) error {
	opts := MergeSetupOptions(setupOpts...)
	var err error

	switch {
	case opts.URI != nil:
		testContext.connString, err = connstring.ParseAndValidate(*opts.URI)
	default:
		testContext.connString, err = getClusterConnString()
	}
	if err != nil {
		return fmt.Errorf("error getting connection string: %v", err)
	}
	testContext.dataLake = os.Getenv("ATLAS_DATA_LAKE_INTEGRATION_TEST") == "true"
	testContext.requireAPIVersion = os.Getenv("REQUIRE_API_VERSION") == "true"

	connectionOpts := []topology.ConnectionOption{
		topology.WithOCSPCache(func(ocsp.Cache) ocsp.Cache {
			return ocsp.NewCache()
		}),
	}
	serverOpts := []topology.ServerOption{
		topology.WithConnectionOptions(func(opts ...topology.ConnectionOption) []topology.ConnectionOption {
			return append(opts, connectionOpts...)
		}),
	}
	if testContext.requireAPIVersion {
		serverOpts = append(serverOpts,
			topology.WithServerAPI(func(*driver.ServerAPIOptions) *driver.ServerAPIOptions {
				return driver.NewServerAPIOptions(driver.TestServerAPIVersion)
			}),
		)
	}

	testContext.topo, err = topology.New(
		topology.WithConnString(func(connstring.ConnString) connstring.ConnString {
			return testContext.connString
		}),
		topology.WithServerOptions(func(opts ...topology.ServerOption) []topology.ServerOption {
			return append(opts, serverOpts...)
		}),
	)
	if err != nil {
		return fmt.Errorf("error creating topology: %v", err)
	}
	if err = testContext.topo.Connect(); err != nil {
		return fmt.Errorf("error connecting topology: %v", err)
	}

	testContext.client, err = setupClient(testContext.connString, options.Client())
	if err != nil {
		return fmt.Errorf("error connecting test client: %v", err)
	}

	pingCtx, cancel := context.WithTimeout(Background, 2*time.Second)
	defer cancel()
	if err := testContext.client.Ping(pingCtx, readpref.Primary()); err != nil {
		return fmt.Errorf("ping error: %v; make sure the deployment is running on URI %v", err,
			testContext.connString.Original)
	}

	if testContext.serverVersion, err = getServerVersion(); err != nil {
		return fmt.Errorf("error getting server version: %v", err)
	}

	switch testContext.topo.Kind() {
	case description.Single:
		testContext.topoKind = Single
	case description.ReplicaSet, description.ReplicaSetWithPrimary, description.ReplicaSetNoPrimary:
		testContext.topoKind = ReplicaSet
	case description.Sharded:
		testContext.topoKind = Sharded
	case description.LoadBalanced:
		testContext.topoKind = LoadBalanced
	default:
		return fmt.Errorf("could not detect topology kind; current topology: %s", testContext.topo.String())
	}

	// If we're connected to a sharded cluster, determine if the cluster is backed by replica sets.
	if testContext.topoKind == Sharded {
		// Run a find against config.shards and get each document in the collection.
		cursor, err := testContext.client.Database("config").Collection("shards").Find(Background, bson.D{})
		if err != nil {
			return fmt.Errorf("error running find against config.shards: %v", err)
		}
		defer cursor.Close(Background)

		var shards []struct {
			Host string `bson:"host"`
		}
		if err := cursor.All(Background, &shards); err != nil {
			return fmt.Errorf("error getting results find against config.shards: %v", err)
		}

		// Each document's host field will contain a single hostname if the shard is a standalone. If it's a replica
		// set, the host field will be in the format "replicaSetName/host1,host2,...". Therefore, we can determine that
		// the shard is a standalone if the "/" character isn't present.
		var foundStandalone bool
		for _, shard := range shards {
			if strings.Index(shard.Host, "/") == -1 {
				foundStandalone = true
				break
			}
		}
		if !foundStandalone {
			testContext.shardedReplicaSet = true
		}
	}

	// For load balanced clusters, retrieve the required LB URIs and add additional information (e.g. TLS options) to
	// them if necessary.
	if testContext.topoKind == LoadBalanced {
		singleMongosURI := os.Getenv("SINGLE_MONGOS_LB_URI")
		if singleMongosURI == "" {
			return errors.New("SINGLE_MONGOS_LB_URI must be set when running against load balanced clusters")
		}
		testContext.singleMongosLoadBalancerURI = addNecessaryParamsToURI(singleMongosURI)

		multiMongosURI := os.Getenv("MULTI_MONGOS_LB_URI")
		if multiMongosURI == "" {
			return errors.New("MULTI_MONGOS_LB_URI must be set when running against load balanced clusters")
		}
		testContext.multiMongosLoadBalancerURI = addNecessaryParamsToURI(multiMongosURI)
	}

	testContext.authEnabled = os.Getenv("AUTH") == "auth"
	testContext.sslEnabled = os.Getenv("SSL") == "ssl"
	biRes, err := testContext.client.Database("admin").RunCommand(Background, bson.D{{"buildInfo", 1}}).DecodeBytes()
	if err != nil {
		return fmt.Errorf("buildInfo error: %v", err)
	}
	modulesRaw, err := biRes.LookupErr("modules")
	if err == nil {
		// older server versions don't report "modules" field in buildInfo result
		modules, _ := modulesRaw.Array().Values()
		for _, module := range modules {
			if module.StringValue() == "enterprise" {
				testContext.enterpriseServer = true
				break
			}
		}
	}

	// Get server parameters if test is not running against ADL; ADL does not have "getParameter" command.
	if !testContext.dataLake {
		db := testContext.client.Database("admin")
		testContext.serverParameters, err = db.RunCommand(Background, bson.D{{"getParameter", "*"}}).DecodeBytes()
		if err != nil {
			return fmt.Errorf("error getting serverParameters: %v", err)
		}
	}
	return nil
}

// Teardown cleans up resources initialized by Setup.
// This function must be called once after all tests have finished running.
func Teardown() error {
	// Dropping the test database causes an error against Atlas Data Lake.
	if !testContext.dataLake {
		if err := testContext.client.Database(TestDb).Drop(Background); err != nil {
			return fmt.Errorf("error dropping test database: %v", err)
		}
	}
	if err := testContext.client.Disconnect(Background); err != nil {
		return fmt.Errorf("error disconnecting test client: %v", err)
	}
	if err := testContext.topo.Disconnect(Background); err != nil {
		return fmt.Errorf("error disconnecting test topology: %v", err)
	}
	return nil
}

func getServerVersion() (string, error) {
	var serverStatus bson.Raw
	err := testContext.client.Database(TestDb).RunCommand(
		Background,
		bson.D{{"buildInfo", 1}},
	).Decode(&serverStatus)
	if err != nil {
		return "", err
	}

	version, err := serverStatus.LookupErr("version")
	if err != nil {
		return "", errors.New("no version string in serverStatus response")
	}

	return version.StringValue(), nil
}

// addOptions appends connection string options to a URI.
func addOptions(uri string, opts ...string) string {
	if !strings.ContainsRune(uri, '?') {
		if uri[len(uri)-1] != '/' {
			uri += "/"
		}

		uri += "?"
	} else {
		uri += "&"
	}

	for _, opt := range opts {
		uri += opt
	}

	return uri
}

// addTLSConfig checks for the environmental variable indicating that the tests are being run
// on an SSL-enabled server, and if so, returns a new URI with the necessary configuration.
func addTLSConfig(uri string) string {
	caFile := os.Getenv("MONGO_GO_DRIVER_CA_FILE")
	if len(caFile) == 0 {
		return uri
	}

	return addOptions(uri, "ssl=true&sslCertificateAuthorityFile=", caFile)
}

// addCompressors checks for the environment variable indicating that the tests are being run with compression
// enabled. If so, it returns a new URI with the necessary configuration
func addCompressors(uri string) string {
	comp := os.Getenv("MONGO_GO_DRIVER_COMPRESSOR")
	if len(comp) == 0 {
		return uri
	}

	return addOptions(uri, "compressors=", comp)
}

// getClusterConnString gets the globally configured connection string.
func getClusterConnString() (connstring.ConnString, error) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	uri = addNecessaryParamsToURI(uri)
	return connstring.ParseAndValidate(uri)
}

func addNecessaryParamsToURI(uri string) string {
	uri = addTLSConfig(uri)
	return addCompressors(uri)
}

// CompareServerVersions compares two version number strings (i.e. positive integers separated by
// periods). Comparisons are done to the lesser precision of the two versions. For example, 3.2 is
// considered equal to 3.2.11, whereas 3.2.0 is considered less than 3.2.11.
//
// Returns a positive int if version1 is greater than version2, a negative int if version1 is less
// than version2, and 0 if version1 is equal to version2.
func CompareServerVersions(v1 string, v2 string) int {
	n1 := strings.Split(v1, ".")
	n2 := strings.Split(v2, ".")

	for i := 0; i < int(math.Min(float64(len(n1)), float64(len(n2)))); i++ {
		i1, err := strconv.Atoi(n1[i])
		if err != nil {
			return 1
		}

		i2, err := strconv.Atoi(n2[i])
		if err != nil {
			return -1
		}

		difference := i1 - i2
		if difference != 0 {
			return difference
		}
	}

	return 0
}
