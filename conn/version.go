package conn

import "strconv"

// Version represents a software version.
type Version struct {
	Desc  string
	Parts []uint8
}

// AtLeast ensures that the version is
// at least as large as the "other" version.
func (v *Version) AtLeast(other ...uint8) bool {
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
func (v *Version) String() string {
	if v.Desc == "" {
		var s string
		for i, p := range v.Parts {
			if i != 0 {
				s += "."
			}
			s += strconv.Itoa(int(p))
		}
		return s
	}

	return v.Desc
}
