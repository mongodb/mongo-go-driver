package sessionopt

import (
	"testing"

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
	var nilSession = &Session{}

	var bundle1 *SessionBundle
	bundle1 = bundle1.CausalConsistency(true).CausalConsistency(false)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Session := &Session{
		Consistent: false,
	}

	bundle2 := BundleSession(CausalConsistency(true))
	bundle2Session := &Session{
		Consistent: true,
	}

	nested1 := createNestedBundle1(t)
	nested1Session := &Session{
		Consistent: false,
	}

	nested2 := createdNestedBundle2(t)
	nested2Session := &Session{
		Consistent: true,
	}

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name   string
			bundle *SessionBundle
			s      *Session
		}{
			{"NilBundle", nilBundle, nilSession},
			{"Bundle1", bundle1, bundle1Session},
			{"Bundle2", bundle2, bundle2Session},
			{"Nested1", nested1, nested1Session},
			{"Nested2", nested2, nested2Session},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				sess, err := tc.bundle.Unbundle()
				testhelpers.RequireNil(t, err, "err unbundling session: %s", err)

				switch {
				case sess.Consistent != tc.s.Consistent:
					t.Fatalf("consistent mismatch. expected %v got %v", tc.s.Consistent, sess.Consistent)
				}
			})
		}
	})
}
