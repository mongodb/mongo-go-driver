// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// DistinctOptions represents arguments that can be used to configure a Distinct
// operation.
type DistinctOptions struct {
	// Specifies a collation to use for string comparisons during the operation. This option is only valid for MongoDB
	// versions >= 3.4. For previous server versions, the driver will return an error if this option is used. The
	// default value is nil, which means the default collation of the collection will be used.
	Collation *Collation

	// A string or document that will be included in server logs, profiling logs, and currentOp queries to help trace
	// the operation. The default value is nil, which means that no comment will be included in the logs.
	Comment interface{}
}

// DistinctOptionsBuilder contains options to configure distinct operations. Each
// option can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type DistinctOptionsBuilder struct {
	Opts []func(*DistinctOptions) error
}

// Distinct creates a new DistinctOptions instance.
func Distinct() *DistinctOptionsBuilder {
	return &DistinctOptionsBuilder{}
}

// List returns a list of DistinctArg setter functions.
func (do *DistinctOptionsBuilder) List() []func(*DistinctOptions) error {
	return do.Opts
}

// SetCollation sets the value for the Collation field.
func (do *DistinctOptionsBuilder) SetCollation(c *Collation) *DistinctOptionsBuilder {
	do.Opts = append(do.Opts, func(opts *DistinctOptions) error {
		opts.Collation = c

		return nil
	})

	return do
}

// SetComment sets the value for the Comment field.
func (do *DistinctOptionsBuilder) SetComment(comment interface{}) *DistinctOptionsBuilder {
	do.Opts = append(do.Opts, func(opts *DistinctOptions) error {
		opts.Comment = comment

		return nil
	})

	return do
}
