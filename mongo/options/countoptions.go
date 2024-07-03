// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// CountOptions represents options that can be used to configure a CountDocuments operation.
type CountOptions struct {
	// Specifies a collation to use for string comparisons during the operation. This option is only valid for MongoDB
	// versions >= 3.4. For previous server versions, the driver will return an error if this option is used. The
	// default value is nil, which means the default collation of the collection will be used.
	Collation *Collation

	// A string or document that will be included in server logs, profiling logs, and currentOp queries to help trace
	// the operation.  The default is nil, which means that no comment will be included in the logs.
	Comment interface{}

	// The index to use for the aggregation. This should either be the index name as a string or the index specification
	// as a document. The driver will return an error if the hint parameter is a multi-key map. The default value is nil,
	// which means that no hint will be sent.
	Hint interface{}

	// The maximum number of documents to count. The default value is 0, which means that there is no limit and all
	// documents matching the filter will be counted.
	Limit *int64

	// The number of documents to skip before counting. The default value is 0.
	Skip *int64
}

// Count creates a new CountOptions instance.
func Count() *CountOptions {
	return &CountOptions{}
}

// SetCollation sets the value for the Collation field.
func (co *CountOptions) SetCollation(c *Collation) *CountOptions {
	co.Collation = c
	return co
}

// SetComment sets the value for the Comment field.
func (co *CountOptions) SetComment(comment interface{}) *CountOptions {
	co.Comment = comment
	return co
}

// SetHint sets the value for the Hint field.
func (co *CountOptions) SetHint(h interface{}) *CountOptions {
	co.Hint = h
	return co
}

// SetLimit sets the value for the Limit field.
func (co *CountOptions) SetLimit(i int64) *CountOptions {
	co.Limit = &i
	return co
}

// SetSkip sets the value for the Skip field.
func (co *CountOptions) SetSkip(i int64) *CountOptions {
	co.Skip = &i
	return co
}
