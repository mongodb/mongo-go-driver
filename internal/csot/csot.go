// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package csot

import (
	"context"
	"time"
)

type clientLevel struct{}
type timeoutDurationKey struct{}

func isClientLevel(ctx context.Context) bool {
	val := ctx.Value(clientLevel{})
	if val == nil {
		return false
	}

	return val.(bool)
}

// GetTimeoutDuration retrieves the timeout duration stored in the context.
// This is used to "refresh" the timeout for operations like abortTransaction.
// Returns nil if no timeout duration was stored.
func GetTimeoutDuration(ctx context.Context) *time.Duration {
	val := ctx.Value(timeoutDurationKey{})
	if val == nil {
		return nil
	}

	dur := val.(time.Duration)
	return &dur
}

// IsTimeoutContext checks if the provided context has been assigned a deadline
// or has unlimited retries.
func IsTimeoutContext(ctx context.Context) bool {
	_, ok := ctx.Deadline()

	return ok || isClientLevel(ctx)
}

// WithTimeout will set the given timeout on the context, if no deadline has
// already been set.
//
// This function assumes that the timeout field is static, given that the
// timeout should be sourced from the client. Therefore, once a timeout function
// parameter has  been applied to the context, it will remain for the lifetime
// of the context.
func WithTimeout(parent context.Context, timeout *time.Duration) (context.Context, context.CancelFunc) {
	cancel := func() {}

	// If timeout is nil, do nothing.
	if timeout == nil {
		return parent, cancel
	}

	// If the parent already has a deadline, don't override it.
	if _, hasDeadline := parent.Deadline(); hasDeadline {
		return parent, cancel
	}

	dur := *timeout

	// If the client-level marker is already set but there's no deadline (e.g.,
	// the deadline was stripped by newBackgroundContext), apply a fresh timeout
	// using the stored duration. This enables operations like AbortTransaction
	// to include maxTimeMS even when called with a background context.
	if isClientLevel(parent) {
		// Try to get the stored duration for refreshing (per CSOT spec).
		storedDur := GetTimeoutDuration(parent)
		if storedDur != nil {
			dur = *storedDur
		}
		if dur == 0 {
			return parent, cancel
		}
		return context.WithTimeout(parent, dur)
	}

	// First time applying client-level timeout: set the marker and store the duration.
	parent = context.WithValue(parent, clientLevel{}, true)
	parent = context.WithValue(parent, timeoutDurationKey{}, dur)

	if dur == 0 {
		// 0 means infinite timeout, no deadline needed.
		return parent, cancel
	}

	return context.WithTimeout(parent, dur)
}

// WithServerSelectionTimeout creates a context with a timeout that is the
// minimum of serverSelectionTimeoutMS and context deadline. The usage of
// non-positive values for serverSelectionTimeoutMS are an anti-pattern and are
// not considered in this calculation.
func WithServerSelectionTimeout(
	parent context.Context,
	serverSelectionTimeout time.Duration,
) (context.Context, context.CancelFunc) {
	if serverSelectionTimeout <= 0 {
		return parent, func() {}
	}

	return context.WithTimeout(parent, serverSelectionTimeout)
}

// ZeroRTTMonitor implements the RTTMonitor interface and is used internally for testing. It returns 0 for all
// RTT calculations and an empty string for RTT statistics.
type ZeroRTTMonitor struct{}

// EWMA implements the RTT monitor interface.
func (zrm *ZeroRTTMonitor) EWMA() time.Duration {
	return 0
}

// Min implements the RTT monitor interface.
func (zrm *ZeroRTTMonitor) Min() time.Duration {
	return 0
}

// P90 implements the RTT monitor interface.
func (zrm *ZeroRTTMonitor) P90() time.Duration {
	return 0
}

// Stats implements the RTT monitor interface.
func (zrm *ZeroRTTMonitor) Stats() string {
	return ""
}
