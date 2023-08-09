package topology

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
)

func TestWaitQueueTimeoutError(t *testing.T) {
	t.Run("", func(t *testing.T) {
		err := WaitQueueTimeoutError{Wrapped: context.DeadlineExceeded}
		assert.True(t, errors.Is(err, context.DeadlineExceeded), "")
	})
}
