package feature_test

import (
	"testing"

	"github.com/10gen/mongo-go-driver/conn"
	. "github.com/10gen/mongo-go-driver/internal/feature"
	"github.com/stretchr/testify/require"
)

func TestMaxStaleness(t *testing.T) {
	tests := []struct {
		version  conn.Version
		expected bool
	}{
		{conn.Version{Parts: []uint8{2, 4, 0}}, false},
		{conn.Version{Parts: []uint8{3, 3, 99}}, false},
		{conn.Version{Parts: []uint8{3, 4, 0}}, true},
		{conn.Version{Parts: []uint8{3, 4, 1}}, true},
	}

	for _, test := range tests {
		t.Run(test.version.String(), func(t *testing.T) {
			t.Parallel()

			err := MaxStaleness(test.version)
			if test.expected {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
