package dropcollopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
)

func TestDropCollOpt(t *testing.T) {

	t.Run("TestAll", func(t *testing.T) {
		opts := []DropCollOption{}
		params := make([]DropColl, len(opts))
		for i := range opts {
			params[i] = opts[i]
		}
		bundle := BundleDropColl(params...)

		deleteOpts, _, err := bundle.Unbundle(true)
		testhelpers.RequireNil(t, err, "got non-nill error from unbundle: %s", err)

		if len(deleteOpts) != len(opts) {
			t.Errorf("expected unbundled opts len %d. got %d", len(opts), len(deleteOpts))
		}

		for i, opt := range opts {
			if !reflect.DeepEqual(opt.ConvertDropCollOption(), deleteOpts[i]) {
				t.Errorf("opt mismatch. expected %#v, got %#v", opt, deleteOpts[i])
			}
		}
	})
}
