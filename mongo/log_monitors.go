// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo/mongolog"
)

// newLoggingCommandMonitor wraps cm to return a CommandMonitor that also logs to logger.
func newLoggingCommandMonitor(logger *mongolog.MongoLogger, cm *event.CommandMonitor) *event.CommandMonitor {
	return &event.CommandMonitor{
		Started: func(ctx context.Context, cse *event.CommandStartedEvent) {
			logger.Log(mongolog.Command, mongolog.Debug, "Command started",
				mongolog.Stringer("command", cse.Command),
				mongolog.String("databaseName", cse.DatabaseName),
				mongolog.String("commandName", cse.CommandName),
				mongolog.Int64("requestId", cse.RequestID),
				mongolog.String("driverConnectionId", cse.ConnectionID),
				// TODO: add serverConnectionId, explicitSession
			)
			if cm != nil && cm.Started != nil {
				cm.Started(ctx, cse)
			}
		},
		Succeeded: func(ctx context.Context, cse *event.CommandSucceededEvent) {
			logger.Log(mongolog.Command, mongolog.Debug, "Command succeeded",
				mongolog.Int64("durationNanos", cse.DurationNanos),
				mongolog.Stringer("reply", cse.Reply),
				mongolog.String("commandName", cse.CommandName),
				mongolog.Int64("requestId", cse.RequestID),
				mongolog.String("driverConnectionId", cse.ConnectionID),
				// TODO: add serverConnectionId, explicitSession
			)
			if cm != nil && cm.Succeeded != nil {
				cm.Succeeded(ctx, cse)
			}
		},
		Failed: func(ctx context.Context, cfe *event.CommandFailedEvent) {
			logger.Log(mongolog.Command, mongolog.Debug, "Command failed",
				mongolog.Int64("durationNanos", cfe.DurationNanos),
				mongolog.String("commandName", cfe.CommandName),
				mongolog.String("failure", cfe.Failure),
				mongolog.String("driverConnectionId", cfe.ConnectionID),
				// TODO: add serverConnectionId, explicitSession
			)
			if cm != nil && cm.Failed != nil {
				cm.Failed(ctx, cfe)
			}
		},
	}
}
