// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ops

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNamespace(t *testing.T) {
	t.Parallel()

	ns := Namespace{"foo", "bar.baz"}
	require.NoError(t, ns.validate())
	require.Equal(t, "foo.bar.baz", ns.FullName())

	ns = Namespace{"bar.baz", "foo"}
	require.Error(t, ns.validate())
	require.Error(t, validateDB(ns.DB))

	ns = Namespace{"bar baz", "foo"}
	require.Error(t, ns.validate())
	require.Error(t, validateDB(ns.DB))

	ns = Namespace{"bar", ""}
	require.Error(t, ns.validate())
	require.Error(t, validateCollection(ns.Collection))

	ns = Namespace{"", "foo"}
	require.Error(t, ns.validate())
	require.Error(t, validateDB(ns.DB))
}
