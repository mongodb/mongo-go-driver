// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// DefaultOrdered is the default value for the Ordered option in BulkWriteOptions.
var DefaultOrdered = true

// BulkWriteOptions represents arguments that can be used to configure a
// BulkWrite operation.
type BulkWriteOptions struct {
	// If true, writes executed as part of the operation will opt out of document-level validation on the server. This
	// option is valid for MongoDB versions >= 3.2 and is ignored for previous server versions. The default value is
	// false. See https://www.mongodb.com/docs/manual/core/schema-validation/ for more information about document
	// validation.
	BypassDocumentValidation *bool

	// A string or document that will be included in server logs, profiling logs, and currentOp queries to help trace
	// the operation.  The default value is nil, which means that no comment will be included in the logs.
	Comment interface{}

	// If true, no writes will be executed after one fails. The default value is true.
	Ordered *bool

	// Specifies parameters for all update and delete commands in the BulkWrite. This option is only valid for MongoDB
	// versions >= 5.0. Older servers will report an error for using this option. This must be a document mapping
	// parameter names to values. Values must be constant or closed expressions that do not reference document fields.
	// Parameters can then be accessed as variables in an aggregate expression context (e.g. "$$var").
	Let interface{}
}

// BulkWriteOptionsBuilder contains options to configure bulk write operations.
// Each option can be set through setter functions. See documentation for each
// setter function for an explanation of the option.
type BulkWriteOptionsBuilder struct {
	Opts []func(*BulkWriteOptions) error
}

// BulkWrite creates a new *BulkWriteOptions instance.
func BulkWrite() *BulkWriteOptionsBuilder {
	opts := &BulkWriteOptionsBuilder{}
	opts = opts.SetOrdered(DefaultOrdered)

	return opts
}

// List returns a list of BulkWriteOptions setter functions.
func (b *BulkWriteOptionsBuilder) List() []func(*BulkWriteOptions) error {
	return b.Opts
}

// SetComment sets the value for the Comment field.
func (b *BulkWriteOptionsBuilder) SetComment(comment interface{}) *BulkWriteOptionsBuilder {
	b.Opts = append(b.Opts, func(opts *BulkWriteOptions) error {
		opts.Comment = comment

		return nil
	})

	return b
}

// SetOrdered sets the value for the Ordered field.
func (b *BulkWriteOptionsBuilder) SetOrdered(ordered bool) *BulkWriteOptionsBuilder {
	b.Opts = append(b.Opts, func(opts *BulkWriteOptions) error {
		opts.Ordered = &ordered

		return nil
	})

	return b
}

// SetBypassDocumentValidation sets the value for the BypassDocumentValidation field.
func (b *BulkWriteOptionsBuilder) SetBypassDocumentValidation(bypass bool) *BulkWriteOptionsBuilder {
	b.Opts = append(b.Opts, func(opts *BulkWriteOptions) error {
		opts.BypassDocumentValidation = &bypass

		return nil
	})

	return b
}

// SetLet sets the value for the Let field. Let specifies parameters for all update and delete commands in the BulkWrite.
// This option is only valid for MongoDB versions >= 5.0. Older servers will report an error for using this option.
// This must be a document mapping parameter names to values. Values must be constant or closed expressions that do not
// reference document fields. Parameters can then be accessed as variables in an aggregate expression context (e.g. "$$var").
func (b *BulkWriteOptionsBuilder) SetLet(let interface{}) *BulkWriteOptionsBuilder {
	b.Opts = append(b.Opts, func(opts *BulkWriteOptions) error {
		opts.Let = &let

		return nil
	})

	return b
}
