// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integtest

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

var connectionString *connstring.ConnString
var connectionStringOnce sync.Once
var connectionStringErr error
var liveTopology *topology.Topology
var liveTopologyOnce sync.Once
var liveTopologyErr error

// AddOptionsToURI appends connection string options to a URI.
func AddOptionsToURI(uri string, opts ...string) string {
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

// AddTLSConfigToURI checks for the environmental variable indicating that the tests are being run
// on an SSL-enabled server, and if so, returns a new URI with the necessary configuration.
func AddTLSConfigToURI(uri string) string {
	caFile := os.Getenv("MONGO_GO_DRIVER_CA_FILE")
	if len(caFile) == 0 {
		return uri
	}

	return AddOptionsToURI(uri, "ssl=true&sslCertificateAuthorityFile=", caFile)
}

// AddCompressorToURI checks for the environment variable indicating that the tests are being run with compression
// enabled. If so, it returns a new URI with the necessary configuration
func AddCompressorToURI(uri string) string {
	comp := os.Getenv("MONGO_GO_DRIVER_COMPRESSOR")
	if len(comp) == 0 {
		return uri
	}

	return AddOptionsToURI(uri, "compressors=", comp)
}

// AddTestServerAPIVersion adds the latest server API version in a ServerAPIOptions to passed-in opts.
func AddTestServerAPIVersion(opts *options.ClientOptions) {
	if os.Getenv("REQUIRE_API_VERSION") == "true" {
		opts.SetServerAPIOptions(options.ServerAPI(driver.TestServerAPIVersion))
	}
}

// MonitoredTopology returns a new topology with the command monitor attached
func MonitoredTopology(t *testing.T, dbName string, monitor *event.CommandMonitor) *topology.Topology {
	uri, err := MongoDBURI()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := topology.NewConfig(options.Client().ApplyURI(uri).SetMonitor(monitor), nil)
	if err != nil {
		t.Fatal(err)
	}

	monitoredTopology, err := topology.New(cfg)
	if err != nil {
		t.Fatal(err)
	} else {
		_ = monitoredTopology.Connect()

		err = operation.NewCommand(bsoncore.BuildDocument(nil, bsoncore.AppendInt32Element(nil, "dropDatabase", 1))).
			Database(dbName).ServerSelector(description.WriteSelector()).Deployment(monitoredTopology).Execute(context.Background())

		require.NoError(t, err)
	}

	return monitoredTopology
}

// Topology gets the globally configured topology.
func Topology(t *testing.T) *topology.Topology {
	uri, err := MongoDBURI()
	require.NoError(t, err, "error constructing mongodb URI: %v", err)
	cfg, err := topology.NewConfig(options.Client().ApplyURI(uri), nil)
	require.NoError(t, err, "error constructing topology config: %v", err)

	liveTopologyOnce.Do(func() {
		var err error
		liveTopology, err = topology.New(cfg)
		if err != nil {
			liveTopologyErr = err
		} else {
			_ = liveTopology.Connect()

			err = operation.NewCommand(bsoncore.BuildDocument(nil, bsoncore.AppendInt32Element(nil, "dropDatabase", 1))).
				Database(DBName(t)).ServerSelector(description.WriteSelector()).Deployment(liveTopology).Execute(context.Background())
			require.NoError(t, err)
		}
	})

	if liveTopologyErr != nil {
		t.Fatal(liveTopologyErr)
	}

	return liveTopology
}

// TopologyWithCredential takes an "options.Credential" object and returns a connected topology.
func TopologyWithCredential(t *testing.T, credential options.Credential) *topology.Topology {
	uri, err := MongoDBURI()
	if err != nil {
		t.Fatalf("error constructing mongodb URI: %v", err)
	}
	cfg, err := topology.NewConfig(options.Client().ApplyURI(uri).SetAuth(credential), nil)
	if err != nil {
		t.Fatalf("error constructing topology config: %v", err)
	}
	topology, err := topology.New(cfg)
	if err != nil {
		t.Fatal("Could not construct topology")
	}
	err = topology.Connect()
	if err != nil {
		t.Fatal("Could not start topology connection")
	}
	return topology
}

// ColName gets a collection name that should be unique
// to the currently executing test.
func ColName(t *testing.T) string {
	// Get this indirectly to avoid copying a mutex
	v := reflect.Indirect(reflect.ValueOf(t))
	name := v.FieldByName("name")
	return name.String()
}

// MongoDBURI will construct the MongoDB URI from the MONGODB_URI environment variable for testing. The default host is
// "localhost" and the default port is "27017"
func MongoDBURI() (string, error) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	uri = AddTLSConfigToURI(uri)
	uri = AddCompressorToURI(uri)
	uri, err := AddServerlessAuthCredentials(uri)
	return uri, err
}

// AddServerlessAuthCredentials will attempt to construct the serverless auth credentials for a URI.
func AddServerlessAuthCredentials(uri string) (string, error) {
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

// ConnString gets the globally configured connection string.
func ConnString(t *testing.T) *connstring.ConnString {
	connectionStringOnce.Do(func() {
		uri, err := MongoDBURI()
		require.NoError(t, err, "error constructing mongodb URI: %v", err)

		connectionString, err = connstring.ParseAndValidate(uri)
		if err != nil {
			connectionStringErr = err
		}
	})
	if connectionStringErr != nil {
		t.Fatal(connectionStringErr)
	}

	return connectionString
}

func GetConnString() (*connstring.ConnString, error) {
	mongodbURI := os.Getenv("MONGODB_URI")
	if mongodbURI == "" {
		mongodbURI = "mongodb://localhost:27017"
	}

	mongodbURI = AddTLSConfigToURI(mongodbURI)

	cs, err := connstring.ParseAndValidate(mongodbURI)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

// DBName gets the globally configured database name.
func DBName(t *testing.T) string {
	return GetDBName(ConnString(t))
}

func GetDBName(cs *connstring.ConnString) string {
	if cs.Database != "" {
		return cs.Database
	}

	return fmt.Sprintf("mongo-go-driver-%d", os.Getpid())
}

// CompareVersions compares two version number strings (i.e. positive integers separated by
// periods). Comparisons are done to the lesser precision of the two versions. For example, 3.2 is
// considered equal to 3.2.11, whereas 3.2.0 is considered less than 3.2.11.
//
// Returns a positive int if version1 is greater than version2, a negative int if version1 is less
// than version2, and 0 if version1 is equal to version2.
func CompareVersions(t *testing.T, v1 string, v2 string) int {
	n1 := strings.Split(v1, ".")
	n2 := strings.Split(v2, ".")

	for i := 0; i < int(math.Min(float64(len(n1)), float64(len(n2)))); i++ {
		i1, err := strconv.Atoi(n1[i])
		require.NoError(t, err)

		i2, err := strconv.Atoi(n2[i])
		require.NoError(t, err)

		difference := i1 - i2
		if difference != 0 {
			return difference
		}
	}

	return 0
}
