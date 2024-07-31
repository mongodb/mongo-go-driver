// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// UpdateOptions represents arguments that can be used to configure UpdateOne and
// UpdateMany operations.
type UpdateOptions struct {
	// A set of filters specifying to which array elements an update should apply. This option is only valid for MongoDB
	// versions >= 3.6. For previous server versions, the driver will return an error if this option is used. The
	// default value is nil, which means the update will apply to all array elements.
	ArrayFilters []interface{}

	// If true, writes executed as part of the operation will opt out of document-level validation on the server. This
	// option is valid for MongoDB versions >= 3.2 and is ignored for previous server versions. The default value is
	// false. See https://www.mongodb.com/docs/manual/core/schema-validation/ for more information about document
	// validation.
	BypassDocumentValidation *bool

	// Specifies a collation to use for string comparisons during the operation. This option is only valid for MongoDB
	// versions >= 3.4. For previous server versions, the driver will return an error if this option is used. The
	// default value is nil, which means the default collation of the collection will be used.
	Collation *Collation

	// A string or document that will be included in server logs, profiling logs, and currentOp queries to help trace
	// the operation.  The default value is nil, which means that no comment will be included in the logs.
	Comment interface{}

	// The index to use for the operation. This should either be the index name as a string or the index specification
	// as a document. This option is only valid for MongoDB versions >= 4.2. Server versions >= 3.4 will return an error
	// if this option is specified. For server versions < 3.4, the driver will return a client-side error if this option
	// is specified. The driver will return an error if this option is specified during an unacknowledged write
	// operation. The driver will return an error if the hint parameter is a multi-key map. The default value is nil,
	// which means that no hint will be sent.
	Hint interface{}

	// If true, a new document will be inserted if the filter does not match any documents in the collection. The
	// default value is false.
	Upsert *bool

	// Specifies parameters for the update expression. This option is only valid for MongoDB versions >= 5.0. Older
	// servers will report an error for using this option. This must be a document mapping parameter names to values.
	// Values must be constant or closed expressions that do not reference document fields. Parameters can then be
	// accessed as variables in an aggregate expression context (e.g. "$$var").
	Let interface{}
}

// UpdateOptionsBuilder contains options to configure update operations. Each
// option can be set through setter functions. See documentation for each setter
// function for an explanation of the option.
type UpdateOptionsBuilder struct {
	Opts []func(*UpdateOptions) error
}

// Update creates a new UpdateOptions instance.
func Update() *UpdateOptionsBuilder {
	return &UpdateOptionsBuilder{}
}

// List returns a list of UpdateOptions setter functions.
func (uo *UpdateOptionsBuilder) List() []func(*UpdateOptions) error {
	return uo.Opts
}

// SetArrayFilters sets the value for the ArrayFilters field.
func (uo *UpdateOptionsBuilder) SetArrayFilters(af []interface{}) *UpdateOptionsBuilder {
	uo.Opts = append(uo.Opts, func(opts *UpdateOptions) error {
		opts.ArrayFilters = af

		return nil
	})

	return uo
}

// SetBypassDocumentValidation sets the value for the BypassDocumentValidation field.
func (uo *UpdateOptionsBuilder) SetBypassDocumentValidation(b bool) *UpdateOptionsBuilder {
	uo.Opts = append(uo.Opts, func(opts *UpdateOptions) error {
		opts.BypassDocumentValidation = &b

		return nil
	})

	return uo
}

// SetCollation sets the value for the Collation field.
func (uo *UpdateOptionsBuilder) SetCollation(c *Collation) *UpdateOptionsBuilder {
	uo.Opts = append(uo.Opts, func(opts *UpdateOptions) error {
		opts.Collation = c

		return nil
	})

	return uo
}

// SetComment sets the value for the Comment field.
func (uo *UpdateOptionsBuilder) SetComment(comment interface{}) *UpdateOptionsBuilder {
	uo.Opts = append(uo.Opts, func(opts *UpdateOptions) error {
		opts.Comment = comment

		return nil
	})

	return uo
}

// SetHint sets the value for the Hint field.
func (uo *UpdateOptionsBuilder) SetHint(h interface{}) *UpdateOptionsBuilder {
	uo.Opts = append(uo.Opts, func(opts *UpdateOptions) error {
		opts.Hint = h

		return nil
	})

	return uo
}

// SetUpsert sets the value for the Upsert field.
func (uo *UpdateOptionsBuilder) SetUpsert(b bool) *UpdateOptionsBuilder {
	uo.Opts = append(uo.Opts, func(opts *UpdateOptions) error {
		opts.Upsert = &b

		return nil
	})

	return uo
}

// SetLet sets the value for the Let field.
func (uo *UpdateOptionsBuilder) SetLet(l interface{}) *UpdateOptionsBuilder {
	uo.Opts = append(uo.Opts, func(opts *UpdateOptions) error {
		opts.Let = l

		return nil
	})

	return uo
}
