// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	uri := os.Getenv("MONGODB_URI")
	// expecterror is an expect error substring. Set to empty string to expect no error.
	expecterror := os.Getenv("EXPECT_ERROR")

	if uri == "" {
		fmt.Println("ERROR: Please set required MONGODB_URI environment variable.")
		fmt.Println("The following environment variables are understood:")
		fmt.Println("- MONGODB_URI as a MongoDB URI. Example: 'mongodb://localhost:27017'")
		fmt.Println("- EXPECT_ERROR as an optional expected error substring.")
		os.Exit(1)
	}

	cOpts := options.Client().ApplyURI(uri)
	keyVaultClient, err := mongo.Connect(context.Background(), cOpts)
	if err != nil {
		panic(fmt.Sprintf("Connect error: %v", err))
	}
	defer keyVaultClient.Disconnect(context.Background())

	kmsProvidersMap := map[string]map[string]interface{}{
		"aws": {},
	}
	ceOpts := options.ClientEncryption().SetKmsProviders(kmsProvidersMap).SetKeyVaultNamespace("keyvault.datakeys")
	ce, err := mongo.NewClientEncryption(keyVaultClient, ceOpts)
	if err != nil {
		panic(fmt.Sprintf("Error in NewClientEncryption: %v", err))
	}
	defer ce.Close(context.Background())

	dkOpts := options.DataKey().SetMasterKey(bson.M{
		"region": "us-east-1",
		"key":    "arn:aws:kms:us-east-1:579766882180:key/89fcc2c4-08b0-4bd9-9f25-e30687b580d0",
	})
	_, err = ce.CreateDataKey(context.Background(), "aws", dkOpts)
	if expecterror == "" {
		if err != nil {
			panic(fmt.Sprintf("Expected success, but got error in CreateDataKey: %v", err))
		}
	} else {
		if err == nil {
			panic(fmt.Sprintf("Expected error message to contain %q, but got no error", expecterror))
		}
		if !strings.Contains(err.Error(), expecterror) {
			panic(fmt.Sprintf("Expected error message to contain %q, but got %q", expecterror, err.Error()))
		}
	}
}
