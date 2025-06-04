// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"errors"
	"sync/atomic"
)

type contextListener interface {
	Listen(context.Context, func())
	StopListening() bool
}

// contextDoneListener listens for context-ending eventsand notifies listeners
// via a callback function.
type contextDoneListener struct {
	aborted     atomic.Value
	done        chan struct{}
	blockOnDone bool
}

var _ contextListener = &contextDoneListener{}

// newContextDoneListener constructs a contextDoneListener that will block
// when a context is done until StopListening is called.
func newContextDoneListener() *contextDoneListener {
	return &contextDoneListener{
		done:        make(chan struct{}),
		blockOnDone: true,
	}
}

// newNonBlockingContextDoneLIstener constructs a contextDoneListener that
// will not block when a context is done. In this case there are two ways to
// unblock the listener: a finished context or a call to StopListening.
func newNonBlockingContextDoneListener() *contextDoneListener {
	return &contextDoneListener{
		done:        make(chan struct{}),
		blockOnDone: false,
	}
}

// Listen blocks until the provided context is cancelled or listening is aborted
// via the StopListening function. If this detects that the context has been
// cancelled (i.e. errors.Is(ctx.Err(), context.Canceled), the provided callback
// is called to abort in-progress work. If blockOnDone is true, this function
// will block until StopListening is called, even if the context expires.
func (c *contextDoneListener) Listen(ctx context.Context, abortFn func()) {
	c.aborted.Store(false)

	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.Canceled) {
			c.aborted.Store(true)

			abortFn()
		}

		if c.blockOnDone {
			<-c.done
		}
	case <-c.done:
	}
}

// StopListening stops the in-progress Listen call. If blockOnDone is true, then
// this blocks if there is no in-progress Listen call. This function will return
// true if the provided abort callback was called when listening for
// cancellation on the previous context.
func (c *contextDoneListener) StopListening() bool {
	if c.blockOnDone {
		c.done <- struct{}{}
	} else {
		select {
		case c.done <- struct{}{}:
		default:
		}
	}

	if aborted := c.aborted.Load(); aborted != nil {
		return aborted.(bool)
	}

	return false
}
