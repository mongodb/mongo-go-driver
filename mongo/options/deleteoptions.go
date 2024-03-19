// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// DeleteArgs represents arguments that can be used to configure DeleteOne and
// DeleteMany operations.
type DeleteArgs struct {
	// Specifies a collation to use for string comparisons during the operation. This option is only valid for MongoDB
	// versions >= 3.4. For previous server versions, the driver will return an error if this option is used. The
	// default value is nil, which means the default collation of the collection will be used.
	Collation *Collation

	// A string or document that will be included in server logs, profiling logs, and currentOp queries to help trace
	// the operation.  The default value is nil, which means that no comment will be included in the logs.
	Comment interface{}

	// The index to use for the operation. This should either be the index name as a string or the index specification
	// as a document. This option is only valid for MongoDB versions >= 4.4. Server versions >= 3.4 will return an error
	// if this option is specified. For server versions < 3.4, the driver will return a client-side error if this option
	// is specified. The driver will return an error if this option is specified during an unacknowledged write
	// operation. The driver will return an error if the hint parameter is a multi-key map. The default value is nil,
	// which means that no hint will be sent.
	Hint interface{}

	// Specifies parameters for the delete expression. This option is only valid for MongoDB versions >= 5.0. Older
	// servers will report an error for using this option. This must be a document mapping parameter names to values.
	// Values must be constant or closed expressions that do not reference document fields. Parameters can then be
	// accessed as variables in an aggregate expression context (e.g. "$$var").
	Let interface{}
}

// DeleteOptions contains options to configure delete operations. Each option
// can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type DeleteOptions struct {
	Opts []func(*DeleteArgs) error
}

// Delete creates a new DeleteOptions instance.
func Delete() *DeleteOptions {
	return &DeleteOptions{}
}

// ArgsSetters returns a list of DeleteArgs setter functions.
func (do *DeleteOptions) ArgsSetters() []func(*DeleteArgs) error {
	return do.Opts
}

// SetCollation sets the value for the Collation field.
func (do *DeleteOptions) SetCollation(c *Collation) *DeleteOptions {
	do.Opts = append(do.Opts, func(args *DeleteArgs) error {
		args.Collation = c

		return nil
	})

	return do
}

// SetComment sets the value for the Comment field.
func (do *DeleteOptions) SetComment(comment interface{}) *DeleteOptions {
	do.Opts = append(do.Opts, func(args *DeleteArgs) error {
		args.Comment = comment

		return nil
	})

	return do
}

// SetHint sets the value for the Hint field.
func (do *DeleteOptions) SetHint(hint interface{}) *DeleteOptions {
	do.Opts = append(do.Opts, func(args *DeleteArgs) error {
		args.Hint = hint

		return nil
	})

	return do
}

// SetLet sets the value for the Let field.
func (do *DeleteOptions) SetLet(let interface{}) *DeleteOptions {
	do.Opts = append(do.Opts, func(args *DeleteArgs) error {
		args.Let = let

		return nil
	})

	return do
}
