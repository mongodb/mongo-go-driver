// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"net"
	"sync/atomic"
	"testing"

	"time"

	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/mongodb/mongo-go-driver/mongo/options"
	"github.com/mongodb/mongo-go-driver/mongo/readconcern"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
	"github.com/mongodb/mongo-go-driver/mongo/writeconcern"
	"github.com/mongodb/mongo-go-driver/tag"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"github.com/mongodb/mongo-go-driver/x/mongo/driver/topology"
	"github.com/mongodb/mongo-go-driver/x/network/connstring"
	"github.com/stretchr/testify/require"
	"reflect"
)

func TestClientOptions_simple(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	cs := testutil.ConnString(t)
	client, err := NewClientWithOptions(cs.String(), options.Client().SetAppName("foo"))
	require.NoError(t, err)

	require.Equal(t, "foo", client.connString.AppName)
}

func TestClientOptions_overrideConnString(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	cs := testutil.ConnString(t)
	uri := testutil.AddOptionsToURI(cs.String(), "appname=bar")

	client, err := NewClientWithOptions(uri, options.Client().SetAppName("foo"))
	require.NoError(t, err)

	require.Equal(t, "foo", client.connString.AppName)
}

func TestClientOptions_doesNotAlterConnectionString(t *testing.T) {
	t.Parallel()

	cs := connstring.ConnString{}
	client, err := newClient(cs, options.Client().SetAppName("foobar"))
	require.NoError(t, err)
	if cs.AppName != "" {
		t.Errorf("Creating a new Client should not alter the original connection string, but it did. got %s; want <empty>", cs.AppName)
	}
	if client.connString.AppName != "foobar" {
		t.Errorf("Creating a new Client should alter the internal copy of the connection string, but it didn't. got %s; want %s", client.connString.AppName, "foobar")
	}
}

func TestClientOptions_chainAll(t *testing.T) {
	t.Parallel()
	readPrefMode, err := readpref.ModeFromString("secondary")
	require.NoError(t, err)
	rp, err := readpref.New(
		readPrefMode,
		readpref.WithTagSets(tag.NewTagSetsFromMaps([]map[string]string{{"nyc": "1"}})...),
		readpref.WithMaxStaleness(2*time.Second),
	)
	rc := readconcern.New(readconcern.Level("majority"))
	wc := writeconcern.New(
		writeconcern.J(true),
		writeconcern.WTagSet("majority"),
		writeconcern.W(3),
		writeconcern.WTimeout(2*time.Second),
	)
	require.NoError(t, err)

	retryWrites := true
	opts := options.Client().SetAppName("foo").SetAuth(options.Credential{
		AuthMechanism:           "MONGODB-X509",
		AuthMechanismProperties: map[string]string{"foo": "bar"},
		AuthSource:              "$external",
		Password:                "supersecurepassword",
		Username:                "admin",
	}).SetConnectTimeout(500 * time.Millisecond).SetHeartbeatInterval(15 * time.Second).SetHosts([]string{
		"mongodb://localhost:27018",
		"mongodb://localhost:27019",
	}).SetLocalThreshold(time.Second).SetMaxConnIdleTime(30 * time.Second).SetMaxPoolSize(150).
		SetReadConcern(rc).SetReadPreference(rp).SetReplicaSet("foo").
		SetRetryWrites(retryWrites).SetServerSelectionTimeout(time.Second).
		SetSingle(false).SetSocketTimeout(2 * time.Second).SetSSL(&options.SSLOpt{
		Enabled:                      true,
		ClientCertificateKeyFile:     "client.pem",
		ClientCertificateKeyPassword: nil,
		Insecure:                     false,
		CaFile:                       "ca.pem",
	}).SetWriteConcern(wc)

	expectedClient := &options.ClientOptions{
		TopologyOptions: []topology.Option{},
		ConnString: connstring.ConnString{
			AppName:                 "foo",
			AuthMechanism:           "MONGODB-X509",
			AuthMechanismProperties: map[string]string{"foo": "bar"},
			AuthSource:              "$external",
			Username:                "admin",
			Password:                "supersecurepassword",
			ConnectTimeout:          500 * time.Millisecond,
			ConnectTimeoutSet:       true,
			HeartbeatInterval:       15 * time.Second,
			HeartbeatIntervalSet:    true,
			Hosts: []string{
				"mongodb://localhost:27018",
				"mongodb://localhost:27019",
			},
			LocalThresholdSet:                  true,
			LocalThreshold:                     time.Second,
			MaxConnIdleTime:                    30 * time.Second,
			MaxConnIdleTimeSet:                 true,
			MaxPoolSize:                        150,
			MaxPoolSizeSet:                     true,
			ReplicaSet:                         "foo",
			ServerSelectionTimeoutSet:          true,
			ServerSelectionTimeout:             time.Second,
			Connect:                            connstring.AutoConnect,
			ConnectSet:                         true,
			SocketTimeout:                      2 * time.Second,
			SocketTimeoutSet:                   true,
			SSL:                                true,
			SSLSet:                             true,
			SSLClientCertificateKeyFile:        "client.pem",
			SSLClientCertificateKeyFileSet:     true,
			SSLClientCertificateKeyPassword:    nil,
			SSLClientCertificateKeyPasswordSet: false, // will not be set if it's nil
			SSLInsecure:                        false,
			SSLInsecureSet:                     true,
			SSLCaFile:                          "ca.pem",
			SSLCaFileSet:                       true,
		},
		ReadConcern:    rc,
		ReadPreference: rp,
		WriteConcern:   wc,
		RetryWrites:    &retryWrites,
	}

	require.Equal(t, expectedClient, opts)
}

