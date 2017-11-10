// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// InsertOption is for internal use.
type InsertOption interface {
	InsertName() string
	InsertValue() interface{}
}

// bypassDocumentValidation

// InsertName is for internal use.
func (opt *OptBypassDocumentValidation) InsertName() string {
	return "bypassDocumentValidation"
}

// InsertValue is for internal use.
func (opt *OptBypassDocumentValidation) InsertValue() interface{} {
	return *opt
}

// ordered

// InsertName is for internal use.
func (opt *OptOrdered) InsertName() string {
	return "ordered"
}

// InsertValue is for internal use.
func (opt *OptOrdered) InsertValue() interface{} {
	return *opt
}
