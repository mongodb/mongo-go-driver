// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// InsertOneOptions represents all possible options to the insertOne()
type InsertOneOptions struct {
	BypassDocumentValidation *bool // If true, allows the write to opt-out of document level validation
}

// InsertOne returns a pointer to a new InsertOneOptions
func InsertOne() *InsertOneOptions {
	return &InsertOneOptions{}
}

// SetBypassDocumentValidation allows the write to opt-out of document level
// validation. This only applies when the $out stage is specified
func (ioo *InsertOneOptions) SetBypassDocumentValidation(b bool) *InsertOneOptions {
	ioo.BypassDocumentValidation = &b
	return ioo
}

// ToInsertOneOptions combines the argued InsertOneOptions into a single InsertOneOptions in a last-one-wins fashion
func ToInsertOneOptions(opts ...*InsertOneOptions) *InsertOneOptions {
	ioOpts := InsertOne()
	for _, ioo := range opts {
		if ioo.BypassDocumentValidation != nil {
			ioOpts.BypassDocumentValidation = ioo.BypassDocumentValidation
		}
	}

	return ioOpts
}

// InsertManyOptions represents all possible options to the insertMany()
type InsertManyOptions struct {
	BypassDocumentValidation *bool // If true, allows the write to opt-out of document level validation
	Ordered                  *bool // If true, when an insert fails, return without performing the remaining writes; if false, when a write fails, continue with the remaining writes, if any
}

// InsertMany returns a pointer to a new InsertManyOptions
func InsertMany() *InsertManyOptions {
	return &InsertManyOptions{}
}

// SetBypassDocumentValidation allows the write to opt-out of document level
// validation. This only applies when the $out stage is specified
func (imo *InsertManyOptions) SetBypassDocumentValidation(b bool) *InsertManyOptions {
	imo.BypassDocumentValidation = &b
	return imo
}

// SetOrdered sets whether to continue performing remaining writes when an
// insert fails
func (imo *InsertManyOptions) SetOrdered(b bool) *InsertManyOptions {
	imo.Ordered = &b
	return imo
}

// ToInsertManyOptions combines the argued InsertManyOptions into a single InsertManyOptions in a last-one-wins fashion
func ToInsertManyOptions(opts ...*InsertManyOptions) *InsertManyOptions {
	imOpts := InsertMany()
	for _, imo := range opts {
		if imo.BypassDocumentValidation != nil {
			imOpts.BypassDocumentValidation = imo.BypassDocumentValidation
		}
		if imo.Ordered != nil {
			imOpts.Ordered = imo.Ordered
		}
	}

	return imOpts
}
