// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// AggregateOption is for internal use.
type AggregateOption interface {
	AggregateName() string
	AggregateValue() interface{}
}

// allowDiskUse

// AggregateName is for internal use.
func (opt *OptAllowDiskUse) AggregateName() string {
	return "allowDiskUse"
}

// AggregateValue is for internal use.
func (opt *OptAllowDiskUse) AggregateValue() interface{} {
	return *opt
}

// batchSize

// AggregateName is for internal use.
func (opt *OptBatchSize) AggregateName() string {
	return "batchSize"
}

// AggregateValue is for internal use.
func (opt *OptBatchSize) AggregateValue() interface{} {
	return *opt
}

// bypassDocumentValidation

// AggregateName is for internal use.
func (opt *OptBypassDocumentValidation) AggregateName() string {
	return "bypassDocumentValidation"
}

// AggregateValue is for internal use.
func (opt *OptBypassDocumentValidation) AggregateValue() interface{} {
	return *opt
}

// collation

// AggregateName is for internal use.
func (opt *OptCollation) AggregateName() string {
	return "collation"
}

// AggregateValue is for internal use.
func (opt *OptCollation) AggregateValue() interface{} {
	return opt.Collation
}

// comment

// AggregateName is for internal use.
func (opt *OptComment) AggregateName() string {
	return "comment"
}

// AggregateValue is for internal use.
func (opt *OptComment) AggregateValue() interface{} {
	return *opt
}

// maxTimeMS

// AggregateName is for internal use.
func (opt *OptMaxTime) AggregateName() string {
	return "maxTimeMS"
}

// AggregateValue is for internal use.
func (opt *OptMaxTime) AggregateValue() interface{} {
	return *opt
}
