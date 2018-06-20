package findopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

func createNestedReplaceOneBundle1(t *testing.T) *ReplaceOneBundle {
	nestedBundle := BundleReplaceOne(Upsert(false))
	testhelpers.RequireNotNil(t, nestedBundle, "nested bundle was nil")

	outerBundle := BundleReplaceOne(Upsert(true), MaxTime(500), nestedBundle, MaxTime(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test doubly nested bundle
func createNestedReplaceOneBundle2(t *testing.T) *ReplaceOneBundle {
	b1 := BundleReplaceOne(Upsert(false))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleReplaceOne(MaxTime(100), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	outerBundle := BundleReplaceOne(Upsert(true), MaxTime(500), b2, MaxTime(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test two top level nested bundles
func createNestedReplaceOneBundle3(t *testing.T) *ReplaceOneBundle {
	b1 := BundleReplaceOne(Upsert(false))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleReplaceOne(MaxTime(100), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	b3 := BundleReplaceOne(Upsert(true))
	testhelpers.RequireNotNil(t, b3, "nested bundle was nil")

	b4 := BundleReplaceOne(MaxTime(100), b3)
	testhelpers.RequireNotNil(t, b4, "nested bundle was nil")

	outerBundle := BundleReplaceOne(b4, MaxTime(500), b2, MaxTime(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

func TestFindAndReplaceOneOpt(t *testing.T) {
	var bundle1 *ReplaceOneBundle
	bundle1 = bundle1.Upsert(true).BypassDocumentValidation(false)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []option.Optioner{
		OptUpsert(true).ConvertReplaceOneOption(),
		OptBypassDocumentValidation(false).ConvertReplaceOneOption(),
	}
	bundle1DedupOpts := []option.Optioner{
		OptUpsert(true).ConvertReplaceOneOption(),
		OptBypassDocumentValidation(false).ConvertReplaceOneOption(),
	}

	bundle2 := BundleReplaceOne(MaxTime(1))
	bundle2Opts := []option.Optioner{
		OptMaxTime(1).ConvertReplaceOneOption(),
	}

	bundle3 := BundleReplaceOne().
		MaxTime(1).
		MaxTime(2).
		Upsert(false).
		Upsert(true)

	bundle3Opts := []option.Optioner{
		OptMaxTime(1).ConvertReplaceOneOption(),
		OptMaxTime(2).ConvertReplaceOneOption(),
		OptUpsert(false).ConvertReplaceOneOption(),
		OptUpsert(true).ConvertReplaceOneOption(),
	}

	bundle3DedupOpts := []option.Optioner{
		OptMaxTime(2).ConvertReplaceOneOption(),
		OptUpsert(true).ConvertReplaceOneOption(),
	}

	nilBundle := BundleReplaceOne()
	var nilBundleOpts []option.Optioner

	nestedBundle1 := createNestedReplaceOneBundle1(t)
	nestedBundleOpts1 := []option.Optioner{
		OptUpsert(true).ConvertReplaceOneOption(),
		OptMaxTime(500).ConvertReplaceOneOption(),
		OptUpsert(false).ConvertReplaceOneOption(),
		OptMaxTime(1000).ConvertReplaceOneOption(),
	}
	nestedBundleDedupOpts1 := []option.Optioner{
		OptUpsert(false).ConvertReplaceOneOption(),
		OptMaxTime(1000).ConvertReplaceOneOption(),
	}

	nestedBundle2 := createNestedReplaceOneBundle2(t)
	nestedBundleOpts2 := []option.Optioner{
		OptUpsert(true).ConvertReplaceOneOption(),
		OptMaxTime(500).ConvertReplaceOneOption(),
		OptMaxTime(100).ConvertReplaceOneOption(),
		OptUpsert(false).ConvertReplaceOneOption(),
		OptMaxTime(1000).ConvertReplaceOneOption(),
	}
	nestedBundleDedupOpts2 := []option.Optioner{
		OptUpsert(false).ConvertReplaceOneOption(),
		OptMaxTime(1000).ConvertReplaceOneOption(),
	}

	nestedBundle3 := createNestedReplaceOneBundle3(t)
	nestedBundleOpts3 := []option.Optioner{
		OptMaxTime(100).ConvertReplaceOneOption(),
		OptUpsert(true).ConvertReplaceOneOption(),
		OptMaxTime(500).ConvertReplaceOneOption(),
		OptMaxTime(100).ConvertReplaceOneOption(),
		OptUpsert(false).ConvertReplaceOneOption(),
		OptMaxTime(1000).ConvertReplaceOneOption(),
	}
	nestedBundleDedupOpts3 := []option.Optioner{
		OptUpsert(false).ConvertReplaceOneOption(),
		OptMaxTime(1000).ConvertReplaceOneOption(),
	}

	t.Run("TestAll", func(t *testing.T) {
		c := &mongoopt.Collation{
			Locale: "string locale",
		}
		proj := Projection(true)
		sort := Sort(true)

		opts := []ReplaceOne{
			Collation(c),
			MaxTime(5),
			Projection(proj),
			ReturnDocument(mongoopt.After),
			Sort(sort),
			Upsert(true),
		}
		bundle := BundleReplaceOne(opts...)

		deleteOpts, err := bundle.Unbundle(true)
		testhelpers.RequireNil(t, err, "got non-nill error from unbundle: %s", err)

		if len(deleteOpts) != len(opts) {
			t.Errorf("expected unbundled opts len %d. got %d", len(opts), len(deleteOpts))
		}

		for i, opt := range opts {
			if !reflect.DeepEqual(opt.ConvertReplaceOneOption(), deleteOpts[i]) {
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
			bundle       *ReplaceOneBundle
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
