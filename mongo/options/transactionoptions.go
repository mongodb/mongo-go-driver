// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

// TransactionOptions represents arguments that can be used to configure a
// transaction.
//
// See corresponding setter methods for documentation.
type TransactionOptions struct {
	ReadConcern    *readconcern.ReadConcern
	ReadPreference *readpref.ReadPref
	WriteConcern   *writeconcern.WriteConcern
}

// TransactionOptionsBuilder contains arguments to configure count operations.
// Each option can be set through setter functions. See documentation for each
// setter function for an explanation of the option.
type TransactionOptionsBuilder struct {
	Opts []func(*TransactionOptions) error
}

// Transaction creates a new TransactionOptions instance.
func Transaction() *TransactionOptionsBuilder {
	return &TransactionOptionsBuilder{}
}

// List returns a list of TransactionOptions setter functions.
func (t *TransactionOptionsBuilder) List() []func(*TransactionOptions) error {
	return t.Opts
}

// SetReadConcern sets the value for the ReadConcern field. Specifies the read concern for operations
// in the transaction. The default value is nil, which means that the default read concern of the
// session used to start the transaction will be used.
func (t *TransactionOptionsBuilder) SetReadConcern(rc *readconcern.ReadConcern) *TransactionOptionsBuilder {
	t.Opts = append(t.Opts, func(opts *TransactionOptions) error {
		opts.ReadConcern = rc

		return nil
	})

	return t
}

// SetReadPreference sets the value for the ReadPreference field. Specifies the read preference for
// operations in the transaction. The default value is nil, which means that the default read
// preference of the session used to start the transaction will be used.
func (t *TransactionOptionsBuilder) SetReadPreference(rp *readpref.ReadPref) *TransactionOptionsBuilder {
	t.Opts = append(t.Opts, func(opts *TransactionOptions) error {
		opts.ReadPreference = rp

		return nil
	})

	return t
}

// SetWriteConcern sets the value for the WriteConcern field. Specifies the write concern for
// operations in the transaction. The default value is nil, which means that the default
// write concern of the session used to start the transaction will be used.
func (t *TransactionOptionsBuilder) SetWriteConcern(wc *writeconcern.WriteConcern) *TransactionOptionsBuilder {
	t.Opts = append(t.Opts, func(opts *TransactionOptions) error {
		opts.WriteConcern = wc

		return nil
	})

	return t
}
