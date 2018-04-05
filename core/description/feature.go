package description

import (
	"fmt"
)

// MaxStalenessSupported returns an error if the given server version
// does not support max staleness.
func MaxStalenessSupported(serverVersion Version, wireVersion *VersionRange) error {
	if !serverVersion.AtLeast(3, 4, 0) || (wireVersion != nil && wireVersion.Max < 5) {
		return fmt.Errorf("max staleness is only supported for servers 3.4 or newer")
	}

	return nil
}

// ScramSHA1Supported returns an error if the given server version
// does not support scram-sha-1.
func ScramSHA1Supported(version Version) error {
	if !version.AtLeast(3, 0, 0) {
		return fmt.Errorf("SCRAM-SHA-1 is only supported for servers 3.0 or newer")
	}

	return nil
}
