package aggregateopt

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAggregateOpt_MakeOptions(t *testing.T) {
	var bundle *AggregateBundle
	bundle = bundle.BypassDocumentValidation(true).Comment("hello world").BypassDocumentValidation(false)

	require.NotNil(t, bundle)
	bundleLen := 0
	for bundle != nil && bundle.option != nil {
		bundleLen++
		bundle = bundle.nextOption
	}

	require.Equal(t, bundleLen, 3)
}

func TestAggregateOpt_MakeBundle(t *testing.T) {
	bundle := BundleAggregate(BypassDocumentValidation(true), Comment("hello world"), AllowDiskUse(true))

	require.NotNil(t, bundle)
	bundleLen := 0
	for bundle != nil && bundle.option != nil {
		bundleLen++
		bundle = bundle.nextOption
	}

	require.Equal(t, bundleLen, 3)
}

func TestAggregateOpt_Dedup(t *testing.T) {
	var cases = []struct {
		name        string
		dedup       bool
		expectedLen int
	}{
		{"TestAggregateOpt_DedupFalse", false, 3},
		{"TestAggregateOpt_DedupTrue", true, 2},
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
