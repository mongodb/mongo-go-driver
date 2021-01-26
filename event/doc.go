// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package event is a library for monitoring events from the MongoDB Go
// driver. Monitors can be set for commands sent to the MongoDB server and/or
// changes on the MongoDB server.
//
// Monitoring commands requires setting a CommandMonitor object on the
// mongo.Client. A CommandMonitor can be set to monitor started, succeeded,
// and/or failed events. For example, the following code collects the names of
// started events:
//
//    var commandStarted []string
//    cmdMonitor := &event.CommandMonitor{
//      Started: func(_ context.Context, evt *event.CommandStartedEvent) {
//        commandStarted = append(commandStarted, evt.CommandName)
//      },
//    }
//    clientOpts := options.Client().ApplyURI("mongodb://localhost:27017").SetMonitor(cmdMonitor)
//    client, err := mongo.Connect(context.Background(), clientOpts)
//
// Monitoring server changes requires setting a ServerMonitor object on
// the mongo.Client. Different functions can be set on the ServerMonitor to
// monitor different kinds of events. See ServerMonitor for more details.
// The following code appends ServerHeartbeatStartedEvents to a slice:
//
//    var heartbeatStarted []*event.ServerHeartbeatStartedEvent
//    svrMonitor := &event.ServerMonitor{
//      ServerHeartbeatStarted: func(e *event.ServerHeartbeatStartedEvent) {
//	      heartbeatStarted = append(heartbeatStarted, *e)
//      }
//    }
//    clientOpts := options.Client().ApplyURI("mongodb://localhost:27017").SetServerMonitor(svrMonitor)
//    client, err := mongo.Connect(context.Background(), clientOpts)
package event
