// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// DistinctOptions represents options that can be used to configure a Distinct operation.
type DistinctOptions struct {
	// Specifies a collation to use for string comparisons during the operation. This option is only valid for MongoDB
	// versions >= 3.4. For previous server versions, the driver will return an error if this option is used. The
	// default value is nil, which means the default collation of the collection will be used.
	Collation *Collation

	// A string or document that will be included in server logs, profiling logs, and currentOp queries to help trace
	// the operation. The default value is nil, which means that no comment will be included in the logs.
	Comment interface{}
}

// Distinct creates a new DistinctOptions instance.
func Distinct() *DistinctOptions {
	return &DistinctOptions{}
}

// SetCollation sets the value for the Collation field.
func (do *DistinctOptions) SetCollation(c *Collation) *DistinctOptions {
	do.Collation = c
	return do
}

// SetComment sets the value for the Comment field.
func (do *DistinctOptions) SetComment(comment interface{}) *DistinctOptions {
	do.Comment = comment
	return do
}
