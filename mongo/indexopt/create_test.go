package indexopt

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
)

var wc1 = writeconcern.New(writeconcern.W(10))
var wc2 = writeconcern.New(writeconcern.W(20))

func createNestedBundle1(t *testing.T) *CreateBundle {
	nested := BundleCreate(MaxTime(10))
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	outer := BundleCreate(MaxTime(10), WriteConcern(wc1), nested)
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	return outer
}

func createNestedBundle2(t *testing.T) *CreateBundle {
	b1 := BundleCreate(WriteConcern(wc2))
	testhelpers.RequireNotNil(t, b1, "b1 was nil")

	b2 := BundleCreate(MaxTime(15), b1)
	testhelpers.RequireNotNil(t, b2, "b2 was nil")

	outer := BundleCreate(WriteConcern(wc1), MaxTime(20), b2)
	testhelpers.RequireNotNil(t, outer, "outer was nil")

	return outer
}

func createNestedBundle3(t *testing.T) *CreateBundle {
	b1 := BundleCreate(WriteConcern(wc1))
	testhelpers.RequireNotNil(t, b1, "b1 was nil")

	b2 := BundleCreate(MaxTime(10), b1)
	testhelpers.RequireNotNil(t, b2, "b2 was nil")

	b3 := BundleCreate(MaxTime(11))
	testhelpers.RequireNotNil(t, b3, "b3 was nil")

	b4 := BundleCreate(WriteConcern(wc2), b3)
	testhelpers.RequireNotNil(t, b4, "b4 was nil")

	outer := BundleCreate(b4, MaxTime(1), b2)
	testhelpers.RequireNotNil(t, outer, "outer was nil")

	return outer
}

func TestCreateOpt(t *testing.T) {
	nilBundle := BundleCreate()
	var nilOpts []option.CreateIndexesOptioner

	var bundle1 *CreateBundle
	bundle1 = bundle1.WriteConcern(wc1).WriteConcern(wc1)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []option.CreateIndexesOptioner{
		WriteConcern(wc1).ConvertCreateOption(),
		WriteConcern(wc1).ConvertCreateOption(),
	}
	bundle1OptsDedup := []option.CreateIndexesOptioner{
		WriteConcern(wc1).ConvertCreateOption(),
	}

	bundle2 := BundleCreate(WriteConcern(wc1))
	bundle2Opts := []option.CreateIndexesOptioner{
		WriteConcern(wc1).ConvertCreateOption(),
	}

	nested1 := createNestedBundle1(t)
	nestedOpts := []option.CreateIndexesOptioner{
		MaxTime(10).ConvertCreateOption(),
		WriteConcern(wc1).ConvertCreateOption(),
		MaxTime(10).ConvertCreateOption(),
	}
	nestedOptsDedup := []option.CreateIndexesOptioner{
		WriteConcern(wc1).ConvertCreateOption(),
		MaxTime(10).ConvertCreateOption(),
	}

	nested2 := createNestedBundle2(t)
	nested2Opts := []option.CreateIndexesOptioner{
		WriteConcern(wc1).ConvertCreateOption(),
		MaxTime(20).ConvertCreateOption(),
		MaxTime(15).ConvertCreateOption(),
		WriteConcern(wc2).ConvertCreateOption(),
	}
	nested2OptsDedup := []option.CreateIndexesOptioner{
		MaxTime(15).ConvertCreateOption(),
		WriteConcern(wc2).ConvertCreateOption(),
	}

	nested3 := createNestedBundle3(t)
	nested3Opts := []option.CreateIndexesOptioner{
		WriteConcern(wc2).ConvertCreateOption(),
		MaxTime(11).ConvertCreateOption(),
		MaxTime(1).ConvertCreateOption(),
		MaxTime(10).ConvertCreateOption(),
		WriteConcern(wc1).ConvertCreateOption(),
	}
	nested3OptsDedup := []option.CreateIndexesOptioner{
		MaxTime(10).ConvertCreateOption(),
		WriteConcern(wc1).ConvertCreateOption(),
	}

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name   string
			bundle *CreateBundle
			dedup  bool
			opts   []option.CreateIndexesOptioner
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

				if len(opts) != len(tc.opts) {
					t.Errorf("opts len mismatch. expected %d, got %d", len(tc.opts), len(opts))
				}

				for i, opt := range opts {
					if opt != tc.opts[i] {
						t.Errorf("expected: %s\nreceived: %s", opt, tc.opts[i])
					}
				}
			})
		}
	})
}
