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
	req := NewCommand(10, "admin", true, bson.NewDocument(bson.C.Int32("a", 1))).(*Query)

	AddMeta(req, map[string]*bson.Document{
		"$readPreference": bson.NewDocument(
			bson.C.String("mode", "secondary")),
	})

	expected, err := bson.NewDocument(
		bson.C.SubDocumentFromElements(
			"$query",
			bson.C.Int32("a", 1),
		),
		bson.C.SubDocumentFromElements("$readPreference", bson.C.String("mode", "secondary"))).
		MarshalBSON()
	if err != nil {
		t.Errorf("Unexpected error while marshaling to bytes: %v", err)
	}

	actual, err := req.Query.(*bson.Document).MarshalBSON()
	if err != nil {
		t.Errorf("Unexpected error while marshaling to bytes: %v", err)
	}

	require.Equal(t, expected, actual)
}
