package insertopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
)

var wc1 = writeconcern.New(writeconcern.W(10))
var wc2 = writeconcern.New(writeconcern.W(20))

func createNestedOneBundle1(t *testing.T) *OneBundle {
	nestedBundle := BundleOne(BypassDocumentValidation(false))
	testhelpers.RequireNotNil(t, nestedBundle, "nested bundle was nil")

	outerBundle := BundleOne(BypassDocumentValidation(true), WriteConcern(wc1), nestedBundle)
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test doubly nested bundle
func createNestedOneBundle2(t *testing.T) *OneBundle {
	b1 := BundleOne(BypassDocumentValidation(false))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleOne(WriteConcern(wc2), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	outerBundle := BundleOne(BypassDocumentValidation(true), WriteConcern(wc1), b2)
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test two top level nested bundles
func createNestedOneBundle3(t *testing.T) *OneBundle {
	b1 := BundleOne(BypassDocumentValidation(false))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleOne(WriteConcern(wc2), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	b3 := BundleOne(BypassDocumentValidation(true))
	testhelpers.RequireNotNil(t, b3, "nested bundle was nil")

	b4 := BundleOne(WriteConcern(wc2), b3)
	testhelpers.RequireNotNil(t, b4, "nested bundle was nil")

	outerBundle := BundleOne(b4, WriteConcern(wc1), b2)
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

func TestFindOneOpt(t *testing.T) {
	var bundle1 *OneBundle
	bundle1 = bundle1.BypassDocumentValidation(true).BypassDocumentValidation(false)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []option.Optioner{
		BypassDocumentValidation(true).ConvertOneOption(),
		BypassDocumentValidation(false).ConvertOneOption(),
	}
	bundle1DedupOpts := []option.Optioner{
		BypassDocumentValidation(false).ConvertOneOption(),
	}

	bundle2 := BundleOne(BypassDocumentValidation(true))
	bundle2Opts := []option.Optioner{
		BypassDocumentValidation(true).ConvertOneOption(),
	}

	bundle3 := BundleOne().
		BypassDocumentValidation(false).
		BypassDocumentValidation(true)

	bundle3Opts := []option.Optioner{
		OptBypassDocumentValidation(false).ConvertOneOption(),
		OptBypassDocumentValidation(true).ConvertOneOption(),
	}

	bundle3DedupOpts := []option.Optioner{
		OptBypassDocumentValidation(true).ConvertOneOption(),
	}

	nilBundle := BundleOne()
	var nilBundleOpts []option.Optioner

	nestedBundle1 := createNestedOneBundle1(t)
	nestedBundleOpts1 := []option.Optioner{
		OptBypassDocumentValidation(true).ConvertOneOption(),
		OptWriteConcern{wc1}.ConvertOneOption(),
		OptBypassDocumentValidation(false).ConvertOneOption(),
	}
	nestedBundleDedupOpts1 := []option.Optioner{
		OptWriteConcern{wc1}.ConvertOneOption(),
		OptBypassDocumentValidation(false).ConvertOneOption(),
	}

	nestedBundle2 := createNestedOneBundle2(t)
	nestedBundleOpts2 := []option.Optioner{
		OptBypassDocumentValidation(true).ConvertOneOption(),
		OptWriteConcern{wc1}.ConvertOneOption(),
		OptWriteConcern{wc2}.ConvertOneOption(),
		OptBypassDocumentValidation(false).ConvertOneOption(),
	}
	nestedBundleDedupOpts2 := []option.Optioner{
		OptWriteConcern{wc2}.ConvertOneOption(),
		OptBypassDocumentValidation(false).ConvertOneOption(),
	}

	nestedBundle3 := createNestedOneBundle3(t)
	nestedBundleOpts3 := []option.Optioner{
		OptWriteConcern{wc2}.ConvertOneOption(),
		OptBypassDocumentValidation(true).ConvertOneOption(),
		OptWriteConcern{wc1}.ConvertOneOption(),
		OptWriteConcern{wc2}.ConvertOneOption(),
		OptBypassDocumentValidation(false).ConvertOneOption(),
	}
	nestedBundleDedupOpts3 := []option.Optioner{
		OptWriteConcern{wc2}.ConvertOneOption(),
		OptBypassDocumentValidation(false).ConvertOneOption(),
	}

	t.Run("MakeOptions", func(t *testing.T) {
		head := bundle1

		bundleLen := 0
		for head != nil && head.option != nil {
			bundleLen++
			head = head.next
		}

		if bundleLen != len(bundle1Opts) {
			t.Errorf("expected bundle length %d. got: %d", len(bundle1Opts), bundleLen)
		}
	})

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name         string
			dedup        bool
			bundle       *OneBundle
			expectedOpts []option.Optioner
		}{
			{"NilBundle", false, nilBundle, nilBundleOpts},
			{"Bundle1", false, bundle1, bundle1Opts},
			{"Bundle1Dedup", true, bundle1, bundle1DedupOpts},
			{"Bundle2", false, bundle2, bundle2Opts},
			{"Bundle2Dedup", true, bundle2, bundle2Opts},
			{"Bundle3", false, bundle3, bundle3Opts},
			{"Bundle3Dedup", true, bundle3, bundle3DedupOpts},
			{"NestedBundle1_DedupFalse", false, nestedBundle1, nestedBundleOpts1},
			{"NestedBundle1_DedupTrue", true, nestedBundle1, nestedBundleDedupOpts1},
			{"NestedBundle2_DedupFalse", false, nestedBundle2, nestedBundleOpts2},
			{"NestedBundle2_DedupTrue", true, nestedBundle2, nestedBundleDedupOpts2},
			{"NestedBundle3_DedupFalse", false, nestedBundle3, nestedBundleOpts3},
			{"NestedBundle3_DedupTrue", true, nestedBundle3, nestedBundleDedupOpts3},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				options, err := tc.bundle.Unbundle(tc.dedup)
				testhelpers.RequireNil(t, err, "got non-nill error from unbundle: %s", err)

				if len(options) != len(tc.expectedOpts) {
					t.Errorf("options length does not match expected length. got %d expected %d", len(options),
						len(tc.expectedOpts))
				} else {
					for i, opt := range options {
						if !reflect.DeepEqual(opt, tc.expectedOpts[i]) {
							t.Errorf("expected: %s\nreceived: %s", opt, tc.expectedOpts[i])
						}
					}
				}
			})
		}
	})
}
