// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package writeconcern_test

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

// Configure a Client with write concern "majority" that requests
// acknowledgement that a majority of the nodes have committed write operations.
func Example_majority() {
	wc := writeconcern.Majority()

	opts := options.Client().
		ApplyURI("mongodb://localhost:27017").
		SetWriteConcern(wc)

	_, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		panic(err)
	}
}

// Configure a Client with a write concern that requests acknowledgement that
// exactly 2 nodes have committed and journaled write operations.
func Example_w2Journaled() {
	wc := &writeconcern.WriteConcern{
		W:       2,
		Journal: boolPtr(true),
	}

	opts := options.Client().
		ApplyURI("mongodb://localhost:27017").
		SetWriteConcern(wc)

	_, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		panic(err)
	}
}

// boolPtr is a helper function to convert a bool constant into a bool pointer.
//
// If you're using a version of Go that supports generics, you can define a
// generic version of this function that works with any type. For example:
//
//	func ptr[T any](v T) *T {
//		return &v
//	}
func boolPtr(b bool) *bool {
	return &b
}
