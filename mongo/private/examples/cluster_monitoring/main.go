// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"log"

	"github.com/kr/pretty"
	"github.com/mongodb/mongo-go-driver/mongo/private/cluster"
)

func main() {
	monitor, err := cluster.StartMonitor()
	if err != nil {
		log.Fatalf("could not start cluster monitor: %v", err)
	}

	updates, _, _ := monitor.Subscribe()

	for desc := range updates {
		log.Printf("%# v", pretty.Formatter(desc))
	}

}
