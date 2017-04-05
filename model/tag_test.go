package model_test

import (
	"testing"

	. "github.com/10gen/mongo-go-driver/model"
	"github.com/stretchr/testify/require"
)

func TestTagSets_NewTagSet(t *testing.T) {
	t.Parallel()

	ts := NewTagSet("a", "1")

	require.True(t, ts.Contains("a", "1"))
	require.False(t, ts.Contains("1", "a"))
	require.False(t, ts.Contains("A", "1"))
	require.False(t, ts.Contains("a", "10"))
}

func TestTagSets_NewTagSetFromMap(t *testing.T) {
	t.Parallel()

	ts := NewTagSetFromMap(map[string]string{"a": "1"})

	require.True(t, ts.Contains("a", "1"))
	require.False(t, ts.Contains("1", "a"))
	require.False(t, ts.Contains("A", "1"))
	require.False(t, ts.Contains("a", "10"))
}

func TestTagSets_NewTagSetsFromMaps(t *testing.T) {
	t.Parallel()

	tss := NewTagSetsFromMaps([]map[string]string{{"a": "1"}, {"b": "1"}})

	require.Len(t, tss, 2)

	ts := tss[0]
	require.True(t, ts.Contains("a", "1"))
	require.False(t, ts.Contains("1", "a"))
	require.False(t, ts.Contains("A", "1"))
	require.False(t, ts.Contains("a", "10"))

	ts = tss[1]
	require.True(t, ts.Contains("b", "1"))
	require.False(t, ts.Contains("1", "b"))
	require.False(t, ts.Contains("B", "1"))
	require.False(t, ts.Contains("b", "10"))
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
