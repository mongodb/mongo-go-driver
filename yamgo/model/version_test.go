package model_test

import (
	"testing"

	. "github.com/10gen/mongo-go-driver/yamgo/model"
	"github.com/stretchr/testify/require"
)

func TestVersion_AtLeast(t *testing.T) {
	t.Parallel()

	subject := Version{
		Parts: []uint8{3, 4, 0},
	}

	tests := []struct {
		version  Version
		expected bool
	}{
		{Version{Parts: []uint8{1, 0, 0}}, true},
		{Version{Parts: []uint8{3, 0, 0}}, true},
		{Version{Parts: []uint8{3, 4}}, true},
		{Version{Parts: []uint8{3, 4, 0}}, true},
		{Version{Parts: []uint8{3, 4, 1}}, false},
		{Version{Parts: []uint8{3, 4, 1, 0}}, false},
		{Version{Parts: []uint8{3, 4, 1, 1}}, false},
		{Version{Parts: []uint8{3, 4, 2}}, false},
		{Version{Parts: []uint8{3, 5}}, false},
		{Version{Parts: []uint8{10, 0, 0}}, false},
	}

	for _, test := range tests {
		t.Run(test.version.String(), func(t *testing.T) {
			actual := subject.AtLeast(test.version.Parts...)
			require.Equal(t, test.expected, actual)
		})

	}
}
