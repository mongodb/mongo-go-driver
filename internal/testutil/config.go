// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package testutil

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/mongodb/mongo-go-driver/core/connstring"
	"github.com/mongodb/mongo-go-driver/core/topology"
	"github.com/mongodb/mongo-go-driver/core/event"
	"github.com/mongodb/mongo-go-driver/core/connection"
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/stretchr/testify/require"
	"github.com/mongodb/mongo-go-driver/core/command"
	"github.com/mongodb/mongo-go-driver/bson"
)

var connectionString connstring.ConnString
var connectionStringOnce sync.Once
var connectionStringErr error
var liveTopology *topology.Topology
var liveTopologyOnce sync.Once
var liveTopologyErr error
var monitoredTopology *topology.Topology
var monitoredTopologyOnce sync.Once
var monitoredTopologyErr error

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

// AddCompressorToUri checks for the environment variable indicating that the tests are being run with compression
// enabled. If so, it returns a new URI with the necessary configuration
func AddCompressorToUri(uri string) string {
	comp := os.Getenv("MONGO_GO_DRIVER_COMPRESSOR")
	if len(comp) == 0 {
		return uri
	}

	return AddOptionsToURI(uri, "compressors=", comp)
}

// MonitoredTopology gets the globally configured topology and attaches a command monitor.
func MonitoredTopology(t *testing.T, monitor *event.CommandMonitor) *topology.Topology {
	cs := ConnString(t)
	opts := []topology.Option{
			topology.WithConnString(func(connstring.ConnString) connstring.ConnString { return cs }),
			topology.WithServerOptions(func(opts ...topology.ServerOption) []topology.ServerOption {
				return append(
					opts,
					topology.WithConnectionOptions(func(opts ...connection.Option) []connection.Option {
						return append(
							opts,
							connection.WithMonitor(func(*event.CommandMonitor) *event.CommandMonitor {
								return monitor
							}),
						)
					}),
				)
			}),
	}

	monitoredTopologyOnce.Do(func() {
		var err error
		monitoredTopology, err = topology.New(opts...)
		if err != nil {
			monitoredTopologyErr = err
		} else {
			monitoredTopology.Connect(context.Background())
			s, err := monitoredTopology.SelectServer(context.Background(), description.WriteSelector())
			require.NoError(t, err)

			c, err := s.Connection(context.Background())
			require.NoError(t, err)

			_, err = (&command.Write{
				DB: DBName(t),
				Command: bson.NewDocument(bson.EC.Int32("dropDatabase", 1)),
			}).RoundTrip(context.Background(), s.SelectedDescription(), c)

			require.NoError(t, err)
		}
	})

	if monitoredTopologyErr != nil {
		t.Fatal(monitoredTopologyErr)
	}

	return monitoredTopology
}

// Topology gets the globally configured topology.
func Topology(t *testing.T) *topology.Topology {
	cs := ConnString(t)
	liveTopologyOnce.Do(func() {
		var err error
		liveTopology, err = topology.New(topology.WithConnString(func(connstring.ConnString) connstring.ConnString { return cs }))
		if err != nil {
			liveTopologyErr = err
		} else {
			liveTopology.Connect(context.Background())
			s, err := liveTopology.SelectServer(context.Background(), description.WriteSelector())
			require.NoError(t, err)

			c, err := s.Connection(context.Background())
			require.NoError(t, err)

			_, err = (&command.Write{
				DB:      DBName(t),
				Command: bson.NewDocument(bson.EC.Int32("dropDatabase", 1)),
			}).RoundTrip(context.Background(), s.SelectedDescription(), c)
			require.NoError(t, err)
		}
	})

	if liveTopologyErr != nil {
		t.Fatal(liveTopologyErr)
	}

	return liveTopology
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
		connectionString, connectionStringErr = GetConnString()
		mongodbURI := os.Getenv("MONGODB_URI")
		if mongodbURI == "" {
			mongodbURI = "mongodb://localhost:27017"
		}

		mongodbURI = AddTLSConfigToURI(mongodbURI)
		mongodbURI = AddCompressorToUri(mongodbURI)

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

func GetConnString() (connstring.ConnString, error) {
	mongodbURI := os.Getenv("MONGODB_URI")
	if mongodbURI == "" {
		mongodbURI = "mongodb://localhost:27017"
	}

	mongodbURI = AddTLSConfigToURI(mongodbURI)

	cs, err := connstring.Parse(mongodbURI)
	if err != nil {
		return connstring.ConnString{}, err
	}

	return cs, nil
}

// DBName gets the globally configured database name.
func DBName(t *testing.T) string {
	return GetDBName(ConnString(t))
}

func GetDBName(cs connstring.ConnString) string {
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
