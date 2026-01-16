// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/integtest"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/internal/serverselector"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/operation"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/topology"
)

var (
	host             *string
	connectionString *connstring.ConnString
	dbName           string
)

func TestMain(m *testing.M) {
	flag.Parse()

	mongodbURI := os.Getenv("MONGODB_URI")
	if mongodbURI == "" {
		mongodbURI = "mongodb://localhost:27017"
	}

	mongodbURI = addTLSConfigToURI(mongodbURI)
	mongodbURI = addCompressorToURI(mongodbURI)

	var err error
	connectionString, err = connstring.ParseAndValidate(mongodbURI)
	if err != nil {
		fmt.Printf("Could not parse connection string: %v\n", err)
		os.Exit(1)
	}

	host = &connectionString.Hosts[0]

	dbName = fmt.Sprintf("mongo-go-driver-%d", os.Getpid())
	if connectionString.Database != "" {
		dbName = connectionString.Database
	}
	os.Exit(m.Run())
}

func noerr(t *testing.T, err error) {
	if err != nil {
		t.Helper()
		t.Errorf("Unexpected error: %v", err)
		t.FailNow()
	}
}

func autherr(t *testing.T, err error) {
	t.Helper()
	switch e := err.(type) {
	case topology.ConnectionError:
		if _, ok := e.Wrapped.(*auth.Error); !ok {
			t.Fatal("Expected auth error and didn't get one")
		}
	case *auth.Error:
		return
	default:
		t.Fatal("Expected auth error and didn't get one")
	}
}

// addTLSConfigToURI checks for the environmental variable indicating that the tests are being run
// on an SSL-enabled server, and if so, returns a new URI with the necessary configuration.
func addTLSConfigToURI(uri string) string {
	caFile := os.Getenv("MONGO_GO_DRIVER_CA_FILE")
	if len(caFile) == 0 {
		return uri
	}

	if !strings.ContainsRune(uri, '?') {
		if uri[len(uri)-1] != '/' {
			uri += "/"
		}

		uri += "?"
	} else {
		uri += "&"
	}

	return uri + "ssl=true&sslCertificateAuthorityFile=" + caFile
}

func addCompressorToURI(uri string) string {
	comp := os.Getenv("MONGO_GO_DRIVER_COMPRESSOR")
	if len(comp) == 0 {
		return uri
	}

	if !strings.ContainsRune(uri, '?') {
		if uri[len(uri)-1] != '/' {
			uri += "/"
		}

		uri += "?"
	} else {
		uri += "&"
	}

	return uri + "compressors=" + comp
}

// runCommand runs an arbitrary command on a given database of the target
// server.
func runCommand(s driver.Server, db string, cmd bsoncore.Document) error {
	op := operation.NewCommand(cmd).
		Database(db).
		Deployment(driver.SingleServerDeployment{Server: s})
	return op.Execute(context.Background())
}

// dropCollection drops the collection in the test cluster.
func dropCollection(t *testing.T, dbname, colname string) {
	err := operation.NewCommand(bsoncore.BuildDocument(nil, bsoncore.AppendStringElement(nil, "drop", colname))).
		Database(dbname).ServerSelector(&serverselector.Write{}).Deployment(integtest.Topology(t)).
		Execute(context.Background())
	if de, ok := err.(driver.Error); err != nil && (!ok || !de.NamespaceNotFound()) {
		require.NoError(t, err)
	}
}

// autoInsertDocs inserts the docs into the test cluster.
func autoInsertDocs(t *testing.T, writeConcern *writeconcern.WriteConcern, docs ...bsoncore.Document) {
	insertDocs(t, integtest.DBName(t), integtest.ColName(t), writeConcern, docs...)
}

// insertDocs inserts the docs into the test cluster.
func insertDocs(t *testing.T, dbname, colname string, writeConcern *writeconcern.WriteConcern, docs ...bsoncore.Document) {
	t.Helper()

	// The initial call to integtest.Topology drops the database used by the
	// tests, so we have to call it here first to prevent the existing test code
	// from dropping the database after we've inserted data.
	integtest.Topology(t)

	client, err := mongo.Connect(options.Client().ApplyURI(connectionString.Original).SetWriteConcern(writeConcern))
	require.NoError(t, err)
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	coll := client.Database(dbname).Collection(colname)
	_, err = coll.InsertMany(context.Background(), docs)
	require.NoError(t, err)
}
