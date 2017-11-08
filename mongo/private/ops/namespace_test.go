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
