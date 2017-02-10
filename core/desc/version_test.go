package desc_test

import (
	"testing"

	. "github.com/10gen/mongo-go-driver/core/desc"
)

func TestVersion_NewVersion(t *testing.T) {
	t.Parallel()

	subject := NewVersion(3, 4, 1)

	if subject.String() != "3.4.1" {
		t.Fatalf("expected 3.4.1 but got %s", subject.String())
	}
}

func TestVersion_AtLeast(t *testing.T) {
	t.Parallel()

	subject := NewVersion(3, 4, 1)

	tests := []struct {
		parts    []uint8
		expected bool
	}{
		{[]uint8{1, 0, 0}, true},
		{[]uint8{3, 0, 0}, true},
		{[]uint8{3, 4, 0}, true},
		{[]uint8{3, 4, 1}, true},
		{[]uint8{3, 4, 1, 0}, false},
		{[]uint8{3, 4, 1, 1}, false},
		{[]uint8{3, 4, 2}, false},
		{[]uint8{10, 0, 0}, false},
	}

	for _, test := range tests {
		actual := subject.AtLeast(test.parts...)
		if actual != test.expected {
			t.Fatalf("expected %v to be %t", test.parts, test.expected)
		}
	}
}
