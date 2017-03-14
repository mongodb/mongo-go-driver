package model

import "fmt"

// Range is an inclusive range between 2 uint8.
type Range struct {
	Min uint8
	Max uint8
}

// Includes returns a bool indicating whether the supplied
// integer is included in the range.
func (r *Range) Includes(i uint8) bool {
	return i >= r.Min && i <= r.Max
}

func (r *Range) String() string {
	return fmt.Sprintf("[%d, %d]", r.Min, r.Max)
}
