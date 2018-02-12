// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"log"
	"time"

	"github.com/kr/pretty"
	"github.com/mongodb/mongo-go-driver/mongo/model"
	"github.com/mongodb/mongo-go-driver/mongo/private/conn"
	"github.com/mongodb/mongo-go-driver/mongo/private/server"
)

func main() {
	monitor, err := server.StartMonitor(
		model.Addr("localhost:27017"),
		server.WithHeartbeatInterval(2*time.Second),
		server.WithConnectionOptions(
			conn.WithAppName("server_monitor test"),
		),
	)
	if err != nil {
		log.Fatalf("could not start server monitor: %v", err)
	}

	updates, _, _ := monitor.Subscribe()

	for desc := range updates {
		log.Printf("%# v", pretty.Formatter(desc))
	}
}
