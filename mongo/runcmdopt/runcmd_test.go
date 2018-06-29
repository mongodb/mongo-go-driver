package runcmdopt

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
)

var rpPrimary = readpref.Primary()
var rpSeconadary = readpref.Secondary()

func createNestedBundle1(t *testing.T) *RunCmdBundle {
	nested := BundleRunCmd(ReadPreference(rpPrimary))
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	outer := BundleRunCmd(ReadPreference(rpSeconadary), nested)
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	return outer
}

func TestRunCmdOpt(t *testing.T) {
	nilBundle := BundleRunCmd()
	var nilRc = &RunCmd{}

	var bundle1 *RunCmdBundle
	bundle1 = bundle1.ReadPreference(rpSeconadary)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Rc := &RunCmd{
		ReadPreference: rpSeconadary,
	}

	bundle2 := BundleRunCmd(ReadPreference(rpPrimary))
	testhelpers.RequireNotNil(t, bundle2, "created bundle was nil")
	bundle2Rc := &RunCmd{
		ReadPreference: rpPrimary,
	}

	nested1 := createNestedBundle1(t)
	nested1Rc := &RunCmd{
		ReadPreference: rpPrimary,
	}

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name   string
			bundle *RunCmdBundle
			rc     *RunCmd
		}{
			{"NilBundle", nilBundle, nilRc},
			{"Bundle1", bundle1, bundle1Rc},
			{"Bundle2", bundle2, bundle2Rc},
			{"Nested1", nested1, nested1Rc},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				rc, err := tc.bundle.Unbundle()
				testhelpers.RequireNil(t, err, "err unbundling rc: %s", err)

				switch {
				case rc.ReadPreference != tc.rc.ReadPreference:
					t.Errorf("read preferences don't match")
				}
			})
		}
	})
}
