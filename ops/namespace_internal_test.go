package ops

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNamespace(t *testing.T) {
	t.Parallel()

	ns := Namespace{"foo", "bar.baz"}
	require.Nil(t, ns.validate())
	require.Equal(t, "foo.bar.baz", ns.FullName())

	ns = Namespace{"bar.baz", "foo"}
	require.NotNil(t, ns.validate())
	require.NotNil(t, validateDB(ns.DB))

	ns = Namespace{"bar baz", "foo"}
	require.NotNil(t, ns.validate())
	require.NotNil(t, validateDB(ns.DB))

	ns = Namespace{"bar", ""}
	require.NotNil(t, ns.validate())
	require.NotNil(t, validateCollection(ns.Collection))

	ns = Namespace{"", "foo"}
	require.NotNil(t, ns.validate())
	require.NotNil(t, validateDB(ns.DB))
}
