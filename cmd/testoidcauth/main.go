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
		fmt.Printf("Error connecting client: %v", err)
	}
	return client
}

func connectWithMachineCBAndProperties(uri string, cb driver.OIDCCallback, props map[string]string) *mongo.Client {
	opts := options.Client().ApplyURI(uri)

	opts.Auth.OIDCMachineCallback = cb
	opts.Auth.AuthMechanismProperties = props
	client, err := mongo.Connect(context.Background(), opts)
	if err != nil {
		fmt.Printf("Error connecting client: %v", err)
	}
	return client
}

func main() {
	hasError := false
	aux := func(test_name string, f func() bool) {
		fmt.Printf("%s...", test_name)
		testResult := f()
		if testResult {
			fmt.Println("...Failed")
		} else {
			fmt.Println("...Ok")
		}
		hasError = hasError || testResult
	}
	aux("machine_1_1_callbackIsCalled", machine_1_1_callbackIsCalled)
	aux("machine_1_2_callbackIsCalledOnlyOneForMultipleConnections", machine_1_2_callbackIsCalledOnlyOneForMultipleConnections)
	aux("machine_2_1_validCallbackInputs", machine_2_1_validCallbackInputs)
	aux("machine_2_3_oidcCallbackReturnMissingData", machine_2_3_oidcCallbackReturnMissingData)
	if hasError {
		log.Fatal("One or more tests failed")
	}
}

func machine_1_1_callbackIsCalled() bool {
	callbackCount := 0
	callbackFailed := false
	countMutex := sync.Mutex{}

	client := connectWithMachineCB(uriSingle, func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		t := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			fmt.Printf("machine_1_1: failed reading token file: %v\n", err)
			callbackFailed = true
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
		fmt.Printf("machine_1_1: failed executing Find: %v", err)
		return true
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		fmt.Printf("machine_1_1: expected callback count to be 1, got %d\n", callbackCount)
		return true
	}
	return callbackFailed
}

func machine_1_2_callbackIsCalledOnlyOneForMultipleConnections() bool {
	callbackCount := 0
	callbackFailed := false
	countMutex := sync.Mutex{}

	client := connectWithMachineCB(uriSingle, func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		t := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			fmt.Printf("machine_1_2: failed reading token file: %v\n", err)
			callbackFailed = true
		}
		return &driver.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &t,
			RefreshToken: nil,
		}, nil
	})

	var wg sync.WaitGroup

	findFailed := false
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			coll := client.Database("test").Collection("test")
			_, err := coll.Find(context.Background(), bson.D{})
			if err != nil {
				fmt.Printf("machine_1_2: failed executing Find: %v\n", err)
				findFailed = true
			}
		}()
	}

	wg.Wait()
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		fmt.Printf("machine_1_2: expected callback count to be 1, got %d\n", callbackCount)
		return true
	}
	return callbackFailed || findFailed
}

func machine_2_1_validCallbackInputs() bool {
	callbackCount := 0
	callbackFailed := false
	countMutex := sync.Mutex{}

	client := connectWithMachineCB(uriSingle, func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
		if args.RefreshToken != nil {
			fmt.Printf("machine_2_1: expected RefreshToken to be nil, got %v\n", args.RefreshToken)
			callbackFailed = true
		}
		if args.Timeout.Before(time.Now()) {
			fmt.Printf("machine_2_1: expected timeout to be in the future, got %v\n", args.Timeout)
			callbackFailed = true
		}
		if args.Version < 1 {
			fmt.Printf("machine_2_1: expected Version to be at least 1, got %d\n", args.Version)
			callbackFailed = true
		}
		if args.IDPInfo != nil {
			fmt.Printf("machine_2_1: expected IdpID to be nil for Machine flow, got %v\n", args.IDPInfo)
			callbackFailed = true
		}
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		t := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			fmt.Printf("machine_2_1: failed reading token file: %v\n", err)
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
		fmt.Printf("machine_2_1: failed executing Find: %v", err)
		return true
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		fmt.Printf("machine_2_1: expected callback count to be 1, got %d\n", callbackCount)
		return true
	}
	return callbackFailed
}

func machine_2_3_oidcCallbackReturnMissingData() bool {
	callbackCount := 0
	countMutex := sync.Mutex{}

	client := connectWithMachineCB(uriSingle, func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		t := time.Now().Add(time.Hour)
		return &driver.OIDCCredential{
			AccessToken:  "",
			ExpiresAt:    &t,
			RefreshToken: nil,
		}, nil
	})

	coll := client.Database("test").Collection("test")

	_, err := coll.Find(context.Background(), bson.D{})
	if err == nil {
		fmt.Println("machine_2_3: should have failed to executed Find, but succeeded")
		return true
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		fmt.Printf("machine_2_3: expected callback count to be 1, got %d\n", callbackCount)
		return true
	}
	return true
}

func machine_2_4_invalidClientConfigurationWithCallback() bool {
	callbackCount := 0
	callbackFailed := false
	countMutex := sync.Mutex{}

	client := connectWithMachineCBAndProperties(uriSingle, func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		t := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			fmt.Printf("machine_2_4: failed reading token file: %v\n", err)
			callbackFailed = true
		}
		return &driver.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &t,
			RefreshToken: nil,
		}, nil
	},
		map[string]string{"ENVIRONMENT": "test"},
	)

	coll := client.Database("test").Collection("test")

	_, err := coll.Find(context.Background(), bson.D{})
	if err == nil {
		fmt.Println("machine_2_4: succeeded executing Find when it should fail")
		return true
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		fmt.Printf("machine_2_4: expected callback count to be 1, got %d\n", callbackCount)
		return true
	}
	return callbackFailed
}
