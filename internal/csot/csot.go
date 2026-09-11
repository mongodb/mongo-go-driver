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

func isClientLevel(ctx context.Context) bool {
	val := ctx.Value(clientLevel{})
	if val == nil {
		return false
	}

	return val.(bool)
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

	if timeout == nil || IsTimeoutContext(parent) {
		// In the following conditions, do nothing:
		//	1. The parent already has a deadline
		//	2. The parent does not have a deadline, but a client-level timeout has
		//		 been applied.
		//	3. The parent does not have a deadline, there is not client-level
		//     timeout, and the timeout parameter DNE.
		return parent, cancel
	}

	// If a client-level timeout has not been applied, then apply it.
	parent = context.WithValue(parent, clientLevel{}, true)

	dur := *timeout

	if dur == 0 {
		// If the parent does not have a deadline and the timeout is zero, then
		// do nothing.
		return parent, cancel
	}

	// If the parent does not have a dealine and the timeout is non-zero, then
	// apply the timeout.
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
