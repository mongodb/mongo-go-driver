package description

import "fmt"

// VersionRange represents a range of versions.
type VersionRange struct {
	Min int32
	Max int32
}

// NewVersionRange creates a new VersionRange given a min and a max.
func NewVersionRange(min, max int32) VersionRange {
	return VersionRange{Min: min, Max: max}
}

// Includes returns a bool indicating whether the supplied integer is included
// in the range.
func (vr VersionRange) Includes(v int32) bool {
	return v >= vr.Min && v <= vr.Max
}

// String implements the fmt.Stringer interface.
func (vr VersionRange) String() string {
	return fmt.Sprintf("[%d, %d]", vr.Min, vr.Max)
}
