// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package model

import "fmt"

// Range is an inclusive range between 2 uint32.
type Range struct {
	Min int32
	Max int32
}

// NewRange creates a new Range given a min and a max.
func NewRange(min int32, max int32) Range {
	return Range{Min: min, Max: max}
}

// Includes returns a bool indicating whether the supplied
// integer is included in the range.
func (r *Range) Includes(i int32) bool {
	return i >= r.Min && i <= r.Max
}

func (r *Range) String() string {
	return fmt.Sprintf("[%d, %d]", r.Min, r.Max)
}
