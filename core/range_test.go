package core_test

import (
	"testing"

	. "github.com/10gen/mongo-go-driver/core"
)

func TestRange_Includes(t *testing.T) {
	subject := Range{1, 3}

	tests := []struct {
		n        uint8
		expected bool
	}{
		{0, false},
		{1, true},
		{2, true},
		{3, true},
		{4, false},
		{10, false},
	}

	for _, test := range tests {
		actual := subject.Includes(test.n)
		if actual != test.expected {
			t.Fatalf("expected %v to be %t", test.n, test.expected)
		}
	}
}
