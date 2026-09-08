// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"go.mongodb.org/mongo-driver/ext/awsauth"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	_ options.AWSCredentialsProvider = (*awsauth.CredentialsProvider)(nil)
	_ options.AWSCredentials         = awsauth.AWSCredentials(aws.Credentials{})
)

func main() {}
