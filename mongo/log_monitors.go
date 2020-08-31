// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo/mongologger"
)

func newCommandLogger(l *mongologger.MongoLogger, cm *event.CommandMonitor) *event.CommandMonitor {
	return &event.CommandMonitor{
		Started: func(ctx context.Context, cse *event.CommandStartedEvent) {
			l.Log(mongologger.Command, mongologger.Debug, "Command started",
				mongologger.Stringer("command", cse.Command),
				mongologger.String("databaseName", cse.DatabaseName),
				mongologger.String("commandName", cse.CommandName),
				mongologger.Int64("requestId", cse.RequestID),
				mongologger.String("driverConnectionId", cse.ConnectionID),
				// TODO: add serverConnectionId, explicitSession
			)
			if cm != nil && cm.Started != nil {
				cm.Started(ctx, cse)
			}
		},
		Succeeded: func(ctx context.Context, cse *event.CommandSucceededEvent) {
			l.Log(mongologger.Command, mongologger.Debug, "Command succeeded",
				mongologger.Int64("durationNanos", cse.DurationNanos),
				mongologger.Stringer("reply", cse.Reply),
				mongologger.String("commandName", cse.CommandName),
				mongologger.Int64("requestId", cse.RequestID),
				mongologger.String("driverConnectionId", cse.ConnectionID),
				// TODO: add serverConnectionId, explicitSession
			)
			if cm != nil && cm.Started != nil {
				cm.Succeeded(ctx, cse)
			}
		},
		Failed: func(ctx context.Context, cfe *event.CommandFailedEvent) {
			l.Log(mongologger.Command, mongologger.Debug, "Command failed",
				mongologger.Int64("durationNanos", cfe.DurationNanos),
				mongologger.String("commandName", cfe.CommandName),
				mongologger.String("failure", cfe.Failure),
				mongologger.String("driverConnectionId", cfe.ConnectionID),
				// TODO: add serverConnectionId, explicitSession
			)
			if cm != nil && cm.Started != nil {
				cm.Failed(ctx, cfe)
			}
		},
	}
}
