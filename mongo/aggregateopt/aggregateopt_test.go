package aggregateopt

import (
	"reflect"
	"testing"

	"github.com/mongodb/mongo-go-driver/core/option"
)

func isNil(object interface{}) bool {
	if object == nil {
		return true
	}

	value := reflect.ValueOf(object)
	kind := value.Kind()

	// checking to see if type is Chan, Func, Interface, Map, Ptr, or Slice
	if kind >= reflect.Chan && kind <= reflect.Slice && value.IsNil() {
		return true
	}

	return false
}

// throw error if var is nil
func requireNotNil(t *testing.T, variable interface{}, msgFormat string, msgVars ...interface{}) {
	if isNil(variable) {
		t.Errorf(msgFormat, msgVars...)
	}
}

// throw error if var is not nil
func requireNil(t *testing.T, variable interface{}, msgFormat string, msgVars ...interface{}) {
	if !isNil(variable) {
		t.Errorf(msgFormat, msgVars...)
	}
}

func createNestedBundle(t *testing.T) *AggregateBundle {
	nestedBundle := BundleAggregate(AllowDiskUse(false), Comment("hello world nested"))
	requireNotNil(t, nestedBundle, "nested bundle was nil")

	outerBundle := BundleAggregate(AllowDiskUse(true), MaxTime(500), nestedBundle, BatchSize(1000))
	requireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

func TestAggregateOpt(t *testing.T) {
	var bundle1 *AggregateBundle
	bundle1 = bundle1.BypassDocumentValidation(true).Comment("hello world").BypassDocumentValidation(false)
	requireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []option.Optioner{
		BypassDocumentValidation(true).ConvertOption(),
		Comment("hello world").ConvertOption(),
		BypassDocumentValidation(false).ConvertOption(),
	}
	bundle1DedupOpts := []option.Optioner{
		Comment("hello world").ConvertOption(),
		BypassDocumentValidation(false).ConvertOption(),
	}

	bundle2 := BundleAggregate(BatchSize(1))
	bundle2Opts := []option.Optioner{
		OptBatchSize(1).ConvertOption(),
	}

	bundle3 := BundleAggregate().
		BatchSize(1).
		Comment("Hello").
		BatchSize(2).
		BypassDocumentValidation(false).
		BypassDocumentValidation(true).
		Comment("World")

	bundle3Opts := []option.Optioner{
		OptBatchSize(1).ConvertOption(),
		OptComment("Hello").ConvertOption(),
		OptBatchSize(2).ConvertOption(),
		OptBypassDocumentValidation(false).ConvertOption(),
		OptBypassDocumentValidation(true).ConvertOption(),
		OptComment("World").ConvertOption(),
	}

	bundle3DedupOpts := []option.Optioner{
		OptBatchSize(2).ConvertOption(),
		OptBypassDocumentValidation(true).ConvertOption(),
		OptComment("World").ConvertOption(),
	}

	nilBundle := BundleAggregate()
	nilBundleOpts := []option.Optioner{}

	nestedBundle := createNestedBundle(t)
	nestedBundleOpts := []option.Optioner{
		OptAllowDiskUse(true).ConvertOption(),
		OptMaxTime(500).ConvertOption(),
		OptAllowDiskUse(false).ConvertOption(),
		OptComment("hello world nested").ConvertOption(),
		OptBatchSize(1000).ConvertOption(),
	}
	nestedBundleDedupOpts := []option.Optioner{
		OptMaxTime(500).ConvertOption(),
		OptAllowDiskUse(false).ConvertOption(),
		OptComment("hello world nested").ConvertOption(),
		OptBatchSize(1000).ConvertOption(),
	}

	t.Run("MakeOptions", func(t *testing.T) {
		head := bundle1

		bundleLen := 0
		for head != nil && head.option != nil {
			bundleLen++
			head = head.next
		}

		if bundleLen != 3 {
			t.Errorf("expected bundle length 3. got: %d", bundleLen)
		}
	})

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name         string
			dedup        bool
			bundle       *AggregateBundle
			expectedOpts []option.Optioner
		}{
			{"NilBundle", false, nilBundle, nilBundleOpts},
			{"Bundle1", false, bundle1, bundle1Opts},
			{"Bundle1Dedup", true, bundle1, bundle1DedupOpts},
			{"Bundle2", false, bundle2, bundle2Opts},
			{"Bundle2Dedup", true, bundle2, bundle2Opts},
			{"Bundle3", false, bundle3, bundle3Opts},
			{"Bundle3Dedup", true, bundle3, bundle3DedupOpts},
			{"NestedBundle_DedupFalse", false, nestedBundle, nestedBundleOpts},
			{"NestedBundle_DedupTrue", true, nestedBundle, nestedBundleDedupOpts},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				options, err := tc.bundle.Unbundle(tc.dedup)
				requireNil(t, err, "got non-nill error from unbundle: %s", err)

				if len(options) != len(tc.expectedOpts) {
					t.Errorf("options length does not match expected length. got %d expected %d", len(options),
						len(tc.expectedOpts))
				}
				for i, opt := range options {
					if tc.expectedOpts[i] != opt {
						t.Errorf("expected: %s\nreceived: %s", opt, tc.expectedOpts[i])
					}
				}
			})
		}
	})
}
