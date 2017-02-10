package core

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewNamespace(t *testing.T) {
	ns, err := NewNamespace("foo", "bar.baz")
	require.Nil(t, err)
	require.Equal(t, "foo", ns.DatabaseName())
	require.Equal(t, "bar.baz", ns.CollectionName())
	require.Equal(t, "foo.bar.baz", ns.FullName())

	ns, err = NewNamespace("bar.baz", "foo")
	require.NotNil(t, err)

	ns, err = NewNamespace("bar baz", "foo")
	require.NotNil(t, err)

	ns, err = NewNamespace("bar.baz", "")
	require.NotNil(t, err)

	ns, err = NewNamespace("", "foo")
	require.NotNil(t, err)
}

func TestParseNamespace(t *testing.T) {
	ns, err := ParseNamespace("foo.bar.baz")
	require.Nil(t, err)
	require.Equal(t, "foo", ns.DatabaseName())
	require.Equal(t, "bar.baz", ns.CollectionName())
	require.Equal(t, "foo.bar.baz", ns.FullName())

	ns, err = ParseNamespace("foo")
	require.NotNil(t, err)

	ns, err = ParseNamespace(".foo")
	require.NotNil(t, err)

	ns, err = ParseNamespace("foo.")
	require.NotNil(t, err)

	ns, err = ParseNamespace("fo o.bar")
	require.NotNil(t, err)
}
