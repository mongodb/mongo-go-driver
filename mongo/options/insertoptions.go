// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// InsertOneArgs represents arguments that can be used to configure an InsertOne
// operation.
type InsertOneArgs struct {
	// If true, writes executed as part of the operation will opt out of document-level validation on the server. This
	// option is valid for MongoDB versions >= 3.2 and is ignored for previous server versions. The default value is
	// false. See https://www.mongodb.com/docs/manual/core/schema-validation/ for more information about document
	// validation.
	BypassDocumentValidation *bool

	// A string or document that will be included in server logs, profiling logs, and currentOp queries to help trace
	// the operation.  The default value is nil, which means that no comment will be included in the logs.
	Comment interface{}
}

// InsertOneOptions represents functional options that configure an InsertOneArgs.
type InsertOneOptions struct {
	Opts []func(*InsertOneArgs) error
}

// InsertOne creates a new InsertOneOptions instance.
func InsertOne() *InsertOneOptions {
	return &InsertOneOptions{}
}

// ArgsSetters returns a list of InsertOneArgs setter functions.
func (ioo *InsertOneOptions) ArgsSetters() []func(*InsertOneArgs) error {
	return ioo.Opts
}

// SetBypassDocumentValidation sets the value for the BypassDocumentValidation field.
func (ioo *InsertOneOptions) SetBypassDocumentValidation(b bool) *InsertOneOptions {
	ioo.Opts = append(ioo.Opts, func(args *InsertOneArgs) error {
		args.BypassDocumentValidation = &b
		return nil
	})
	return ioo
}

// SetComment sets the value for the Comment field.
func (ioo *InsertOneOptions) SetComment(comment interface{}) *InsertOneOptions {
	ioo.Opts = append(ioo.Opts, func(args *InsertOneArgs) error {
		args.Comment = &comment
		return nil
	})
	return ioo
}

// InsertManyArgs represents arguments that can be used to configure an
// InsertMany operation.
type InsertManyArgs struct {
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
}

// InsertManyOptions contains options to configure insert operations. Each
// option can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type InsertManyOptions struct {
	Opts []func(*InsertManyArgs) error
}

// InsertMany creates a new InsertManyOptions instance.
func InsertMany() *InsertManyOptions {
	opts := &InsertManyOptions{}
	opts.SetOrdered(DefaultOrdered)

	return opts
}

// ArgsSetters returns a list of InsertManyArgs setter functions.
func (imo *InsertManyOptions) ArgsSetters() []func(*InsertManyArgs) error {
	return imo.Opts
}

// SetBypassDocumentValidation sets the value for the BypassDocumentValidation field.
func (imo *InsertManyOptions) SetBypassDocumentValidation(b bool) *InsertManyOptions {
	imo.Opts = append(imo.Opts, func(args *InsertManyArgs) error {
		args.BypassDocumentValidation = &b

		return nil
	})

	return imo
}

// SetComment sets the value for the Comment field.
func (imo *InsertManyOptions) SetComment(comment interface{}) *InsertManyOptions {
	imo.Opts = append(imo.Opts, func(args *InsertManyArgs) error {
		args.Comment = comment

		return nil
	})

	return imo
}

// SetOrdered sets the value for the Ordered field.
func (imo *InsertManyOptions) SetOrdered(b bool) *InsertManyOptions {
	imo.Opts = append(imo.Opts, func(args *InsertManyArgs) error {
		args.Ordered = &b

		return nil
	})

	return imo
}
