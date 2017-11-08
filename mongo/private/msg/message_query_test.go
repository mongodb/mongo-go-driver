// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package msg_test

import (
	"testing"

	"github.com/10gen/mongo-go-driver/bson"
	. "github.com/10gen/mongo-go-driver/mongo/private/msg"
	"github.com/stretchr/testify/require"
)

func TestWrapWithMeta(t *testing.T) {
	req := NewCommand(10, "admin", true, bson.M{"a": 1}).(*Query)

	buf, err := bson.Marshal(req.Query)
	require.NoError(t, err)
	var actual bson.D
	err = bson.Unmarshal(buf, &actual)
	require.NoError(t, err)
	expected := bson.D{
		bson.NewDocElem("a", 1),
	}
	require.Equal(t, expected, actual)

	AddMeta(req, map[string]interface{}{
		"$readPreference": bson.M{
			"mode": "secondary",
		},
	})

	buf, err = bson.Marshal(req.Query)
	require.NoError(t, err)
	err = bson.Unmarshal(buf, &actual)
	require.NoError(t, err)
	expected = bson.D{
		bson.NewDocElem("$query",
			bson.D{bson.NewDocElem("a", 1)},
		),
		bson.NewDocElem("$readPreference",
			bson.D{bson.NewDocElem("mode", "secondary")},
		),
	}
	require.Equal(t, expected, actual)
}
