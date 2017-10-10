package model

import "fmt"

// Range is an inclusive range between 2 uint8.
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
