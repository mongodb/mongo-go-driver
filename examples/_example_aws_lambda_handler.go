// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package examples

import (
	"context"
	"os"

	runtime "github.com/aws/aws-lambda-go/lambda"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Start AWS Lambda Example 1

var client, err = mongo.Connect(context.TODO(), options.Client().ApplyURI(os.Getenv("MONGODB_URI")))

func HandleRequest(ctx context.Context) error {
	if err != nil {
		return err
	}
	return client.Ping(context.TODO(), nil)
}

// End AWS Lambda Example 1

func main() {
	runtime.Start(HandleRequest)
}
