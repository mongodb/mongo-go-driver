package findopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

func createNestedOneBundle1(t *testing.T) *OneBundle {
	nestedBundle := BundleOne(AllowPartialResults(false), Comment("hello world nested"))
	testhelpers.RequireNotNil(t, nestedBundle, "nested bundle was nil")

	outerBundle := BundleOne(AllowPartialResults(true), MaxTime(500), nestedBundle, BatchSize(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test doubly nested bundle
func createNestedOneBundle2(t *testing.T) *OneBundle {
	b1 := BundleOne(AllowPartialResults(false), Comment("nest1"))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleOne(MaxTime(100), b1, Comment("nest2"))
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	outerBundle := BundleOne(AllowPartialResults(true), MaxTime(500), b2, BatchSize(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test two top level nested bundles
func createNestedOneBundle3(t *testing.T) *OneBundle {
	b1 := BundleOne(AllowPartialResults(false), Comment("nest1"))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleOne(MaxTime(100), b1, Comment("nest2"))
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	b3 := BundleOne(AllowPartialResults(true), Comment("nest3"))
	testhelpers.RequireNotNil(t, b3, "nested bundle was nil")

	b4 := BundleOne(MaxTime(100), b3, Comment("nest4"))
	testhelpers.RequireNotNil(t, b4, "nested bundle was nil")

	outerBundle := BundleOne(b4, MaxTime(500), b2, BatchSize(1000))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

func TestFindOneOpt(t *testing.T) {
	var bundle1 *OneBundle
	bundle1 = bundle1.ReturnKey(true).Comment("hello world").ReturnKey(false)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []option.Optioner{
		ReturnKey(true).ConvertFindOption(),
		Comment("hello world").ConvertFindOption(),
		ReturnKey(false).ConvertFindOption(),
	}
	bundle1DedupOpts := []option.Optioner{
		Comment("hello world").ConvertFindOption(),
		ReturnKey(false).ConvertFindOption(),
	}

	bundle2 := BundleOne(BatchSize(1))
	bundle2Opts := []option.Optioner{
		OptBatchSize(1).ConvertFindOption(),
	}

	bundle3 := BundleOne().
		BatchSize(1).
		Comment("Hello").
		BatchSize(2).
		ReturnKey(false).
		ReturnKey(true).
		Comment("World")

	bundle3Opts := []option.Optioner{
		OptBatchSize(1).ConvertFindOption(),
		OptComment("Hello").ConvertFindOption(),
		OptBatchSize(2).ConvertFindOption(),
		OptReturnKey(false).ConvertFindOption(),
		OptReturnKey(true).ConvertFindOption(),
		OptComment("World").ConvertFindOption(),
	}

	bundle3DedupOpts := []option.Optioner{
		OptBatchSize(2).ConvertFindOption(),
		OptReturnKey(true).ConvertFindOption(),
		OptComment("World").ConvertFindOption(),
	}

	nilBundle := BundleOne()
	var nilBundleOpts []option.Optioner

	nestedBundle1 := createNestedOneBundle1(t)
	nestedBundleOpts1 := []option.Optioner{
		OptAllowPartialResults(true).ConvertFindOption(),
		OptMaxTime(500).ConvertFindOption(),
		OptAllowPartialResults(false).ConvertFindOption(),
		OptComment("hello world nested").ConvertFindOption(),
		OptBatchSize(1000).ConvertFindOption(),
	}
	nestedBundleDedupOpts1 := []option.Optioner{
		OptMaxTime(500).ConvertFindOption(),
		OptAllowPartialResults(false).ConvertFindOption(),
		OptComment("hello world nested").ConvertFindOption(),
		OptBatchSize(1000).ConvertFindOption(),
	}

	nestedBundle2 := createNestedOneBundle2(t)
	nestedBundleOpts2 := []option.Optioner{
		OptAllowPartialResults(true).ConvertFindOption(),
		OptMaxTime(500).ConvertFindOption(),
		OptMaxTime(100).ConvertFindOption(),
		OptAllowPartialResults(false).ConvertFindOption(),
		OptComment("nest1").ConvertFindOption(),
		OptComment("nest2").ConvertFindOption(),
		OptBatchSize(1000).ConvertFindOption(),
	}
	nestedBundleDedupOpts2 := []option.Optioner{
		OptMaxTime(100).ConvertFindOption(),
		OptAllowPartialResults(false).ConvertFindOption(),
		OptComment("nest2").ConvertFindOption(),
		OptBatchSize(1000).ConvertFindOption(),
	}

	nestedBundle3 := createNestedOneBundle3(t)
	nestedBundleOpts3 := []option.Optioner{
		OptMaxTime(100).ConvertFindOption(),
		OptAllowPartialResults(true).ConvertFindOption(),
		OptComment("nest3").ConvertFindOption(),
		OptComment("nest4").ConvertFindOption(),
		OptMaxTime(500).ConvertFindOption(),
		OptMaxTime(100).ConvertFindOption(),
		OptAllowPartialResults(false).ConvertFindOption(),
		OptComment("nest1").ConvertFindOption(),
		OptComment("nest2").ConvertFindOption(),
		OptBatchSize(1000).ConvertFindOption(),
	}
	nestedBundleDedupOpts3 := []option.Optioner{
		OptMaxTime(100).ConvertFindOption(),
		OptAllowPartialResults(false).ConvertFindOption(),
		OptComment("nest2").ConvertFindOption(),
		OptBatchSize(1000).ConvertFindOption(),
	}

	t.Run("TestAll", func(t *testing.T) {
		c := &mongoopt.Collation{
			Locale: "string locale",
		}

		opts := []One{
			AllowPartialResults(true),
			BatchSize(5),
			Collation(c),
			Comment("hello world testing find"),
			CursorType(mongoopt.Tailable),
			Hint("hint for find"),
			Max("max for find"),
			MaxAwaitTime(100),
			MaxScan(1000),
			MaxTime(5000),
			Min("min for find"),
			NoCursorTimeout(false),
			OplogReplay(true),
			Projection("projection for find"),
			ReturnKey(true),
			ShowRecordID(false),
			Skip(50),
			Snapshot(false),
			Sort("sort for find"),
		}
		bundle := BundleOne(opts...)

		deleteOpts, err := bundle.Unbundle(true)
		testhelpers.RequireNil(t, err, "got non-nill error from unbundle: %s", err)

		if len(deleteOpts) != len(opts) {
			t.Errorf("expected unbundled opts len %d. got %d", len(opts), len(deleteOpts))
		}

		for i, opt := range opts {
			if !reflect.DeepEqual(opt.ConvertFindOneOption(), deleteOpts[i]) {
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
