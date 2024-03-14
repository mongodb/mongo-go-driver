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

// TransactionArgs represents arguments that can be used to configure a
// transaction.
type TransactionArgs struct {
	// The read concern for operations in the transaction. The default value is nil, which means that the default
	// read concern of the session used to start the transaction will be used.
	ReadConcern *readconcern.ReadConcern

	// The read preference for operations in the transaction. The default value is nil, which means that the default
	// read preference of the session used to start the transaction will be used.
	ReadPreference *readpref.ReadPref

	// The write concern for operations in the transaction. The default value is nil, which means that the default
	// write concern of the session used to start the transaction will be used.
	WriteConcern *writeconcern.WriteConcern

	// The default maximum amount of time that a CommitTransaction operation executed in the session can run on the
	// server. The default value is nil, meaning that there is no time limit for execution.

	// The maximum amount of time that a CommitTransaction operation can executed in the transaction can run on the
	// server. The default value is nil, which means that the default maximum commit time of the session used to
	// start the transaction will be used.
	//
	// NOTE(benjirewis): MaxCommitTime will be deprecated in a future release. The more general Timeout option may
	// be used in its place to control the amount of time that a single operation can run before returning an error.
	// MaxCommitTime is ignored if Timeout is set on the client.
	MaxCommitTime *time.Duration
}

// TransactionOptions contains arguments to configure count operations. Each
// option can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type TransactionOptions struct {
	Opts []func(*TransactionArgs) error
}

// Transaction creates a new TransactionOptions instance.
func Transaction() *TransactionOptions {
	return &TransactionOptions{}
}

// ArgsSetters returns a list of TransactionArgs setter functions.
func (t *TransactionOptions) ArgsSetters() []func(*TransactionArgs) error {
	return t.Opts
}

// SetReadConcern sets the value for the ReadConcern field.
func (t *TransactionOptions) SetReadConcern(rc *readconcern.ReadConcern) *TransactionOptions {
	t.Opts = append(t.Opts, func(args *TransactionArgs) error {
		args.ReadConcern = rc

		return nil
	})

	return t
}

// SetReadPreference sets the value for the ReadPreference field.
func (t *TransactionOptions) SetReadPreference(rp *readpref.ReadPref) *TransactionOptions {
	t.Opts = append(t.Opts, func(args *TransactionArgs) error {
		args.ReadPreference = rp

		return nil
	})

	return t
}

// SetWriteConcern sets the value for the WriteConcern field.
func (t *TransactionOptions) SetWriteConcern(wc *writeconcern.WriteConcern) *TransactionOptions {
	t.Opts = append(t.Opts, func(args *TransactionArgs) error {
		args.WriteConcern = wc

		return nil
	})

	return t
}

// SetMaxCommitTime sets the value for the MaxCommitTime field.
//
// NOTE(benjirewis): MaxCommitTime will be deprecated in a future release. The more general Timeout
// option may be used in its place to control the amount of time that a single operation can run before
// returning an error. MaxCommitTime is ignored if Timeout is set on the client.
func (t *TransactionOptions) SetMaxCommitTime(mct *time.Duration) *TransactionOptions {
	t.Opts = append(t.Opts, func(args *TransactionArgs) error {
		args.MaxCommitTime = mct

		return nil
	})

	return t
}
