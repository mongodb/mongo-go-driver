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

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
)

var uriAdmin = os.Getenv("MONGODB_URI")
var uriSingle = os.Getenv("MONGODB_URI_SINGLE")
var uriMulti = os.Getenv("MONGODB_URI_MULTI")
var oidcTokenDir = os.Getenv("OIDC_TOKEN_DIR")
var oidcDomain = os.Getenv("OIDC_DOMAIN")

func explicitUser(user string) string {
	return fmt.Sprintf("%s@%s", user, oidcDomain)
}

func tokenFile(user string) string {
	return path.Join(oidcTokenDir, user)
}

func connectWithMachineCB(uri string, cb driver.OIDCCallback) *mongo.Client {
	opts := options.Client().ApplyURI(uri)

	opts.Auth.OIDCMachineCallback = cb
	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		log.Fatalf("Error connecting client: %v", err)
	}
	return client
}

func main() {
	machine_1_1_callbackIsCalled()
}

func machine_1_1_callbackIsCalled() {
	callbackCount := 0
	countMutex := sync.Mutex{}

	client := connectWithMachineCB(uriSingle, func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		t := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		fmt.Println(tokenFile)
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			log.Fatalf("machine_1_1_callbackIsCalled: failed reading token file: %v", err)
		}
		return &driver.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &t,
			RefreshToken: nil,
		}, nil
	})

	coll := client.Database("test").Collection("test")

	_, err := coll.Find(context.Background(), bson.D{})
	if err != nil {
		log.Fatalf("machine_1_1: failed executing FindOne: %v", err)
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		log.Fatalf("machine_1_1: expected callback count to be 1, got %d", callbackCount)
	}
}

func machine_1_2_callbackIsCalledOnlyOneForMultipleConnections() {
	callbackCount := 0
	countMutex := sync.Mutex{}

	client := connectWithMachineCB(uriSingle, func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		t := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		fmt.Println(tokenFile)
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			log.Fatalf("machine_1_2: failed reading token file: %v", err)
		}
		return &driver.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &t,
			RefreshToken: nil,
		}, nil
	})

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			coll := client.Database("test").Collection("test")
			_, err := coll.Find(context.Background(), bson.D{})
			if err != nil {
				log.Fatalf("machine_1_2: failed executing FindOne: %v", err)
			}
		}()
	}

	wg.Wait()
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		log.Fatalf("machine_1_2: expected callback count to be 1, got %d", callbackCount)
	}
}
