package description

import (
	"path"
	"testing"

	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
)

const selectorTestsDir = "../../data/server-selection/server_selection"

// Test case for all SDAM spec tests.
func TestServerSelectionSpec(t *testing.T) {
	for _, topology := range [...]string{
		"ReplicaSetNoPrimary",
		"ReplicaSetWithPrimary",
		"Sharded",
		"Single",
		"Unknown",
	} {
		for _, subdir := range [...]string{"read", "write"} {
			subdirPath := path.Join(topology, subdir)

			for _, file := range testhelpers.FindJSONFilesInDir(t,
				path.Join(selectorTestsDir, subdirPath)) {

				runTest(t, selectorTestsDir, subdirPath, file)
			}
		}
	}
}
