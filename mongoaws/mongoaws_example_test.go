// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongoaws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func ExampleNewCredentialsProvider() {
	awsCredentialProvider := NewCredentialsProvider(aws.NewConfig())
	credential := options.Credential{
		AuthMechanism:          "MONGODB-AWS",
		AWSCredentialsProvider: awsCredentialProvider,
	}
	client, err := mongo.Connect(
		options.Client().SetAuth(credential))
	if err != nil {
		panic(err)
	}
	_ = client
}
