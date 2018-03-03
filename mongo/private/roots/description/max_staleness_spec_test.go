package description

import (
	"path"
	"testing"

	"github.com/mongodb/mongo-go-driver/mongo/internal/testutil/helpers"
)

const maxStalenessTestsDir = "../../../../data/max-staleness"

// Test case for all max staleness spec tests.
func TestMaxStalenessSpec(t *testing.T) {
	for _, topology := range [...]string{
		"ReplicaSetNoPrimary",
		"ReplicaSetWithPrimary",
		"Sharded",
		"Single",
		"Unknown",
	} {
		for _, file := range testhelpers.FindJSONFilesInDir(t,
			path.Join(maxStalenessTestsDir, topology)) {

			runTest(t, maxStalenessTestsDir, topology, file)
		}
	}
}
