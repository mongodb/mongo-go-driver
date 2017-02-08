package core_test

import (
	"testing"

	"reflect"

	. "github.com/10gen/mongo-go-driver/core"
)

func TestParseURI(t *testing.T) {

	tests := []URI{
		URI{
			Original: "mongodb://foo:27017,bar:27017",
			Hosts:    []string{"foo:27017", "bar:27017"},
		},
	}

	for _, test := range tests {
		actual, err := ParseURI(test.Original)
		if err != nil {
			t.Fatalf("failed parsing: %s", err)
		}

		if !reflect.DeepEqual(test, actual) {
			t.Fatalf("expected does not match actual\n  expected: %#v\n    actual: %#v", test, actual)
		}
	}
}
