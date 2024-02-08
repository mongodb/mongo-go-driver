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

	"go.mongodb.org/mongo-driver/internal/integtest"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/auth"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

var host *string
var connectionString *connstring.ConnString
var dbName string

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

// runCommand runs an arbitrary command on a given database of target server
func runCommand(s driver.Server, db string, cmd bsoncore.Document) (bsoncore.Document, error) {
	op := operation.NewCommand(cmd).
		Database(db).Deployment(driver.SingleServerDeployment{Server: s})
	err := op.Execute(context.Background())
	res := op.Result()
	return res, err
}

// dropCollection drops the collection in the test cluster.
func dropCollection(t *testing.T, dbname, colname string) {
	err := operation.NewCommand(bsoncore.BuildDocument(nil, bsoncore.AppendStringElement(nil, "drop", colname))).
		Database(dbname).ServerSelector(description.WriteSelector()).Deployment(integtest.Topology(t)).
		Execute(context.Background())
	if de, ok := err.(driver.Error); err != nil && !(ok && de.NamespaceNotFound()) {
		require.NoError(t, err)
	}
}

// autoInsertDocs inserts the docs into the test cluster.
func autoInsertDocs(t *testing.T, writeConcern *writeconcern.WriteConcern, docs ...bsoncore.Document) {
	insertDocs(t, integtest.DBName(t), integtest.ColName(t), writeConcern, docs...)
}

// insertDocs inserts the docs into the test cluster.
func insertDocs(t *testing.T, dbname, colname string, writeConcern *writeconcern.WriteConcern, docs ...bsoncore.Document) {
	err := operation.NewInsert(docs...).
		Collection(colname).
		Database(dbname).
		Deployment(integtest.Topology(t)).
		ServerSelector(description.WriteSelector()).
		WriteConcern(writeConcern).
		Execute(context.Background())
	require.NoError(t, err)
}
