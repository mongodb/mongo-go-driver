package feature

import (
	"fmt"

	"github.com/10gen/mongo-go-driver/desc"
)

// MaxStaleness returns an error if the given server
// does not support max staleness.
func MaxStaleness(version desc.Version) error {
	if !version.AtLeast(3, 4, 0) {
		return fmt.Errorf("max staleness is only supported for servers 3.4 or newer")
	}

	return nil
}
