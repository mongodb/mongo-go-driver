package sessionopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
)

func createNestedBundle1(t *testing.T) *SessionBundle {
	nested := BundleSession(CausalConsistency(false))
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	outer := BundleSession(CausalConsistency(false), CausalConsistency(true), nested)
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	return outer
}

func createdNestedBundle2(t *testing.T) *SessionBundle {
	b1 := BundleSession(CausalConsistency(false))
	testhelpers.RequireNotNil(t, b1, "b1 was nil")

	b2 := BundleSession(b1, CausalConsistency(true))
	testhelpers.RequireNotNil(t, b2, "b2 was nil")

	outer := BundleSession(CausalConsistency(true), CausalConsistency(false), b2, CausalConsistency(true))
	testhelpers.RequireNotNil(t, outer, "outer was nil")

	return outer
}

func TestSessionOpt(t *testing.T) {
	nilBundle := BundleSession()
	var nilOpts []session.ClientOptioner

	var bundle1 *SessionBundle
	bundle1 = bundle1.CausalConsistency(true).CausalConsistency(false)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []session.ClientOptioner{
		CausalConsistency(true).ConvertSessionOption(),
		CausalConsistency(false).ConvertSessionOption(),
	}

	bundle1DedupOpts := []session.ClientOptioner{
		CausalConsistency(false).ConvertSessionOption(),
	}

	bundle2 := BundleSession(CausalConsistency(true))
	bundle2Opts := []session.ClientOptioner{
		CausalConsistency(true).ConvertSessionOption(),
	}

	nested1 := createNestedBundle1(t)
	nested1Opts := []session.ClientOptioner{
		CausalConsistency(false).ConvertSessionOption(),
		CausalConsistency(true).ConvertSessionOption(),
		CausalConsistency(false).ConvertSessionOption(),
	}
	nested1DedupOpts := []session.ClientOptioner{
		CausalConsistency(false).ConvertSessionOption(),
	}

	nested2 := createdNestedBundle2(t)
	nested2Opts := []session.ClientOptioner{
		CausalConsistency(true).ConvertSessionOption(),
		CausalConsistency(false).ConvertSessionOption(),
		CausalConsistency(false).ConvertSessionOption(),
		CausalConsistency(true).ConvertSessionOption(),
		CausalConsistency(true).ConvertSessionOption(),
	}
	nested2DedupOpts := []session.ClientOptioner{
		CausalConsistency(true).ConvertSessionOption(),
	}

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name         string
			bundle       *SessionBundle
			expectedOpts []session.ClientOptioner
			dedup        bool
		}{
			{"NilBundle", nilBundle, nilOpts, false},
			{"NilBundle", nilBundle, nilOpts, true},
			{"Bundle1", bundle1, bundle1Opts, false},
			{"Bundle1Dedup", bundle1, bundle1DedupOpts, true},
			{"Bundle2", bundle2, bundle2Opts, false},
			{"Bundle2", bundle2, bundle2Opts, true},
			{"Nested1", nested1, nested1Opts, false},
			{"Nested1", nested1, nested1DedupOpts, true},
			{"Nested2", nested2, nested2Opts, false},
			{"Nested2", nested2, nested2DedupOpts, true},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				opts, err := tc.bundle.Unbundle(tc.dedup)
				testhelpers.RequireNil(t, err, "error unbundling: %s", err)

				if len(opts) != len(tc.expectedOpts) {
					t.Fatalf("options length mismatch; expected %d got %d", len(tc.expectedOpts), len(opts))
				}

				for i, opt := range opts {
					if !reflect.DeepEqual(opt, tc.expectedOpts[i]) {
						t.Fatalf("expected opt %s got %s", tc.expectedOpts[i], opt)
					}
				}
			})
		}
	})
}
