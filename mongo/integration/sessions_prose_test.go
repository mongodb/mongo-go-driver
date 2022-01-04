package integration

import (
	"errors"
	"testing"

	"github.com/mailgun/mongo-go-driver/internal/testutil/assert"
	"github.com/mailgun/mongo-go-driver/mongo/integration/mtest"
	"github.com/mailgun/mongo-go-driver/mongo/options"
)

func TestSessionsProse(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().CreateClient(false))
	mt.Run("causalConsistency and Snapshot true", func(mt *mtest.T) {
		// causalConsistency and snapshot are mutually exclusive
		sessOpts := options.Session().SetCausalConsistency(true).SetSnapshot(true)
		_, err := mt.Client.StartSession(sessOpts)
		assert.NotNil(mt, err, "expected StartSession error, got nil")
		expectedErr := errors.New("causal consistency and snapshot cannot both be set for a session")
		assert.Equal(mt, expectedErr, err, "expected error %v, got %v", expectedErr, err)
	})
}
