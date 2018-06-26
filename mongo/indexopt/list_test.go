package indexopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
)

func createNestedBundle1List(t *testing.T) *ListBundle {
	nested := BundleList(MaxTime(10))
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	outer := BundleList(MaxTime(10), BatchSize(5), nested)
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	return outer
}

func createNestedBundle2List(t *testing.T) *ListBundle {
	b1 := BundleList(BatchSize(10))
	testhelpers.RequireNotNil(t, b1, "b1 was nil")

	b2 := BundleList(MaxTime(15), b1)
	testhelpers.RequireNotNil(t, b2, "b2 was nil")

	outer := BundleList(BatchSize(5), MaxTime(20), b2)
	testhelpers.RequireNotNil(t, outer, "outer was nil")

	return outer
}

func createNestedBundle3List(t *testing.T) *ListBundle {
	b1 := BundleList(BatchSize(5))
	testhelpers.RequireNotNil(t, b1, "b1 was nil")

	b2 := BundleList(MaxTime(10), b1)
	testhelpers.RequireNotNil(t, b2, "b2 was nil")

	b3 := BundleList(MaxTime(11))
	testhelpers.RequireNotNil(t, b3, "b3 was nil")

	b4 := BundleList(BatchSize(10), b3)
	testhelpers.RequireNotNil(t, b4, "b4 was nil")

	outer := BundleList(b4, MaxTime(1), b2)
	testhelpers.RequireNotNil(t, outer, "outer was nil")

	return outer
}

func TestListOpt(t *testing.T) {
	nilBundle := BundleList()
	var nilOpts []option.ListIndexesOptioner

	var bundle1 *ListBundle
	bundle1 = bundle1.BatchSize(5).BatchSize(5)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []option.ListIndexesOptioner{
		BatchSize(5).ConvertListOption(),
		BatchSize(5).ConvertListOption(),
	}
	bundle1OptsDedup := []option.ListIndexesOptioner{
		BatchSize(5).ConvertListOption(),
	}

	bundle2 := BundleList(BatchSize(5))
	bundle2Opts := []option.ListIndexesOptioner{
		BatchSize(5).ConvertListOption(),
	}

	nested1 := createNestedBundle1List(t)
	nestedOpts := []option.ListIndexesOptioner{
		MaxTime(10).ConvertListOption(),
		BatchSize(5).ConvertListOption(),
		MaxTime(10).ConvertListOption(),
	}
	nestedOptsDedup := []option.ListIndexesOptioner{
		BatchSize(5).ConvertListOption(),
		MaxTime(10).ConvertListOption(),
	}

	nested2 := createNestedBundle2List(t)
	nested2Opts := []option.ListIndexesOptioner{
		BatchSize(5).ConvertListOption(),
		MaxTime(20).ConvertListOption(),
		MaxTime(15).ConvertListOption(),
		BatchSize(10).ConvertListOption(),
	}
	nested2OptsDedup := []option.ListIndexesOptioner{
		MaxTime(15).ConvertListOption(),
		BatchSize(10).ConvertListOption(),
	}

	nested3 := createNestedBundle3List(t)
	nested3Opts := []option.ListIndexesOptioner{
		BatchSize(10).ConvertListOption(),
		MaxTime(11).ConvertListOption(),
		MaxTime(1).ConvertListOption(),
		MaxTime(10).ConvertListOption(),
		BatchSize(5).ConvertListOption(),
	}
	nested3OptsDedup := []option.ListIndexesOptioner{
		MaxTime(10).ConvertListOption(),
		BatchSize(5).ConvertListOption(),
	}

	t.Run("TestAll", func(t *testing.T) {
		opts := []List{
			BatchSize(5),
			MaxTime(5000),
		}
		bundle := BundleList(opts...)

		deleteOpts, err := bundle.Unbundle(true)
		testhelpers.RequireNil(t, err, "got non-nill error from unbundle: %s", err)

		if len(deleteOpts) != len(opts) {
			t.Errorf("expected unbundled opts len %d. got %d", len(opts), len(deleteOpts))
		}

		for i, opt := range opts {
			if !reflect.DeepEqual(opt.ConvertListOption(), deleteOpts[i]) {
				t.Errorf("opt mismatch. expected %#v, got %#v", opt, deleteOpts[i])
			}
		}
	})

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name         string
			bundle       *ListBundle
			dedup        bool
			expectedOpts []option.ListIndexesOptioner
		}{
			{"NilBundle", nilBundle, false, nilOpts},
			{"Bundle1", bundle1, false, bundle1Opts},
			{"Bundle1Dedup", bundle1, true, bundle1OptsDedup},
			{"Bundle2", bundle2, false, bundle2Opts},
			{"Bundle2", bundle2, true, bundle2Opts},
			{"Nested1", nested1, false, nestedOpts},
			{"Nested1Dedup", nested1, true, nestedOptsDedup},
			{"Nested2", nested2, false, nested2Opts},
			{"Nested2Dedup", nested2, true, nested2OptsDedup},
			{"Nested3", nested3, false, nested3Opts},
			{"Nested3Dedup", nested3, true, nested3OptsDedup},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				opts, err := tc.bundle.Unbundle(tc.dedup)
				testhelpers.RequireNil(t, err, "err unbundling db: %s", err)

				if len(opts) != len(tc.expectedOpts) {
					t.Errorf("expectedOpts len mismatch. expected %d, got %d", len(tc.expectedOpts), len(opts))
				}

				for i, opt := range opts {
					if !reflect.DeepEqual(opt, tc.expectedOpts[i]) {
						t.Errorf("expected: %s\nreceived: %s", opt, tc.expectedOpts[i])
					}
				}
			})
		}
	})
}
