package aggregateopt

import (
	"reflect"
	"testing"
)

func isNil(object interface{}) bool {
	if object == nil {
		return true
	}

	value := reflect.ValueOf(object)
	kind := value.Kind()
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

func TestAggregateOpt(t *testing.T) {
	var basicBundle *AggregateBundle
	basicBundle = basicBundle.BypassDocumentValidation(true).Comment("hello world").BypassDocumentValidation(false)
	requireNotNil(t, basicBundle, "created bundle was nil")

	t.Run("MakeOptions", func(t *testing.T) {
		head := basicBundle

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
			name        string
			dedup       bool
			expectedLen int
			bundle      *AggregateBundle
		}{
			{"DedupFalse", false, 3, basicBundle},
			{"DedupTrue", true, 2, basicBundle},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				options, err := tc.bundle.Unbundle(tc.dedup)
				requireNil(t, err, "got non-nill error from unbundle: %s", err)

				if len(options) != tc.expectedLen {
					t.Errorf("options length does not match expected length. got %d expected %d", len(options),
						tc.expectedLen)
				}
			})
		}
	})
}
