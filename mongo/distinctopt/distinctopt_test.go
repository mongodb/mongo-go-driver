package distinctopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var c = &mongoopt.Collation{}

func createNestedDistinctBundle1(t *testing.T) *DistinctBundle {
	nestedBundle := BundleDistinct(Collation(c))
	testhelpers.RequireNotNil(t, nestedBundle, "nested bundle was nil")

	outerBundle := BundleDistinct(Collation(c), MaxTime(5), nestedBundle)
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test doubly nested bundle
func createNestedDistinctBundle2(t *testing.T) *DistinctBundle {
	b1 := BundleDistinct(Collation(c))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleDistinct(MaxTime(10), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	outerBundle := BundleDistinct(Collation(c), MaxTime(5), b2)
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test two top level nested bundles
func createNestedDistinctBundle3(t *testing.T) *DistinctBundle {
	b1 := BundleDistinct(Collation(c))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleDistinct(MaxTime(10), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	b3 := BundleDistinct(Collation(c))
	testhelpers.RequireNotNil(t, b3, "nested bundle was nil")

	b4 := BundleDistinct(MaxTime(10), b3)
	testhelpers.RequireNotNil(t, b4, "nested bundle was nil")

	outerBundle := BundleDistinct(b4, MaxTime(5), b2)
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

func TestDistinctOpt(t *testing.T) {
	var bundle1 *DistinctBundle
	bundle1 = bundle1.Collation(c).Collation(c)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []option.Optioner{
		Collation(c).ConvertDistinctOption(),
		Collation(c).ConvertDistinctOption(),
	}
	bundle1DedupOpts := []option.Optioner{
		Collation(c).ConvertDistinctOption(),
	}

	bundle2 := BundleDistinct(Collation(c))
	bundle2Opts := []option.Optioner{
		Collation(c).ConvertDistinctOption(),
	}

	bundle3 := BundleDistinct().
		Collation(c).
		Collation(c)

	bundle3Opts := []option.Optioner{
		OptCollation{c.Convert()}.ConvertDistinctOption(),
		OptCollation{c.Convert()}.ConvertDistinctOption(),
	}

	bundle3DedupOpts := []option.Optioner{
		OptCollation{c.Convert()}.ConvertDistinctOption(),
	}

	nilBundle := BundleDistinct()
	var nilBundleOpts []option.Optioner

	nestedBundle1 := createNestedDistinctBundle1(t)
	nestedBundleOpts1 := []option.Optioner{
		OptCollation{c.Convert()}.ConvertDistinctOption(),
		OptMaxTime(5).ConvertDistinctOption(),
		OptCollation{c.Convert()}.ConvertDistinctOption(),
	}
	nestedBundleDedupOpts1 := []option.Optioner{
		OptMaxTime(5).ConvertDistinctOption(),
		OptCollation{c.Convert()}.ConvertDistinctOption(),
	}

	nestedBundle2 := createNestedDistinctBundle2(t)
	nestedBundleOpts2 := []option.Optioner{
		OptCollation{c.Convert()}.ConvertDistinctOption(),
		OptMaxTime(5).ConvertDistinctOption(),
		OptMaxTime(10).ConvertDistinctOption(),
		OptCollation{c.Convert()}.ConvertDistinctOption(),
	}
	nestedBundleDedupOpts2 := []option.Optioner{
		OptMaxTime(10).ConvertDistinctOption(),
		OptCollation{c.Convert()}.ConvertDistinctOption(),
	}

	nestedBundle3 := createNestedDistinctBundle3(t)
	nestedBundleOpts3 := []option.Optioner{
		OptMaxTime(10).ConvertDistinctOption(),
		OptCollation{c.Convert()}.ConvertDistinctOption(),
		OptMaxTime(5).ConvertDistinctOption(),
		OptMaxTime(10).ConvertDistinctOption(),
		OptCollation{c.Convert()}.ConvertDistinctOption(),
	}
	nestedBundleDedupOpts3 := []option.Optioner{
		OptMaxTime(10).ConvertDistinctOption(),
		OptCollation{c.Convert()}.ConvertDistinctOption(),
	}

	t.Run("TestAll", func(t *testing.T) {
		c := &mongoopt.Collation{
			Locale: "string locale",
		}

		opts := []Distinct{
			Collation(c),
			MaxTime(5000),
		}
		bundle := BundleDistinct(opts...)

		deleteOpts, err := bundle.Unbundle(true)
		testhelpers.RequireNil(t, err, "got non-nill error from unbundle: %s", err)

		if len(deleteOpts) != len(opts) {
			t.Errorf("expected unbundled opts len %d. got %d", len(opts), len(deleteOpts))
		}

		for i, opt := range opts {
			if !reflect.DeepEqual(opt.ConvertDistinctOption(), deleteOpts[i]) {
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
			bundle       *DistinctBundle
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
