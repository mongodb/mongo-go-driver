package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddr_String(t *testing.T) {
	tests := []struct {
		in       string
		expected string
	}{
		{"a", "a:27017"},
		{"A", "a:27017"},
		{"A:27017", "a:27017"},
		{"a:27017", "a:27017"},
		{"a.sock", "a.sock"},
	}

	for _, test := range tests {
		t.Run(test.in, func(t *testing.T) {
			require.Equal(t, Addr(test.in).String(), test.expected)
		})
	}
}

func TestAddr_Canonicalize(t *testing.T) {
	tests := []struct {
		in       string
		expected string
	}{
		{"a", "a:27017"},
		{"A", "a:27017"},
		{"A:27017", "a:27017"},
		{"a:27017", "a:27017"},
		{"a.sock", "a.sock"},
	}

	for _, test := range tests {
		t.Run(test.in, func(t *testing.T) {
			require.Equal(t, Addr(test.in).Canonicalize(), Addr(test.expected))
		})
	}
}
