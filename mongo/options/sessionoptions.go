// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"time"

	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

// DefaultCausalConsistency is the default value for the CausalConsistency option.
var DefaultCausalConsistency = true

// SessionArgs represents arguments that can be used to configure a Session.
type SessionArgs struct {
	// If true, causal consistency will be enabled for the session. This option cannot be set to true if Snapshot is
	// set to true. The default value is true unless Snapshot is set to true. See
	// https://www.mongodb.com/docs/manual/core/read-isolation-consistency-recency/#sessions for more information.
	CausalConsistency *bool

	// The default read concern for transactions started in the session. The default value is nil, which means that
	// the read concern of the client used to start the session will be used.
	DefaultReadConcern *readconcern.ReadConcern

	// The default read preference for transactions started in the session. The default value is nil, which means that
	// the read preference of the client used to start the session will be used.
	DefaultReadPreference *readpref.ReadPref

	// The default write concern for transactions started in the session. The default value is nil, which means that
	// the write concern of the client used to start the session will be used.
	DefaultWriteConcern *writeconcern.WriteConcern

	// The default maximum amount of time that a CommitTransaction operation executed in the session can run on the
	// server. The default value is nil, which means that that there is no time limit for execution.
	//
	// NOTE(benjirewis): DefaultMaxCommitTime will be deprecated in a future release. The more general Timeout option
	// may be used in its place to control the amount of time that a single operation can run before returning an
	// error. DefaultMaxCommitTime is ignored if Timeout is set on the client.
	DefaultMaxCommitTime *time.Duration

	// If true, all read operations performed with this session will be read from the same snapshot. This option cannot
	// be set to true if CausalConsistency is set to true. Transactions and write operations are not allowed on
	// snapshot sessions and will error. The default value is false.
	Snapshot *bool
}

// SessionOptions represents functional options that configure a SessionArgs.
type SessionOptions struct {
	Opts []func(*SessionArgs) error
}

// Session creates a new SessionOptions instance.
func Session() *SessionOptions {
	return &SessionOptions{}
}

// ArgsSetters returns a list of SessionArgs setter functions.
func (s *SessionOptions) ArgsSetters() []func(*SessionArgs) error {
	return s.Opts
}

// SetCausalConsistency sets the value for the CausalConsistency field.
func (s *SessionOptions) SetCausalConsistency(b bool) *SessionOptions {
	s.Opts = append(s.Opts, func(args *SessionArgs) error {
		args.CausalConsistency = &b
		return nil
	})
	return s
}

// SetDefaultReadConcern sets the value for the DefaultReadConcern field.
func (s *SessionOptions) SetDefaultReadConcern(rc *readconcern.ReadConcern) *SessionOptions {
	s.Opts = append(s.Opts, func(args *SessionArgs) error {
		args.DefaultReadConcern = rc
		return nil
	})
	return s
}

// SetDefaultReadPreference sets the value for the DefaultReadPreference field.
func (s *SessionOptions) SetDefaultReadPreference(rp *readpref.ReadPref) *SessionOptions {
	s.Opts = append(s.Opts, func(args *SessionArgs) error {
		args.DefaultReadPreference = rp
		return nil
	})
	return s
}

// SetDefaultWriteConcern sets the value for the DefaultWriteConcern field.
func (s *SessionOptions) SetDefaultWriteConcern(wc *writeconcern.WriteConcern) *SessionOptions {
	s.Opts = append(s.Opts, func(args *SessionArgs) error {
		args.DefaultWriteConcern = wc
		return nil
	})
	return s
}

// SetDefaultMaxCommitTime sets the value for the DefaultMaxCommitTime field.
//
// NOTE(benjirewis): DefaultMaxCommitTime will be deprecated in a future release. The more
// general Timeout option may be used in its place to control the amount of time that a
// single operation can run before returning an error. DefaultMaxCommitTime is ignored if
// Timeout is set on the client.
func (s *SessionOptions) SetDefaultMaxCommitTime(mct *time.Duration) *SessionOptions {
	s.Opts = append(s.Opts, func(args *SessionArgs) error {
		args.DefaultMaxCommitTime = mct
		return nil
	})
	return s
}

// SetSnapshot sets the value for the Snapshot field.
func (s *SessionOptions) SetSnapshot(b bool) *SessionOptions {
	s.Opts = append(s.Opts, func(args *SessionArgs) error {
		args.Snapshot = &b
		return nil
	})
	return s
}
