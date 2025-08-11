// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import "go.mongodb.org/mongo-driver/v2/internal/optionsutil"

// DeleteOneOptions represents arguments that can be used to configure DeleteOne
// operations.
//
// See corresponding setter methods for documentation.
type DeleteOneOptions struct {
	Collation *Collation
	Comment   any
	Hint      any
	Let       any

	// Deprecated: This option is for internal use only and should not be set. It may be changed or removed in any
	// release.
	Internal optionsutil.Options
}

// DeleteOneOptionsBuilder contains options to configure DeleteOne operations. Each
// option can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type DeleteOneOptionsBuilder struct {
	Opts []func(*DeleteOneOptions) error
}

// DeleteOne creates a new DeleteOneOptions instance.
func DeleteOne() *DeleteOneOptionsBuilder {
	return &DeleteOneOptionsBuilder{}
}

// List returns a list of DeleteOneOptions setter functions.
func (do *DeleteOneOptionsBuilder) List() []func(*DeleteOneOptions) error {
	return do.Opts
}

// SetCollation sets the value for the Collation field. Specifies a collation to
// use for string comparisons during the operation. The default value is nil,
// which means the default collation of the collection will be used.
func (do *DeleteOneOptionsBuilder) SetCollation(c *Collation) *DeleteOneOptionsBuilder {
	do.Opts = append(do.Opts, func(opts *DeleteOneOptions) error {
		opts.Collation = c

		return nil
	})

	return do
}

// SetComment sets the value for the Comment field. Specifies a string or document that will
// be included in server logs, profiling logs, and currentOp queries to help trace the operation.
// The default value is nil, which means that no comment will be included in the logs.
func (do *DeleteOneOptionsBuilder) SetComment(comment any) *DeleteOneOptionsBuilder {
	do.Opts = append(do.Opts, func(opts *DeleteOneOptions) error {
		opts.Comment = comment

		return nil
	})

	return do
}

// SetHint sets the value for the Hint field. Specifies the index to use for the
// operation. This should either be the index name as a string or the index
// specification as a document. This option is only valid for MongoDB versions
// >= 4.4. Server versions < 4.4 will return an error if this option is
// specified. The driver will return an error if this option is specified during
// an unacknowledged write operation. The driver will return an error if the
// hint parameter is a multi-key map. The default value is nil, which means that
// no hint will be sent.
func (do *DeleteOneOptionsBuilder) SetHint(hint any) *DeleteOneOptionsBuilder {
	do.Opts = append(do.Opts, func(opts *DeleteOneOptions) error {
		opts.Hint = hint

		return nil
	})

	return do
}

// SetLet sets the value for the Let field. Specifies parameters for the delete expression. This
// option is only valid for MongoDB versions >= 5.0. Older servers will report an error for using
// this option. This must be a document mapping parameter names to values. Values must be constant
// or closed expressions that do not reference document fields. Parameters can then be accessed as
// variables in an aggregate expression context (e.g. "$$var").
func (do *DeleteOneOptionsBuilder) SetLet(let any) *DeleteOneOptionsBuilder {
	do.Opts = append(do.Opts, func(opts *DeleteOneOptions) error {
		opts.Let = let

		return nil
	})

	return do
}

// DeleteManyOptions represents arguments that can be used to configure DeleteMany
// operations.
//
// See corresponding setter methods for documentation.
type DeleteManyOptions struct {
	Collation *Collation
	Comment   any
	Hint      any
	Let       any

	// Deprecated: This option is for internal use only and should not be set. It may be changed or removed in any
	// release.
	Internal optionsutil.Options
}

// DeleteManyOptionsBuilder contains options to configure DeleteMany operations.
// Each option can be set through setter functions. See documentation for each
// setter function for an explanation of the option.
type DeleteManyOptionsBuilder struct {
	Opts []func(*DeleteManyOptions) error
}

// DeleteMany creates a new DeleteManyOptions instance.
func DeleteMany() *DeleteManyOptionsBuilder {
	return &DeleteManyOptionsBuilder{}
}

// List returns a list of DeleteOneOptions setter functions.
func (do *DeleteManyOptionsBuilder) List() []func(*DeleteManyOptions) error {
	return do.Opts
}

// SetCollation sets the value for the Collation field. Specifies a collation to
// use for string comparisons during the operation. The default value is nil,
// which means the default collation of the collection will be used.
func (do *DeleteManyOptionsBuilder) SetCollation(c *Collation) *DeleteManyOptionsBuilder {
	do.Opts = append(do.Opts, func(opts *DeleteManyOptions) error {
		opts.Collation = c

		return nil
	})

	return do
}

// SetComment sets the value for the Comment field. Specifies a string or document that will be
// included in server logs, profiling logs, and currentOp queries to help trace the operation.
// The default value is nil, which means that no comment will be included in the logs.
func (do *DeleteManyOptionsBuilder) SetComment(comment any) *DeleteManyOptionsBuilder {
	do.Opts = append(do.Opts, func(opts *DeleteManyOptions) error {
		opts.Comment = comment

		return nil
	})

	return do
}

// SetHint sets the value for the Hint field. Specifies the index to use for the
// operation. This should either be the index name as a string or the index
// specification as a document. This option is only valid for MongoDB versions
// >= 4.4. Server versions < 4.4 will return an error if this option is
// specified. The driver will return an error if this option is specified during
// an unacknowledged write operation. The driver will return an error if the
// hint parameter is a multi-key map. The default value is nil, which means that
// no hint will be sent.
func (do *DeleteManyOptionsBuilder) SetHint(hint any) *DeleteManyOptionsBuilder {
	do.Opts = append(do.Opts, func(opts *DeleteManyOptions) error {
		opts.Hint = hint

		return nil
	})

	return do
}

// SetLet sets the value for the Let field. Specifies parameters for the delete expression.
// This option is only valid for MongoDB versions >= 5.0. Older servers will report an error
// for using this option. This must be a document mapping parameter names to values. Values
// must be constant or closed expressions that do not reference document fields. Parameters
// can then be accessed as variables in an aggregate expression context (e.g. "$$var").
func (do *DeleteManyOptionsBuilder) SetLet(let any) *DeleteManyOptionsBuilder {
	do.Opts = append(do.Opts, func(opts *DeleteManyOptions) error {
		opts.Let = let

		return nil
	})

	return do
}
