// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology_test

import (
	"log"

	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/topology"
)

func Example_clusterMonitoring() {
	topo, err := topology.New(nil)
	if err != nil {
		log.Fatalf("could not create topology: %v", err)
	}
	err = topo.Connect()
	if err != nil {
		log.Fatalf("could not create topology: %v", err)
	}

	sub, err := topo.Subscribe()
	if err != nil {
		log.Fatalf("could not subscribe to topology: %v", err)
	}

	for desc := range sub.Updates {
		log.Printf("%#v", desc)
	}
}
