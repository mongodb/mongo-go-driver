// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/csot"
	"go.mongodb.org/mongo-driver/v2/internal/ptrutil"
)

// TestSessionTimeoutContext covers how a session's timeout interacts with a
// caller-supplied context deadline. CommitTransaction, AbortTransaction, and
// EndSession each wrap their context with csot.WithTimeout, so these cases pin
// which of the two budgets wins.
func TestSessionTimeoutContext(t *testing.T) {
	t.Parallel()

	// A caller deadline far longer than the session timeout.
	const callerTimeout = time.Hour

	sessionTimeout := ptrutil.Ptr(500 * time.Millisecond)

	t.Run("caller deadline wins over a shorter session timeout", func(t *testing.T) {
		t.Parallel()

		parent, cancel := context.WithTimeout(context.Background(), callerTimeout)
		defer cancel()

		ctx, cancel := csot.WithTimeout(parent, sessionTimeout)
		defer cancel()

		deadline, ok := ctx.Deadline()
		assert.True(t, ok, "expected the caller deadline to be retained")
		assert.True(t, time.Until(deadline) > *sessionTimeout,
			"expected the caller deadline (~%v) to survive, got %v", callerTimeout, time.Until(deadline))
	})

	t.Run("session timeout applies when the caller sets no deadline", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := csot.WithTimeout(context.Background(), sessionTimeout)
		defer cancel()

		deadline, ok := ctx.Deadline()
		assert.True(t, ok, "expected the session timeout to be applied")
		assert.True(t, time.Until(deadline) <= *sessionTimeout,
			"expected a deadline within %v, got %v", *sessionTimeout, time.Until(deadline))
	})

	// WithTransaction hands cleanup operations a newBackgroundContext so that
	// caller deadlines and cancellations are not respected during commit and
	// abort, as WithTransaction's documentation states. These cases pin the
	// resulting context shape.
	t.Run("cleanup contexts", func(t *testing.T) {
		t.Parallel()

		newCallerContext := func(t *testing.T) context.Context {
			t.Helper()

			parent, cancel := context.WithTimeout(context.Background(), callerTimeout)
			t.Cleanup(cancel)

			ctx, cancel := csot.WithTimeout(parent, sessionTimeout)
			t.Cleanup(cancel)

			return ctx
		}

		// test for a context with no deadline that is nonetheless
		// marked client-level means "no timeout" to CSOT, which makes
		// driver.Operation retry indefinitely.
		t.Run("commit cleanup is not an unlimited-retry context", func(t *testing.T) {
			t.Parallel()

			cleanupCtx := newBackgroundContext(newCallerContext(t))

			_, hasDeadline := cleanupCtx.Deadline()
			assert.False(t, hasDeadline, "expected newBackgroundContext to drop the caller deadline")
			assert.False(t, csot.IsTimeoutContext(cleanupCtx),
				"expected the cleanup context to not be a timeout context; a deadline-less "+
					"timeout context means unlimited retries")

			// With no deadline and no marker, CommitTransaction's own
			// csot.WithTimeout applies the session timeout rather than
			// retrying without bound.
			ctx, cancel := csot.WithTimeout(cleanupCtx, sessionTimeout)
			defer cancel()

			deadline, ok := ctx.Deadline()
			assert.True(t, ok, "expected the session timeout to bound the commit cleanup")
			assert.True(t, time.Until(deadline) <= *sessionTimeout,
				"expected a deadline within %v, got %v", *sessionTimeout, time.Until(deadline))
		})

		// The CSOT spec requires timeoutMS to be refreshed for the
		// abortTransaction issued during cleanup. See the
		// client-side-operations-timeout spec, "withTransaction refreshes the
		// timeout for abortTransaction".
		t.Run("abort cleanup refreshes the session timeout", func(t *testing.T) {
			t.Parallel()

			cleanupCtx := csot.WithoutClientLevel(newBackgroundContext(newCallerContext(t)))

			ctx, cancel := csot.WithTimeout(cleanupCtx, sessionTimeout)
			defer cancel()

			deadline, ok := ctx.Deadline()
			assert.True(t, ok, "expected a refreshed deadline on the abort cleanup path")
			assert.True(t, time.Until(deadline) <= *sessionTimeout,
				"expected the refreshed deadline to be within %v, got %v",
				*sessionTimeout, time.Until(deadline))
		})
	})
}
