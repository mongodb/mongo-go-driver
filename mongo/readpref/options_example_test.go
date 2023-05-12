// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package readpref_test

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/tag"
)

// Configure a Client with a read preference that selects the nearest replica
// set member that includes tags "region: South" and "datacenter: A".
func ExampleWithTags() {
	rp := readpref.Nearest(
		readpref.WithTags(
			"region", "South",
			"datacenter", "A"))

	opts := options.Client().
		ApplyURI("mongodb://localhost:27017").
		SetReadPreference(rp)

	_, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		panic(err)
	}
}

// Configure a Client with a read preference that selects the nearest replica
// set member matching a set of tags. Try to match members in 3 stages:
//
//  1. Match replica set members that include tags "region: South" and
//     "datacenter: A".
//  2. Match replica set members that includes tag "region: South".
//  3. Match any replica set member.
//
// Stage 3 is used to avoid errors when no members match the previous 2 stages.
func ExampleWithTagSets() {
	tagSetList := tag.NewTagSetsFromMaps([]map[string]string{
		{"region": "South", "datacenter": "A"},
		{"region": "South"},
		{},
	})

	rp := readpref.Nearest(readpref.WithTagSets(tagSetList...))

	opts := options.Client().
		ApplyURI("mongodb://localhost").
		SetReadPreference(rp)

	_, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		panic(err)
	}
}
