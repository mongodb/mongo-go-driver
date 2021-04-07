// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

var defaultRunKillAllSessionsValue = true

// Options is the type used to configure tests
type Options struct {
	// Specifies if killAllSessions should be run after the test completes.
	// Defaults to true
	RunKillAllSessions *bool
}

// NewOptions creates an empty options interface
func NewOptions() *Options {
	return &Options{&defaultRunKillAllSessionsValue}
}

// SetRunKillAllSessions sets the value for RunKillAllSessions
func (op *Options) SetRunKillAllSessions(killAllSessions bool) *Options {
	op.RunKillAllSessions = &killAllSessions
	return op
}

// MergeOptions combines the given *Options into a single *Options in a last one wins fashion.
func MergeOptions(opts ...*Options) *Options {
	op := NewOptions()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.RunKillAllSessions != nil {
			op.RunKillAllSessions = opt.RunKillAllSessions
		}
	}

	return op
}
