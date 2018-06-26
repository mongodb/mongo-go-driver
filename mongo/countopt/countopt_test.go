package countopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
)

func createNestedCountBundle1(t *testing.T) *CountBundle {
	nestedBundle := BundleCount(Skip(5))
	testhelpers.RequireNotNil(t, nestedBundle, "nested bundle was nil")

	outerBundle := BundleCount(Skip(10), Limit(500), nestedBundle, MaxTimeMs(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test doubly nested bundle
func createNestedCountBundle2(t *testing.T) *CountBundle {
	b1 := BundleCount(Skip(5))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleCount(Limit(100), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	outerBundle := BundleCount(Skip(10), Limit(500), b2, MaxTimeMs(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test two top level nested bundles
func createNestedCountBundle3(t *testing.T) *CountBundle {
	b1 := BundleCount(Skip(5))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleCount(Limit(100), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	b3 := BundleCount(Skip(10))
	testhelpers.RequireNotNil(t, b3, "nested bundle was nil")

	b4 := BundleCount(Limit(100), b3)
	testhelpers.RequireNotNil(t, b4, "nested bundle was nil")

	outerBundle := BundleCount(b4, Limit(500), b2, MaxTimeMs(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

func TestCountOpt(t *testing.T) {
	var bundle1 *CountBundle
	bundle1 = bundle1.MaxTimeMs(5).Skip(10).MaxTimeMs(15)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []option.Optioner{
		MaxTimeMs(5).ConvertOption(),
		Skip(10).ConvertOption(),
		MaxTimeMs(15).ConvertOption(),
	}
	bundle1DedupOpts := []option.Optioner{
		Skip(10).ConvertOption(),
		MaxTimeMs(15).ConvertOption(),
	}

	bundle2 := BundleCount(MaxTimeMs(1))
	bundle2Opts := []option.Optioner{
		OptMaxTimeMs(1).ConvertOption(),
	}

	rc := &readconcern.ReadConcern{}
	bundle3 := BundleCount().
		MaxTimeMs(1).
		ReadConcern(rc).
		MaxTimeMs(2).
		Limit(6).
		Limit(10)

	bundle3Opts := []option.Optioner{
		OptMaxTimeMs(1).ConvertOption(),
		OptReadConcern{rc}.ConvertOption(),
		OptMaxTimeMs(2).ConvertOption(),
		OptLimit(6).ConvertOption(),
		OptLimit(10).ConvertOption(),
	}

	bundle3DedupOpts := []option.Optioner{
		OptReadConcern{rc}.ConvertOption(),
		OptMaxTimeMs(2).ConvertOption(),
		OptLimit(10).ConvertOption(),
	}

	nilBundle := BundleCount()
	var nilBundleOpts []option.Optioner

	nestedBundle1 := createNestedCountBundle1(t)
	nestedBundleOpts1 := []option.Optioner{
		OptSkip(10).ConvertOption(),
		OptLimit(500).ConvertOption(),
		OptSkip(5).ConvertOption(),
		OptMaxTimeMs(1000).ConvertOption(),
	}
	nestedBundleDedupOpts1 := []option.Optioner{
		OptLimit(500).ConvertOption(),
		OptSkip(5).ConvertOption(),
		OptMaxTimeMs(1000).ConvertOption(),
	}

	nestedBundle2 := createNestedCountBundle2(t)
	nestedBundleOpts2 := []option.Optioner{
		OptSkip(10).ConvertOption(),
		OptLimit(500).ConvertOption(),
		OptLimit(100).ConvertOption(),
		OptSkip(5).ConvertOption(),
		OptMaxTimeMs(1000).ConvertOption(),
	}
	nestedBundleDedupOpts2 := []option.Optioner{
		OptLimit(100).ConvertOption(),
		OptSkip(5).ConvertOption(),
		OptMaxTimeMs(1000).ConvertOption(),
	}

	nestedBundle3 := createNestedCountBundle3(t)
	nestedBundleOpts3 := []option.Optioner{
		OptLimit(100).ConvertOption(),
		OptSkip(10).ConvertOption(),
		OptLimit(500).ConvertOption(),
		OptLimit(100).ConvertOption(),
		OptSkip(5).ConvertOption(),
		OptMaxTimeMs(1000).ConvertOption(),
	}
	nestedBundleDedupOpts3 := []option.Optioner{
		OptLimit(100).ConvertOption(),
		OptSkip(5).ConvertOption(),
		OptMaxTimeMs(1000).ConvertOption(),
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
			bundle       *CountBundle
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
