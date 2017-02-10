package desc

import "strconv"

// NewVersion creates a new version.
func NewVersion(parts ...uint8) Version {
	desc := ""
	for i, p := range parts {
		if i != 0 {
			desc += "."
		}
		desc += strconv.Itoa(int(p))
	}

	return Version{
		desc:  desc,
		parts: parts,
	}
}

// NewVersionWithDesc creates a new version given a description.
func NewVersionWithDesc(desc string, parts ...uint8) Version {
	return Version{
		desc:  desc,
		parts: parts,
	}
}

// Version represents a software version.
type Version struct {
	desc  string
	parts []uint8
}

// AtLeast ensures that the version is
// at least as large as the "other" version.
func (v *Version) AtLeast(other ...uint8) bool {
	for i := range other {
		if i == len(v.parts) {
			return false
		}
		if v.parts[i] < other[i] {
			return false
		}
	}
	return true
}

// String provides the string representation of the Version.
func (v *Version) String() string {
	return v.desc
}
