// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/mongodb/mongo-go-driver/core/connstring"
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
