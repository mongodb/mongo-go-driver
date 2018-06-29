package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServerSession(t *testing.T) {

	t.Run("Expired", func(t *testing.T) {
		sess, err := newServerSession()
		require.Nil(t, err, "Unexpected error")
		if !sess.expired(0) {
			t.Errorf("session should be expired")
		}
		sess.LastUsed = time.Now().Add(-30 * time.Minute)
		if !sess.expired(30) {
			t.Errorf("session should be expired")
		}

	})
}
