package findopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

func createNestedUpdateOneBundle1(t *testing.T) *UpdateOneBundle {
	nestedBundle := BundleUpdateOne(Upsert(false))
	testhelpers.RequireNotNil(t, nestedBundle, "nested bundle was nil")

	outerBundle := BundleUpdateOne(Upsert(true), MaxTime(500), nestedBundle, MaxTime(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test doubly nested bundle
func createNestedUpdateOneBundle2(t *testing.T) *UpdateOneBundle {
	b1 := BundleUpdateOne(Upsert(false))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleUpdateOne(MaxTime(100), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	outerBundle := BundleUpdateOne(Upsert(true), MaxTime(500), b2, MaxTime(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test two top level nested bundles
func createNestedUpdateOneBundle3(t *testing.T) *UpdateOneBundle {
	b1 := BundleUpdateOne(Upsert(false))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleUpdateOne(MaxTime(100), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	b3 := BundleUpdateOne(Upsert(true))
	testhelpers.RequireNotNil(t, b3, "nested bundle was nil")

	b4 := BundleUpdateOne(MaxTime(100), b3)
	testhelpers.RequireNotNil(t, b4, "nested bundle was nil")

	outerBundle := BundleUpdateOne(b4, MaxTime(500), b2, MaxTime(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

func TestFindAndUpdateOneOpt(t *testing.T) {
	var bundle1 *UpdateOneBundle
	bundle1 = bundle1.Upsert(true).BypassDocumentValidation(false)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []option.Optioner{
		OptUpsert(true).ConvertUpdateOneOption(),
		OptBypassDocumentValidation(false).ConvertUpdateOneOption(),
	}
	bundle1DedupOpts := []option.Optioner{
		OptUpsert(true).ConvertUpdateOneOption(),
		OptBypassDocumentValidation(false).ConvertUpdateOneOption(),
	}

	bundle2 := BundleUpdateOne(MaxTime(1))
	bundle2Opts := []option.Optioner{
		OptMaxTime(1).ConvertUpdateOneOption(),
	}

	bundle3 := BundleUpdateOne().
		MaxTime(1).
		MaxTime(2).
		Upsert(false).
		Upsert(true)

	bundle3Opts := []option.Optioner{
		OptMaxTime(1).ConvertUpdateOneOption(),
		OptMaxTime(2).ConvertUpdateOneOption(),
		OptUpsert(false).ConvertUpdateOneOption(),
		OptUpsert(true).ConvertUpdateOneOption(),
	}

	bundle3DedupOpts := []option.Optioner{
		OptMaxTime(2).ConvertUpdateOneOption(),
		OptUpsert(true).ConvertUpdateOneOption(),
	}

	nilBundle := BundleUpdateOne()
	var nilBundleOpts []option.Optioner

	nestedBundle1 := createNestedUpdateOneBundle1(t)
	nestedBundleOpts1 := []option.Optioner{
		OptUpsert(true).ConvertUpdateOneOption(),
		OptMaxTime(500).ConvertUpdateOneOption(),
		OptUpsert(false).ConvertUpdateOneOption(),
		OptMaxTime(1000).ConvertUpdateOneOption(),
	}
	nestedBundleDedupOpts1 := []option.Optioner{
		OptUpsert(false).ConvertUpdateOneOption(),
		OptMaxTime(1000).ConvertUpdateOneOption(),
	}

	nestedBundle2 := createNestedUpdateOneBundle2(t)
	nestedBundleOpts2 := []option.Optioner{
		OptUpsert(true).ConvertUpdateOneOption(),
		OptMaxTime(500).ConvertUpdateOneOption(),
		OptMaxTime(100).ConvertUpdateOneOption(),
		OptUpsert(false).ConvertUpdateOneOption(),
		OptMaxTime(1000).ConvertUpdateOneOption(),
	}
	nestedBundleDedupOpts2 := []option.Optioner{
		OptUpsert(false).ConvertUpdateOneOption(),
		OptMaxTime(1000).ConvertUpdateOneOption(),
	}

	nestedBundle3 := createNestedUpdateOneBundle3(t)
	nestedBundleOpts3 := []option.Optioner{
		OptMaxTime(100).ConvertUpdateOneOption(),
		OptUpsert(true).ConvertUpdateOneOption(),
		OptMaxTime(500).ConvertUpdateOneOption(),
		OptMaxTime(100).ConvertUpdateOneOption(),
		OptUpsert(false).ConvertUpdateOneOption(),
		OptMaxTime(1000).ConvertUpdateOneOption(),
	}
	nestedBundleDedupOpts3 := []option.Optioner{
		OptUpsert(false).ConvertUpdateOneOption(),
		OptMaxTime(1000).ConvertUpdateOneOption(),
	}

	t.Run("TestAll", func(t *testing.T) {
		c := &mongoopt.Collation{
			Locale: "string locale",
		}
		proj := Projection(true)
		sort := Sort(true)

		opts := []UpdateOne{
			BypassDocumentValidation(false),
			Collation(c),
			MaxTime(5),
			Projection(proj),
			ReturnDocument(mongoopt.After),
			Sort(sort),
			Upsert(true),
		}
		bundle := BundleUpdateOne(opts...)

		deleteOpts, err := bundle.Unbundle(true)
		testhelpers.RequireNil(t, err, "got non-nill error from unbundle: %s", err)

		if len(deleteOpts) != len(opts) {
			t.Errorf("expected unbundled opts len %d. got %d", len(opts), len(deleteOpts))
		}

		for i, opt := range opts {
			if !reflect.DeepEqual(opt.ConvertUpdateOneOption(), deleteOpts[i]) {
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
			bundle       *UpdateOneBundle
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
