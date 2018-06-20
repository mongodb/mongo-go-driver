package deleteopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var c = &mongoopt.Collation{}
var c1 = &mongoopt.Collation{
	Locale: "string locale 1",
}

var c2 = &mongoopt.Collation{
	Locale: "string locale 1",
}

func createNestedDeleteBundle1(t *testing.T) *DeleteBundle {
	nestedBundle := BundleDelete(Collation(c))
	testhelpers.RequireNotNil(t, nestedBundle, "nested bundle was nil")

	outerBundle := BundleDelete(Collation(c), Collation(c1), nestedBundle)
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test doubly nested bundle
func createNestedDeleteBundle2(t *testing.T) *DeleteBundle {
	b1 := BundleDelete(Collation(c))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleDelete(Collation(c2), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	outerBundle := BundleDelete(Collation(c), Collation(c1), b2)
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test two top level nested bundles
func createNestedDeleteBundle3(t *testing.T) *DeleteBundle {
	b1 := BundleDelete(Collation(c))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleDelete(Collation(c2), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	b3 := BundleDelete(Collation(c))
	testhelpers.RequireNotNil(t, b3, "nested bundle was nil")

	b4 := BundleDelete(Collation(c2), b3)
	testhelpers.RequireNotNil(t, b4, "nested bundle was nil")

	outerBundle := BundleDelete(b4, Collation(c1), b2)
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

func TestDeleteOpt(t *testing.T) {
	var bundle1 *DeleteBundle
	bundle1 = bundle1.Collation(c).Collation(c)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []option.Optioner{
		Collation(c).ConvertDeleteOption(),
		Collation(c).ConvertDeleteOption(),
	}
	bundle1DedupOpts := []option.Optioner{
		Collation(c).ConvertDeleteOption(),
	}

	bundle2 := BundleDelete(Collation(c))
	bundle2Opts := []option.Optioner{
		Collation(c).ConvertDeleteOption(),
	}

	bundle3 := BundleDelete().
		Collation(c).
		Collation(c)

	bundle3Opts := []option.Optioner{
		OptCollation{c.Convert()}.ConvertDeleteOption(),
		OptCollation{c.Convert()}.ConvertDeleteOption(),
	}

	bundle3DedupOpts := []option.Optioner{
		OptCollation{c.Convert()}.ConvertDeleteOption(),
	}

	nilBundle := BundleDelete()
	var nilBundleOpts []option.Optioner

	nestedBundle1 := createNestedDeleteBundle1(t)
	nestedBundleOpts1 := []option.Optioner{
		OptCollation{c.Convert()}.ConvertDeleteOption(),
		OptCollation{c1.Convert()}.ConvertDeleteOption(),
		OptCollation{c.Convert()}.ConvertDeleteOption(),
	}
	nestedBundleDedupOpts1 := []option.Optioner{
		OptCollation{c.Convert()}.ConvertDeleteOption(),
	}

	nestedBundle2 := createNestedDeleteBundle2(t)
	nestedBundleOpts2 := []option.Optioner{
		OptCollation{c.Convert()}.ConvertDeleteOption(),
		OptCollation{c1.Convert()}.ConvertDeleteOption(),
		OptCollation{c2.Convert()}.ConvertDeleteOption(),
		OptCollation{c.Convert()}.ConvertDeleteOption(),
	}
	nestedBundleDedupOpts2 := []option.Optioner{
		OptCollation{c.Convert()}.ConvertDeleteOption(),
	}

	nestedBundle3 := createNestedDeleteBundle3(t)
	nestedBundleOpts3 := []option.Optioner{
		OptCollation{c2.Convert()}.ConvertDeleteOption(),
		OptCollation{c.Convert()}.ConvertDeleteOption(),
		OptCollation{c1.Convert()}.ConvertDeleteOption(),
		OptCollation{c2.Convert()}.ConvertDeleteOption(),
		OptCollation{c.Convert()}.ConvertDeleteOption(),
	}
	nestedBundleDedupOpts3 := []option.Optioner{
		OptCollation{c.Convert()}.ConvertDeleteOption(),
	}

	t.Run("TestAll", func(t *testing.T) {
		c := &mongoopt.Collation{
			Locale: "string locale",
		}

		opts := []Delete{
			Collation(c),
		}
		bundle := BundleDelete(opts...)

		deleteOpts, err := bundle.Unbundle(true)
		testhelpers.RequireNil(t, err, "got non-nil error from unbundle: %s", err)

		if len(deleteOpts) != len(opts) {
			t.Errorf("expected unbundled opts len %d. got %d", len(opts), len(deleteOpts))
		}

		for i, opt := range opts {
			if !reflect.DeepEqual(opt.ConvertDeleteOption(), deleteOpts[i]) {
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
			bundle       *DeleteBundle
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
