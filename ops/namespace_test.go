package ops

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParseNamespace(t *testing.T) {
	t.Parallel()

	ns := ParseNamespace("foo.bar.baz")
	require.Equal(t, "foo", ns.DB)
	require.Equal(t, "bar.baz", ns.Collection)
	require.Equal(t, "foo.bar.baz", ns.FullName())

	ns = ParseNamespace("foo")
	require.Equal(t, Namespace{}, ns)

	ns = ParseNamespace(".foo")
	require.Equal(t, Namespace{"", "foo"}, ns)

	ns = ParseNamespace("foo.")
	require.Equal(t, Namespace{"foo", ""}, ns)
}
