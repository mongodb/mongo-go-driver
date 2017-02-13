package feature_test

import (
	"testing"

	"github.com/10gen/mongo-go-driver/desc"
	. "github.com/10gen/mongo-go-driver/feature"
	"github.com/stretchr/testify/require"
)

func TestMaxStaleness(t *testing.T) {

	tests := []struct {
		version  desc.Version
		expected bool
	}{
		{desc.NewVersion(2, 4, 0), false},
		{desc.NewVersion(3, 3, 99), false},
		{desc.NewVersion(3, 4, 0), true},
		{desc.NewVersion(3, 4, 1), true},
	}
	for _, test := range tests {
		t.Run(test.version.String(), func(t *testing.T) {
			err := MaxStaleness(test.version)
			if test.expected {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
