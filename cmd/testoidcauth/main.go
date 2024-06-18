// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
)

var uriAdmin = os.Getenv("MONGODB_URI")
var uriSingle = os.Getenv("MONGODB_URI_SINGLE")
var uriMulti = os.Getenv("MONGODB_URI_MULTI")
var oidcTokenDir = path.Join(os.Getenv("OIDC_TOKEN_DIR"), "tmp", "tokens")
var noUserTokenFile = os.Getenv("OIDC_TOKEN_FILE")
var oidcDomain = os.Getenv("OIDC_DOMAIN")

func explicitUser(user string) string {
	return fmt.Sprintf("%s@%s", user, oidcDomain)
}

func main() {
	machine_1_1_callbackIsCalled()
}

func machine_1_1_callbackIsCalled() {
	callbackCount := 0
	countMutex := sync.Mutex{}

	opts := options.Client().ApplyURI(uriSingle)
	fmt.Println("machine_1_1_callbackIsCalled: uriSingle: ", uriSingle)
	opts.Auth.OIDCMachineCallback = func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		t := time.Now().Add(time.Hour)
		accessToken, err := os.ReadFile(noUserTokenFile)
		if err != nil {
			log.Fatalf("machine_1_1_callbackIsCalled: failed reading token file: %v", err)
		}
		return &driver.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &t,
			RefreshToken: nil,
		}, nil
	}

	client, err := mongo.Connect(
		context.Background(),
		options.Client().ApplyURI(uriSingle))
	if err != nil {
		log.Fatalf("Error connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")

	res := coll.FindOne(context.Background(), nil)
	if res == nil || res.Err() != nil {
		log.Fatalf("machine_1_1_callbackIsCalled: failed executing FindOne: %v", err)
	}
	if callbackCount != 1 {
		log.Fatalf("machine_1_1_callbackIsCalled: expected callback count to be 1, got %d", callbackCount)
	}
}
