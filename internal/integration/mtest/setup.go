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

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/integtest"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/topology"
)

const (
	// TestDb specifies the name of default test database.
	TestDb = "test"
)

// testContext holds the global context for the integration tests. The testContext members should only be initialized
// once during the global setup in TestMain. These variables should only be accessed indirectly through MongoTest
// instances.
var testContext struct {
	connString *connstring.ConnString
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
	serverless                  bool
}

func setupClient(opts *options.ClientOptions) (*mongo.Client, error) {
	wcMajority := writeconcern.Majority()
	// set ServerAPIOptions to latest version if required
	if opts.ServerAPIOptions == nil && testContext.requireAPIVersion {
		opts.SetServerAPIOptions(options.ServerAPI(driver.TestServerAPIVersion))
	}
	// for sharded clusters, pin to one host. Due to how the cache is implemented on 4.0 and 4.2, behavior
	// can be inconsistent when multiple mongoses are used
	return mongo.Connect(opts.SetWriteConcern(wcMajority).SetHosts(opts.Hosts[:1]))
}

// Setup initializes the current testing context.
// This function must only be called one time and must be called before any tests run.
func Setup(setupOpts ...*SetupOptions) error {
	opts := NewSetupOptions()
	for _, opt := range setupOpts {
		if opt == nil {
			continue
		}
		if opt.URI != nil {
			opts.URI = opt.URI
		}
	}

	var uri string
	var err error

	switch {
	case opts.URI != nil:
		uri = *opts.URI
	default:
		var err error
		uri, err = integtest.MongoDBURI()
		if err != nil {
			return fmt.Errorf("error getting uri: %w", err)
		}
	}

	testContext.connString, err = connstring.ParseAndValidate(uri)
	if err != nil {
		return fmt.Errorf("error parsing and validating connstring: %w", err)
	}

	testContext.dataLake = os.Getenv("ATLAS_DATA_LAKE_INTEGRATION_TEST") == "true"
	testContext.requireAPIVersion = os.Getenv("REQUIRE_API_VERSION") == "true"

	clientOpts := options.Client().ApplyURI(uri)
	integtest.AddTestServerAPIVersion(clientOpts)

	cfg, err := topology.NewConfig(clientOpts, nil)
	if err != nil {
		return fmt.Errorf("error constructing topology config: %w", err)
	}

	testContext.topo, err = topology.New(cfg)
	if err != nil {
		return fmt.Errorf("error creating topology: %w", err)
	}
	if err = testContext.topo.Connect(); err != nil {
		return fmt.Errorf("error connecting topology: %w", err)
	}

	testContext.client, err = setupClient(options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("error connecting test client: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := testContext.client.Ping(pingCtx, readpref.Primary()); err != nil {
		return fmt.Errorf("ping error: %w; make sure the deployment is running on URI %v", err,
			testContext.connString.Original)
	}

	if testContext.serverVersion, err = getServerVersion(); err != nil {
		return fmt.Errorf("error getting server version: %w", err)
	}

	switch testContext.topo.Kind() {
	case description.TopologyKindSingle:
		testContext.topoKind = Single
	case description.TopologyKindReplicaSet, description.TopologyKindReplicaSetWithPrimary, description.TopologyKindReplicaSetNoPrimary:
		testContext.topoKind = ReplicaSet
	case description.TopologyKindSharded:
		testContext.topoKind = Sharded
	case description.TopologyKindLoadBalanced:
		testContext.topoKind = LoadBalanced
	default:
		return fmt.Errorf("could not detect topology kind; current topology: %s", testContext.topo.String())
	}

	// If we're connected to a sharded cluster, determine if the cluster is backed by replica sets.
	if testContext.topoKind == Sharded {
		// Run a find against config.shards and get each document in the collection.
		cursor, err := testContext.client.Database("config").Collection("shards").Find(context.Background(), bson.D{})
		if err != nil {
			return fmt.Errorf("error running find against config.shards: %w", err)
		}
		defer cursor.Close(context.Background())

		var shards []struct {
			Host string `bson:"host"`
		}
		if err := cursor.All(context.Background(), &shards); err != nil {
			return fmt.Errorf("error getting results find against config.shards: %w", err)
		}

		// Each document's host field will contain a single hostname if the shard is a standalone. If it's a replica
		// set, the host field will be in the format "replicaSetName/host1,host2,...". Therefore, we can determine that
		// the shard is a standalone if the "/" character isn't present.
		var foundStandalone bool
		for _, shard := range shards {
			if !strings.Contains(shard.Host, "/") {
				foundStandalone = true
				break
			}
		}
		if !foundStandalone {
			testContext.shardedReplicaSet = true
		}
	}

	// For non-serverless, load balanced clusters, retrieve the required LB URIs and add additional information (e.g. TLS options) to
	// them if necessary.
	testContext.serverless = os.Getenv("SERVERLESS") == "serverless"
	if !testContext.serverless && testContext.topoKind == LoadBalanced {
		singleMongosURI := os.Getenv("SINGLE_MONGOS_LB_URI")
		if singleMongosURI == "" {
			return errors.New("SINGLE_MONGOS_LB_URI must be set when running against load balanced clusters")
		}
		testContext.singleMongosLoadBalancerURI, err = addNecessaryParamsToURI(singleMongosURI)
		if err != nil {
			return fmt.Errorf("error getting single mongos load balancer uri: %w", err)
		}

		multiMongosURI := os.Getenv("MULTI_MONGOS_LB_URI")
		if multiMongosURI == "" {
			return errors.New("MULTI_MONGOS_LB_URI must be set when running against load balanced clusters")
		}
		testContext.multiMongosLoadBalancerURI, err = addNecessaryParamsToURI(multiMongosURI)
		if err != nil {
			return fmt.Errorf("error getting multi mongos load balancer uri: %w", err)
		}
	}

	testContext.authEnabled = os.Getenv("AUTH") == "auth"
	testContext.sslEnabled = os.Getenv("SSL") == "ssl"
	biRes, err := testContext.client.Database("admin").RunCommand(context.Background(), bson.D{{"buildInfo", 1}}).Raw()
	if err != nil {
		return fmt.Errorf("buildInfo error: %w", err)
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
		testContext.serverParameters, err = db.RunCommand(context.Background(), bson.D{{"getParameter", "*"}}).Raw()
		if err != nil {
			return fmt.Errorf("error getting serverParameters: %w", err)
		}
	}
	return nil
}

// Teardown cleans up resources initialized by Setup.
// This function must be called once after all tests have finished running.
func Teardown() error {
	// Dropping the test database causes an error against Atlas Data Lake.
	if !testContext.dataLake {
		if err := testContext.client.Database(TestDb).Drop(context.Background()); err != nil {
			return fmt.Errorf("error dropping test database: %w", err)
		}
	}
	if err := testContext.client.Disconnect(context.Background()); err != nil {
		return fmt.Errorf("error disconnecting test client: %w", err)
	}
	if err := testContext.topo.Disconnect(context.Background()); err != nil {
		return fmt.Errorf("error disconnecting test topology: %w", err)
	}
	return nil
}

func getServerVersion() (string, error) {
	var serverStatus bson.Raw
	err := testContext.client.Database(TestDb).RunCommand(
		context.Background(),
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
	if os.Getenv("SSL") == "ssl" {
		uri = addOptions(uri, "ssl=", "true")
	}
	caFile := os.Getenv("MONGO_GO_DRIVER_CA_FILE")
	if len(caFile) == 0 {
		return uri
	}

	return addOptions(uri, "sslCertificateAuthorityFile=", caFile)
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

func addServerlessAuthCredentials(uri string) (string, error) {
	if os.Getenv("SERVERLESS") != "serverless" {
		return uri, nil
	}
	user := os.Getenv("SERVERLESS_ATLAS_USER")
	if user == "" {
		return "", fmt.Errorf("serverless expects SERVERLESS_ATLAS_USER to be set")
	}
	password := os.Getenv("SERVERLESS_ATLAS_PASSWORD")
	if password == "" {
		return "", fmt.Errorf("serverless expects SERVERLESS_ATLAS_PASSWORD to be set")
	}

	var scheme string
	// remove the scheme
	switch {
	case strings.HasPrefix(uri, "mongodb+srv://"):
		scheme = "mongodb+srv://"
	case strings.HasPrefix(uri, "mongodb://"):
		scheme = "mongodb://"
	default:
		return "", errors.New(`scheme must be "mongodb" or "mongodb+srv"`)
	}

	uri = scheme + user + ":" + password + "@" + uri[len(scheme):]
	return uri, nil
}

func addNecessaryParamsToURI(uri string) (string, error) {
	uri = addTLSConfig(uri)
	uri = addCompressors(uri)
	return addServerlessAuthCredentials(uri)
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
