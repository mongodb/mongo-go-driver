// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// DistinctOption is for internal use.
type DistinctOption interface {
	DistinctName() string
	DistinctValue() interface{}
}

// collation

// DistinctName is for internal use.
func (opt *OptCollation) DistinctName() string {
	return "collation"
}

// DistinctValue is for internal use.
func (opt *OptCollation) DistinctValue() interface{} {
	return opt.Collation
}

// maxTimeMS

// DistinctName is for internal use.
func (opt *OptMaxTime) DistinctName() string {
	return "maxTimeMS"
}

// DistinctValue is for internal use.
func (opt *OptMaxTime) DistinctValue() interface{} {
	return *opt
}
