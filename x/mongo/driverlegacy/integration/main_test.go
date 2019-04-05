// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"testing"

	"go.mongodb.org/mongo-driver/x/mongo/driverlegacy/auth"
	"go.mongodb.org/mongo-driver/x/network/connection"
	"go.mongodb.org/mongo-driver/x/network/connstring"
)

var host = flag.String("host", "127.0.0.1:27017", "specify the location of a running mongodb server.")
var connectionString connstring.ConnString
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
	connectionString, err = connstring.Parse(mongodbURI)
	if err != nil {
		fmt.Printf("Could not parse connection string: %v\n", err)
		os.Exit(1)
	}

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
	switch err.(type) {
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

type netconn struct {
	net.Conn
	closed chan struct{}
	d      *dialer
}

func (nc *netconn) Close() error {
	nc.closed <- struct{}{}
	nc.d.connclosed(nc)
	return nc.Conn.Close()
}

type dialer struct {
	connection.Dialer
	opened map[*netconn]struct{}
	closed map[*netconn]struct{}
	sync.Mutex
}

func newdialer(d connection.Dialer) *dialer {
	return &dialer{Dialer: d, opened: make(map[*netconn]struct{}), closed: make(map[*netconn]struct{})}
}

func (d *dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d.Lock()
	defer d.Unlock()
	c, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	nc := &netconn{Conn: c, closed: make(chan struct{}, 1), d: d}
	d.opened[nc] = struct{}{}
	return nc, nil
}

func (d *dialer) connclosed(nc *netconn) {
	d.Lock()
	defer d.Unlock()
	d.closed[nc] = struct{}{}
}

func (d *dialer) lenopened() int {
	d.Lock()
	defer d.Unlock()
	return len(d.opened)
}

func (d *dialer) lenclosed() int {
	d.Lock()
	defer d.Unlock()
	return len(d.closed)
}

// bootstrapConnections lists for num connections on the returned address. The user provided run
// function will be called with the accepted connection. The user is responsible for closing the
// connection.
func bootstrapConnections(t *testing.T, num int, run func(net.Conn)) net.Addr {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Errorf("Could not set up a listener: %v", err)
		t.FailNow()
	}
	go func() {
		for i := 0; i < num; i++ {
			c, err := l.Accept()
			if err != nil {
				t.Errorf("Could not accept a connection: %v", err)
			}
			go run(c)
		}
		_ = l.Close()
	}()
	return l.Addr()
}
