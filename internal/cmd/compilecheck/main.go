// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson/v2"
	"go.mongodb.org/mongo-driver/mongo/options/v2"
	"go.mongodb.org/mongo-driver/mongo/v2"
)

func main() {
	_, _ = mongo.Connect(options.Client())
	fmt.Println(bson.D{{Key: "key", Value: "value"}})
}
