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

type withoutMaxTime struct{}

// WithoutMaxTime returns a new context with a "withoutMaxTime" value that
// is used to inform operation construction to not include "maxTimeMS" in a wire
// message, regardless of a context deadline. This is specifically used for
// non-awaitable hello commands.
func WithoutMaxTime(ctx context.Context) context.Context {
	return context.WithValue(ctx, withoutMaxTime{}, true)
}

// IsWithoutMaxTime checks if the provided context has been assigned the
// "withoutMaxTime" value.
func IsWithoutMaxTime(ctx context.Context) bool {
	return ctx.Value(withoutMaxTime{}) != nil
}

type clientLevel struct{}

func AsClientLevel(parent context.Context) context.Context {
	return context.WithValue(parent, clientLevel{}, true)
}

func IsClientLevel(ctx context.Context) bool {
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

	return ok || IsClientLevel(ctx)
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

	// In the following conditions, do nothing:
	//	1. The parent already has a deadline
	//	2. The parent does not have a deadline, but a client-level timeout has
	//		 been applied.
	//	3. The parent does not have a deadline, there is not client-level timeout,
	//     and the timeout parameter DNE.
	if IsTimeoutContext(parent) || timeout == nil {
		return parent, cancel
	}

	// If a client-level timeout has not been applied, then apply it.
	parent = AsClientLevel(parent)

	// If the parent does not have a deadline and the timeout is zero, then
	// set the zero value.
	if timeout != nil && *timeout == 0 {
		return parent, cancel
	}

	// If the parent does not have a dealine and the timeout is non-zero, then
	// apply the timeout.
	return context.WithTimeout(parent, *timeout)
}

// WithServerSelectionTimeout creates a context with a timeout that is the
// minimum of serverSelectionTimeoutMS and context deadline. The usage of
// non-positive values for serverSelectionTimeoutMS are an anti-pattern and are
// not considered in this calculation.
func WithServerSelectionTimeout(
	parent context.Context,
	serverSelectionTimeout time.Duration,
) (context.Context, context.CancelFunc) {
	var timeout time.Duration

	deadline, ok := parent.Deadline()
	if ok {
		timeout = time.Until(deadline)
	}

	// If there is no deadline on the parent context and the server selection
	// timeout DNE, then do nothing.
	if !ok && serverSelectionTimeout <= 0 {
		return parent, func() {}
	}

	// Otherwise, take the minimum of the two and return a new context with that
	// value as the deadline.
	if !ok {
		timeout = serverSelectionTimeout
	} else if timeout >= serverSelectionTimeout && serverSelectionTimeout > 0 {
		// Only use the serverSelectionTimeout value if it is less than the existing
		// timeout and is positive.
		timeout = serverSelectionTimeout
	}

	return context.WithTimeout(parent, timeout)
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
