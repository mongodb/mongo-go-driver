package server_test

import (
	"testing"

	. "github.com/10gen/mongo-go-driver/server"
	"github.com/stretchr/testify/require"
)

func TestTagSets_Contains(t *testing.T) {
	t.Parallel()

	ts := NewTagSet("a", "1")

	require.True(t, ts.Contains("a", "1"))
	require.False(t, ts.Contains("1", "a"))
	require.False(t, ts.Contains("A", "1"))
	require.False(t, ts.Contains("a", "10"))
}

func TestTagSets_ContainsAll(t *testing.T) {
	t.Parallel()

	ts := NewTagSet("a", "1", "b", "2")

	test := NewTagSet("a", "1")
	require.True(t, ts.ContainsAll(test))
	test = NewTagSet("a", "1", "b", "2")
	require.True(t, ts.ContainsAll(test))
	test = NewTagSet("a", "1", "b", "2")
	require.True(t, ts.ContainsAll(test))

	test = NewTagSet("a", "2", "b", "1")
	require.False(t, ts.ContainsAll(test))
	test = NewTagSet("a", "1", "b", "1")
	require.False(t, ts.ContainsAll(test))
	test = NewTagSet("a", "2", "b", "2")
	require.False(t, ts.ContainsAll(test))
}
