package util

// Version represents a software version.
type Version struct {
	Desc  string
	Parts []uint8
}

// AtLeast ensures that the version is
// at least as large as the "other" version.
func (v Version) AtLeast(other ...uint8) bool {
	for i := range other {
		if i == len(v.Parts) {
			return false
		}
		if v.Parts[i] < other[i] {
			return false
		}
	}
	return true
}

// String provides the string representation of the Version.
func (v Version) String() string {
	return v.Desc
}

// Range is a range of uint8.
type Range struct {
	Min uint8
	Max uint8
}
