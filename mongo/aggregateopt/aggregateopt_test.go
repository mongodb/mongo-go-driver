package aggregateopt

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAggregateOpt(t *testing.T) {
	t.Run("MakeOptions", func(t *testing.T) {
		var bundle *AggregateBundle
		bundle = bundle.BypassDocumentValidation(true).Comment("hello world").BypassDocumentValidation(false)

		require.NotNil(t, bundle)
		bundleLen := 0
		for bundle != nil && bundle.option != nil {
			bundleLen++
			bundle = bundle.next
		}

		require.Equal(t, bundleLen, 3)
	})

	var cases = []struct {
		name        string
		dedup       bool
		expectedLen int
	}{
		{"DedupFalse", false, 3},
		{"DedupTrue", true, 2},
	}

	var bundle *AggregateBundle
	bundle = bundle.BypassDocumentValidation(true).Comment("hello world").BypassDocumentValidation(false)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			options, err := bundle.Unbundle(tc.dedup)
			require.NoError(t, err)
			require.Equal(t, len(options), tc.expectedLen)
		})
	}
}
