// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package testutil

import (
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"

	"github.com/10gen/mongo-go-driver/yamgo/connstring"
	"github.com/10gen/mongo-go-driver/yamgo/private/cluster"
)

var connectionString connstring.ConnString
var connectionStringOnce sync.Once
var connectionStringErr error
var liveCluster *cluster.Cluster
var liveClusterOnce sync.Once
var liveClusterErr error

// Cluster gets the globally configured cluster.
func Cluster(t *testing.T) *cluster.Cluster {

	cs := ConnString(t)

	liveClusterOnce.Do(func() {
		var err error
		liveCluster, err = cluster.New(cluster.WithConnString(cs))
		if err != nil {
			liveClusterErr = err
		} else {
			autoDropDB(t, liveCluster)
		}
	})

	if liveClusterErr != nil {
		t.Fatal(liveClusterErr)
	}

	return liveCluster
}

// ColName gets a collection name that should be unique
// to the currently executing test.
func ColName(t *testing.T) string {
	// Get this indirectly to avoid copying a mutex
	v := reflect.Indirect(reflect.ValueOf(t))
	name := v.FieldByName("name")
	return name.String()
}

// ConnString gets the globally configured connection string.
func ConnString(t *testing.T) connstring.ConnString {
	connectionStringOnce.Do(func() {
		mongodbURI := os.Getenv("MONGODB_URI")
		if mongodbURI == "" {
			mongodbURI = "mongodb://localhost:27017"
		}
		var err error
		connectionString, err = connstring.Parse(mongodbURI)
		if err != nil {
			connectionStringErr = err
		}
	})

	if connectionStringErr != nil {
		t.Fatal(connectionStringErr)
	}

	return connectionString
}

// DBName gets the globally configured database name.
func DBName(t *testing.T) string {
	cs := ConnString(t)
	if cs.Database != "" {
		return cs.Database
	}

	return fmt.Sprintf("mongo-go-driver-%d", os.Getpid())
}

// Integration should be called at the beginning of integration
// tests to ensure that they are skipped if integration testing is
// turned off.
func Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}
