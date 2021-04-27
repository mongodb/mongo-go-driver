// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"log"
	"os"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestMain(m *testing.M) {
	// If the cluster is behind a load balancer, enable the SetMockServiceID flag to mock server-side LB support.
	if strings.Contains(os.Getenv("MONGODB_URI"), "loadBalanced=true") {
		internal.SetMockServiceID = true
		defer func() {
			internal.SetMockServiceID = false
		}()
	}

	if err := mtest.Setup(); err != nil {
		log.Fatal(err)
	}
	defer os.Exit(m.Run())
	if err := mtest.Teardown(); err != nil {
		log.Fatal(err)
	}
}
