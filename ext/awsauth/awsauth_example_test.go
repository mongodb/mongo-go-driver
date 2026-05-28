// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package awsauth_test

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"go.mongodb.org/mongo-driver/ext/awsauth"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func ExampleNewCredentialsProvider() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(err)
	}
	awsCredentialProvider, err := awsauth.NewCredentialsProvider(cfg.Credentials)
	if err != nil {
		panic(err)
	}
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
