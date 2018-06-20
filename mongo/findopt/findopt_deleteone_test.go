package findopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
)

func createNestedDeleteOneBundle1(t *testing.T) *DeleteOneBundle {
	nestedBundle := BundleDeleteOne(Projection(false))
	testhelpers.RequireNotNil(t, nestedBundle, "nested bundle was nil")

	outerBundle := BundleDeleteOne(Projection(true), MaxTime(500), nestedBundle, MaxTime(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test doubly nested bundle
func createNestedDeleteOneBundle2(t *testing.T) *DeleteOneBundle {
	b1 := BundleDeleteOne(Projection(false))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleDeleteOne(MaxTime(100), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	outerBundle := BundleDeleteOne(Projection(true), MaxTime(500), b2, MaxTime(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test two top level nested bundles
func createNestedDeleteOneBundle3(t *testing.T) *DeleteOneBundle {
	b1 := BundleDeleteOne(Projection(false))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleDeleteOne(MaxTime(100), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	b3 := BundleDeleteOne(Projection(true))
	testhelpers.RequireNotNil(t, b3, "nested bundle was nil")

	b4 := BundleDeleteOne(MaxTime(100), b3)
	testhelpers.RequireNotNil(t, b4, "nested bundle was nil")

	outerBundle := BundleDeleteOne(b4, MaxTime(500), b2, MaxTime(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

func TestFindAndDeleteOneOpt(t *testing.T) {
	var bundle1 *DeleteOneBundle
	bundle1 = bundle1.Projection(true).Sort(false)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []option.Optioner{
		OptProjection{true}.ConvertDeleteOneOption(),
		OptSort{false}.ConvertDeleteOneOption(),
	}
	bundle1DedupOpts := []option.Optioner{
		OptProjection{true}.ConvertDeleteOneOption(),
		OptSort{false}.ConvertDeleteOneOption(),
	}

	bundle2 := BundleDeleteOne(MaxTime(1))
	bundle2Opts := []option.Optioner{
		OptMaxTime(1).ConvertDeleteOneOption(),
	}

	bundle3 := BundleDeleteOne().
		MaxTime(1).
		MaxTime(2).
		Projection(false).
		Projection(true)

	bundle3Opts := []option.Optioner{
		OptMaxTime(1).ConvertDeleteOneOption(),
		OptMaxTime(2).ConvertDeleteOneOption(),
		OptProjection{false}.ConvertDeleteOneOption(),
		OptProjection{true}.ConvertDeleteOneOption(),
	}

	bundle3DedupOpts := []option.Optioner{
		OptMaxTime(2).ConvertDeleteOneOption(),
		OptProjection{true}.ConvertDeleteOneOption(),
	}

	nilBundle := BundleDeleteOne()
	var nilBundleOpts []option.Optioner

	nestedBundle1 := createNestedDeleteOneBundle1(t)
	nestedBundleOpts1 := []option.Optioner{
		OptProjection{true}.ConvertDeleteOneOption(),
		OptMaxTime(500).ConvertDeleteOneOption(),
		OptProjection{false}.ConvertDeleteOneOption(),
		OptMaxTime(1000).ConvertDeleteOneOption(),
	}
	nestedBundleDedupOpts1 := []option.Optioner{
		OptProjection{false}.ConvertDeleteOneOption(),
		OptMaxTime(1000).ConvertDeleteOneOption(),
	}

	nestedBundle2 := createNestedDeleteOneBundle2(t)
	nestedBundleOpts2 := []option.Optioner{
		OptProjection{true}.ConvertDeleteOneOption(),
		OptMaxTime(500).ConvertDeleteOneOption(),
		OptMaxTime(100).ConvertDeleteOneOption(),
		OptProjection{false}.ConvertDeleteOneOption(),
		OptMaxTime(1000).ConvertDeleteOneOption(),
	}
	nestedBundleDedupOpts2 := []option.Optioner{
		OptProjection{false}.ConvertDeleteOneOption(),
		OptMaxTime(1000).ConvertDeleteOneOption(),
	}

	nestedBundle3 := createNestedDeleteOneBundle3(t)
	nestedBundleOpts3 := []option.Optioner{
		OptMaxTime(100).ConvertDeleteOneOption(),
		OptProjection{true}.ConvertDeleteOneOption(),
		OptMaxTime(500).ConvertDeleteOneOption(),
		OptMaxTime(100).ConvertDeleteOneOption(),
		OptProjection{false}.ConvertDeleteOneOption(),
		OptMaxTime(1000).ConvertDeleteOneOption(),
	}
	nestedBundleDedupOpts3 := []option.Optioner{
		OptProjection{false}.ConvertDeleteOneOption(),
		OptMaxTime(1000).ConvertDeleteOneOption(),
	}

	t.Run("TestAll", func(t *testing.T) {
		proj := Projection(true)
		sort := Sort(true)

		opts := []DeleteOne{
			MaxTime(5),
			Projection(proj),
			Sort(sort),
		}
		bundle := BundleDeleteOne(opts...)

		deleteOpts, err := bundle.Unbundle(true)
		testhelpers.RequireNil(t, err, "got non-nill error from unbundle: %s", err)

		if len(deleteOpts) != len(opts) {
			t.Errorf("expected unbundled opts len %d. got %d", len(opts), len(deleteOpts))
		}

		for i, opt := range opts {
			if !reflect.DeepEqual(opt.ConvertDeleteOneOption(), deleteOpts[i]) {
				t.Errorf("opt mismatch. expected %#v, got %#v", opt, deleteOpts[i])
			}
		}
	})

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
			bundle       *DeleteOneBundle
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
