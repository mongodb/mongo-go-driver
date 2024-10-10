// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// DefaultCausalConsistency is the default value for the CausalConsistency option.
var DefaultCausalConsistency = true

// SessionOptions represents arguments that can be used to configure a Session.
//
// See corresponding setter methods for documentation.
type SessionOptions struct {
	CausalConsistency         *bool
	DefaultTransactionOptions *TransactionOptionsBuilder
	Snapshot                  *bool
}

// SessionOptionsBuilder represents functional options that configure a Sessionopts.
type SessionOptionsBuilder struct {
	Opts []func(*SessionOptions) error
}

// Session creates a new SessionOptions instance.
func Session() *SessionOptionsBuilder {
	return &SessionOptionsBuilder{}
}

// List returns a list of SessionOptions setter functions.
func (s *SessionOptionsBuilder) List() []func(*SessionOptions) error {
	return s.Opts
}

// SetCausalConsistency sets the value for the CausalConsistency field. If true, causal
// consistency will be enabled for the session. This option cannot be set to true if Snapshot
// is set to true. The default value is true unless Snapshot is set to true. See
// https://www.mongodb.com/docs/manual/core/read-isolation-consistency-recency/#sessions
// for more information.
func (s *SessionOptionsBuilder) SetCausalConsistency(b bool) *SessionOptionsBuilder {
	s.Opts = append(s.Opts, func(opts *SessionOptions) error {
		opts.CausalConsistency = &b
		return nil
	})
	return s
}

// SetDefaultTransactionOptions sets the value for the DefaultTransactionOptions field.
// Specifies the default options for transactions started in the session. If this object
// or any value on the object is nil, the client-level read concern, write concern,
// and/or read preference will be used to start the session.
func (s *SessionOptionsBuilder) SetDefaultTransactionOptions(dt *TransactionOptionsBuilder) *SessionOptionsBuilder {
	s.Opts = append(s.Opts, func(opts *SessionOptions) error {
		opts.DefaultTransactionOptions = dt
		return nil
	})
	return s
}

// SetSnapshot sets the value for the Snapshot field. If true, all read operations performed
// with this session will be read from the same snapshot. This option cannot be set to true
// if CausalConsistency is set to true. Transactions and write operations are not allowed on
// snapshot sessions and will error. The default value is false.
func (s *SessionOptionsBuilder) SetSnapshot(b bool) *SessionOptionsBuilder {
	s.Opts = append(s.Opts, func(opts *SessionOptions) error {
		opts.Snapshot = &b
		return nil
	})
	return s
}
