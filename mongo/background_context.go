// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/internal/csot"
)

// backgroundContext is an implementation of the context.Context interface that wraps a child Context. Value requests
// are forwarded to the child Context but the Done and Err functions are overridden to ensure the new context does not
// time out or get cancelled.
type backgroundContext struct {
	context.Context
	childValuesCtx context.Context
}

// newBackgroundContext creates a new Context whose behavior matches that of context.Background(), but Value calls are
// forwarded to the provided ctx parameter. If ctx is nil, context.Background() is returned.
func newBackgroundContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}

	return &backgroundContext{
		Context:        context.Background(),
		childValuesCtx: ctx,
	}
}

func (b *backgroundContext) Value(key any) any {
	return b.childValuesCtx.Value(key)
}

// newCleanupContext returns the Context used for the commitTransaction and
// abortTransaction operations WithTransaction issues on behalf of the caller.
// Deadlines and cancellations from ctx are not respected, as WithTransaction's
// documentation states, but Value lookups are still forwarded.
//
// Clearing the client-level marker is required, not incidental:
// newBackgroundContext drops the deadline while forwarding Value lookups, so a
// context marked client-level by csot.WithTimeout would arrive as a timeout
// context with no deadline. CSOT reads that as "no timeout" and retries the
// operation without bound. Clearing the marker instead lets the cleanup
// operation apply a refreshed timeout, as the CSOT specification requires for
// transaction cleanup.
func newCleanupContext(ctx context.Context) context.Context {
	return csot.WithoutClientLevel(newBackgroundContext(ctx))
}
