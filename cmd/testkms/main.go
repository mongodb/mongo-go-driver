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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var datakeyopts = map[string]primitive.M{
	"aws": bson.M{
		"region": "us-east-1",
		"key":    "arn:aws:kms:us-east-1:579766882180:key/89fcc2c4-08b0-4bd9-9f25-e30687b580d0",
	},
	"azure": bson.M{
		"keyVaultEndpoint": "",
		"keyName":          "",
	},
	"gcp": bson.M{
		"projectId": "devprod-drivers",
		"location":  "global",
		"keyRing":   "key-ring-csfle",
		"keyName":   "key-name-csfle",
	},
}

func main() {
	uri := os.Getenv("MONGODB_URI")
	provider := os.Getenv("PROVIDER")
	// expecterror is an expect error substring. Set to empty string to expect no error.
	expecterror := os.Getenv("EXPECT_ERROR")

	datakeyopt, validKmsProvider := datakeyopts[provider]
	ok := false
	switch {
	case uri == "":
		fmt.Println("ERROR: Please set required MONGODB_URI environment variable.")
	case provider == "":
		fmt.Println("ERROR: Please set required PROVIDER environment variable.")
	case !validKmsProvider:
		fmt.Println("ERROR: Unsupported PROVIDER value.")
	default:
		ok = true
	}
	if provider == "azure" {
		azureKmsKeyName := os.Getenv("AZUREKMS_KEY_NAME")
		azureKmsKeyVaultEndpoint := os.Getenv("AZUREKMS_KEY_VAULT_ENDPOINT")
		if azureKmsKeyName == "" {
			fmt.Println("ERROR: Please set required AZUREKMS_KEY_NAME environment variable.")
			ok = false
		}
		if azureKmsKeyVaultEndpoint == "" {
			fmt.Println("ERROR: Please set required AZUREKMS_KEY_VAULT_ENDPOINT environment variable.")
			ok = false
		}
		datakeyopts["azure"]["keyName"] = azureKmsKeyName
		datakeyopts["azure"]["keyVaultEndpoint"] = azureKmsKeyVaultEndpoint
	}
	if !ok {
		providers := make([]string, 0, len(datakeyopts))
		for p := range datakeyopts {
			providers = append(providers, p)
		}

		fmt.Println("The following environment variables are understood:")
		fmt.Println("- MONGODB_URI as a MongoDB URI. Example: 'mongodb://localhost:27017'")
		fmt.Println("- EXPECT_ERROR as an optional expected error substring.")
		fmt.Println("- PROVIDER as a KMS provider, which supports:", strings.Join(providers, ", "))
		fmt.Println("- AZUREKMS_KEY_NAME as the Azure key name. Required if PROVIDER=azure.")
		fmt.Println("- AZUREKMS_KEY_VAULT_ENDPOINT as the Azure key name. Required if PROVIDER=azure.")
		os.Exit(1)
	}

	cOpts := options.Client().ApplyURI(uri)
	keyVaultClient, err := mongo.Connect(context.Background(), cOpts)
	if err != nil {
		panic(fmt.Sprintf("Connect error: %v", err))
	}
	defer func() { _ = keyVaultClient.Disconnect(context.Background()) }()

	kmsProvidersMap := map[string]map[string]interface{}{
		provider: {},
	}
	ceOpts := options.ClientEncryption().SetKmsProviders(kmsProvidersMap).SetKeyVaultNamespace("keyvault.datakeys")
	ce, err := mongo.NewClientEncryption(keyVaultClient, ceOpts)
	if err != nil {
		panic(fmt.Sprintf("Error in NewClientEncryption: %v", err))
	}
	dkOpts := options.DataKey().SetMasterKey(datakeyopt)
	_, err = ce.CreateDataKey(context.Background(), provider, dkOpts)
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
