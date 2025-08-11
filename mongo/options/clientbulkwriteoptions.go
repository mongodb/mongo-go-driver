// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"go.mongodb.org/mongo-driver/v2/internal/optionsutil"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

// ClientBulkWriteOptions represents options that can be used to configure a client-level BulkWrite operation.
//
// See corresponding setter methods for documentation.
type ClientBulkWriteOptions struct {
	BypassDocumentValidation *bool
	Comment                  any
	Ordered                  *bool
	Let                      any
	WriteConcern             *writeconcern.WriteConcern
	VerboseResults           *bool

	// Deprecated: This option is for internal use only and should not be set. It may be changed or removed in any
	// release.
	Internal optionsutil.Options
}

// ClientBulkWriteOptionsBuilder contains options to configure client-level bulk
// write operations.
// Each option can be set through setter functions. See documentation for each
// setter function for an explanation of the option.
type ClientBulkWriteOptionsBuilder struct {
	Opts []func(*ClientBulkWriteOptions) error
}

// ClientBulkWrite creates a new *ClientBulkWriteOptions instance.
func ClientBulkWrite() *ClientBulkWriteOptionsBuilder {
	opts := &ClientBulkWriteOptionsBuilder{}
	opts = opts.SetOrdered(DefaultOrdered)

	return opts
}

// List returns a list of ClientBulkWriteOptions setter functions.
func (b *ClientBulkWriteOptionsBuilder) List() []func(*ClientBulkWriteOptions) error {
	return b.Opts
}

// SetComment sets the value for the Comment field. Specifies a string or document that will be included in
// server logs, profiling logs, and currentOp queries to help tracethe operation.  The default value is nil,
// which means that no comment will be included in the logs.
func (b *ClientBulkWriteOptionsBuilder) SetComment(comment any) *ClientBulkWriteOptionsBuilder {
	b.Opts = append(b.Opts, func(opts *ClientBulkWriteOptions) error {
		opts.Comment = comment

		return nil
	})

	return b
}

// SetOrdered sets the value for the Ordered field. If true, no writes will be executed after one fails.
// The default value is true.
func (b *ClientBulkWriteOptionsBuilder) SetOrdered(ordered bool) *ClientBulkWriteOptionsBuilder {
	b.Opts = append(b.Opts, func(opts *ClientBulkWriteOptions) error {
		opts.Ordered = &ordered

		return nil
	})

	return b
}

// SetBypassDocumentValidation sets the value for the BypassDocumentValidation field. If true, writes
// executed as part of the operation will opt out of document-level validation on the server. The default
// value is false. See https://www.mongodb.com/docs/manual/core/schema-validation/ for more information
// about document validation.
func (b *ClientBulkWriteOptionsBuilder) SetBypassDocumentValidation(bypass bool) *ClientBulkWriteOptionsBuilder {
	b.Opts = append(b.Opts, func(opts *ClientBulkWriteOptions) error {
		opts.BypassDocumentValidation = &bypass

		return nil
	})

	return b
}

// SetLet sets the value for the Let field. Let specifies parameters for all update and delete commands in the BulkWrite.
// This must be a document mapping parameter names to values. Values must be constant or closed expressions that do not
// reference document fields. Parameters can then be accessed as variables in an aggregate expression context (e.g. "$$var").
func (b *ClientBulkWriteOptionsBuilder) SetLet(let any) *ClientBulkWriteOptionsBuilder {
	b.Opts = append(b.Opts, func(opts *ClientBulkWriteOptions) error {
		opts.Let = &let

		return nil
	})

	return b
}

// SetWriteConcern sets the value for the WriteConcern field. Specifies the write concern for
// operations in the transaction. The default value is nil, which means that the default
// write concern of the session used to start the transaction will be used.
func (b *ClientBulkWriteOptionsBuilder) SetWriteConcern(wc *writeconcern.WriteConcern) *ClientBulkWriteOptionsBuilder {
	b.Opts = append(b.Opts, func(opts *ClientBulkWriteOptions) error {
		opts.WriteConcern = wc

		return nil
	})

	return b
}

// SetVerboseResults sets the value for the VerboseResults field. Specifies whether detailed
// results for each successful operation should be included in the returned BulkWriteResult.
// The defaults value is false.
func (b *ClientBulkWriteOptionsBuilder) SetVerboseResults(verboseResults bool) *ClientBulkWriteOptionsBuilder {
	b.Opts = append(b.Opts, func(opts *ClientBulkWriteOptions) error {
		opts.VerboseResults = &verboseResults

		return nil
	})

	return b
}
