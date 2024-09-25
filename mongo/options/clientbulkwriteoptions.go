// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

// ClientBulkWriteOptions represents options that can be used to configure a client-level BulkWrite operation.
type ClientBulkWriteOptions struct {
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

	// The write concern to use for this bulk write.
	WriteConcern *writeconcern.WriteConcern

	// Whether detailed results for each successful operation should be included in the returned BulkWriteResult.
	VerboseResults *bool
}

// ClientBulkWrite creates a new *ClientBulkWriteOptions instance.
func ClientBulkWrite() *ClientBulkWriteOptions {
	return &ClientBulkWriteOptions{
		Ordered: &DefaultOrdered,
	}
}

// SetComment sets the value for the Comment field.
func (b *ClientBulkWriteOptions) SetComment(comment interface{}) *ClientBulkWriteOptions {
	b.Comment = comment
	return b
}

// SetOrdered sets the value for the Ordered field.
func (b *ClientBulkWriteOptions) SetOrdered(ordered bool) *ClientBulkWriteOptions {
	b.Ordered = &ordered
	return b
}

// SetBypassDocumentValidation sets the value for the BypassDocumentValidation field.
func (b *ClientBulkWriteOptions) SetBypassDocumentValidation(bypass bool) *ClientBulkWriteOptions {
	b.BypassDocumentValidation = &bypass
	return b
}

// SetLet sets the value for the Let field. Let specifies parameters for all update and delete commands in the BulkWrite.
// This option is only valid for MongoDB versions >= 5.0. Older servers will report an error for using this option.
// This must be a document mapping parameter names to values. Values must be constant or closed expressions that do not
// reference document fields. Parameters can then be accessed as variables in an aggregate expression context (e.g. "$$var").
func (b *ClientBulkWriteOptions) SetLet(let interface{}) *ClientBulkWriteOptions {
	b.Let = &let
	return b
}

// SetWriteConcern sets the value for the WriteConcern field.
func (b *ClientBulkWriteOptions) SetWriteConcern(wc *writeconcern.WriteConcern) *ClientBulkWriteOptions {
	b.WriteConcern = wc
	return b
}

// SetVerboseResults sets the value for the VerboseResults field.
func (b *ClientBulkWriteOptions) SetVerboseResults(verboseResults bool) *ClientBulkWriteOptions {
	b.VerboseResults = &verboseResults
	return b
}

// MergeClientBulkWriteOptions combines the given BulkWriteOptions instances into a single BulkWriteOptions in a last-one-wins
// fashion.
//
// Deprecated: Merging options structs will not be supported in Go Driver 2.0. Users should create a
// single options struct instead.
func MergeClientBulkWriteOptions(opts ...*ClientBulkWriteOptions) *ClientBulkWriteOptions {
	b := ClientBulkWrite()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.Comment != nil {
			b.Comment = opt.Comment
		}
		if opt.Ordered != nil {
			b.Ordered = opt.Ordered
		}
		if opt.BypassDocumentValidation != nil {
			b.BypassDocumentValidation = opt.BypassDocumentValidation
		}
		if opt.Let != nil {
			b.Let = opt.Let
		}
		if opt.WriteConcern != nil {
			b.WriteConcern = opt.WriteConcern
		}
		if opt.VerboseResults != nil {
			b.VerboseResults = opt.VerboseResults
		}
	}

	return b
}