func TestClientOptions_sslOptions(t *testing.T) {
	t.Parallel()

	t.Run("TestEmptyOptionsNotSet", func(t *testing.T) {
		ssl := &options.SSLOpt{}
		c, err := NewClientWithOptions("mongodb://localhost", options.Client().SetSSL(ssl))
		require.NoError(t, err)

		require.Equal(t, c.connString.SSLClientCertificateKeyFile, "")
		require.Equal(t, c.connString.SSLClientCertificateKeyFileSet, false)
		require.Nil(t, c.connString.SSLClientCertificateKeyPassword)
		require.Equal(t, c.connString.SSLClientCertificateKeyPasswordSet, false)
		require.Equal(t, c.connString.SSLCaFile, "")
		require.Equal(t, c.connString.SSLCaFileSet, false)
	})

	t.Run("TestNonEmptyOptionsSet", func(t *testing.T) {
		f := func() string {
			return "KeyPassword"
		}

		ssl := &options.SSLOpt{
			ClientCertificateKeyFile:     "KeyFile",
			ClientCertificateKeyPassword: f,
			CaFile:                       "CaFile",
		}
		c, err := NewClientWithOptions("mongodb://localhost", options.Client().SetSSL(ssl))
		require.NoError(t, err)

		require.Equal(t, c.connString.SSLClientCertificateKeyFile, "KeyFile")
		require.Equal(t, c.connString.SSLClientCertificateKeyFileSet, true)
		require.Equal(t, reflect.ValueOf(c.connString.SSLClientCertificateKeyPassword).Pointer(), reflect.ValueOf(f).Pointer())
		require.Equal(t, c.connString.SSLClientCertificateKeyPasswordSet, true)
		require.Equal(t, c.connString.SSLCaFile, "CaFile")
		require.Equal(t, c.connString.SSLCaFileSet, true)
	})
}

func TestClientOptions_CustomDialer(t *testing.T) {
	td := &testDialer{d: &net.Dialer{}}
	opts := options.Client().SetDialer(td)
	client, err := newClient(testutil.ConnString(t), opts)
	require.NoError(t, err)
	err = client.Connect(context.Background())
	require.NoError(t, err)
	_, err = client.ListDatabases(context.Background(), bsonx.Doc{})
	require.NoError(t, err)
	got := atomic.LoadInt32(&td.called)
	if got < 1 {
		t.Errorf("Custom dialer was not used when dialing new connections")
	}
}

type testDialer struct {
	called int32
	d      Dialer
}

func (td *testDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	atomic.AddInt32(&td.called, 1)
	return td.d.DialContext(ctx, network, address)
}
