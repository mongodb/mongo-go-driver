// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package msg_test

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	. "github.com/mongodb/mongo-go-driver/mongo/private/msg"
	"github.com/stretchr/testify/require"
)

func TestWrapWithMeta(t *testing.T) {
	req := NewCommand(10, "admin", true, bson.NewDocument(bson.EC.Int32("a", 1))).(*Query)

	err := AddMeta(req, map[string]*bson.Document{
		"$readPreference": bson.NewDocument(
			bson.EC.String("mode", "secondary")),
	})

	expected, err := bson.NewDocument(
		bson.EC.SubDocumentFromElements(
			"$query",
			bson.EC.Int32("a", 1),
		),
		bson.EC.SubDocumentFromElements("$readPreference", bson.EC.String("mode", "secondary"))).
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
