// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// CountOption is for internal use.
type CountOption interface {
	CountName() string
	CountValue() interface{}
}

// collation

// CountName is for internal use.
func (opt *OptCollation) CountName() string {
	return "collation"
}

// CountValue is for internal use.
func (opt *OptCollation) CountValue() interface{} {
	return opt.Collation
}

// hint

// CountName is for internal use.
func (opt *OptHint) CountName() string {
	return "hint"
}

// CountValue is for internal use.
func (opt *OptHint) CountValue() interface{} {
	return opt.Hint
}

// limit

// CountName is for internal use.
func (opt *OptLimit) CountName() string {
	return "limit"
}

// CountValue is for internal use.
func (opt *OptLimit) CountValue() interface{} {
	return *opt
}

// maxTimeMS

// CountName is for internal use.
func (opt *OptMaxTime) CountName() string {
	return "maxTimeMS"
}

// CountValue is for internal use.
func (opt *OptMaxTime) CountValue() interface{} {
	return *opt
}

// skip

// CountName is for internal use.
func (opt *OptSkip) CountName() string {
	return "skip"
}

// CountValue is for internal use.
func (opt *OptSkip) CountValue() interface{} {
	return *opt
}
