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
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/auth"
)

var uriAdmin = os.Getenv("MONGODB_URI")
var uriSingle = os.Getenv("MONGODB_URI_SINGLE")

// var uriMulti = os.Getenv("MONGODB_URI_MULTI")
var oidcTokenDir = os.Getenv("OIDC_TOKEN_DIR")

//var oidcDomain = os.Getenv("OIDC_DOMAIN")

//func explicitUser(user string) string {
//	return fmt.Sprintf("%s@%s", user, oidcDomain)
//}

func tokenFile(user string) string {
	return path.Join(oidcTokenDir, user)
}

func connectAdminClinet() (*mongo.Client, error) {
	return mongo.Connect(context.Background(), options.Client().ApplyURI(uriAdmin))
}

func connectWithMachineCB(uri string, cb driver.OIDCCallback) (*mongo.Client, error) {
	opts := options.Client().ApplyURI(uri)

	opts.Auth.OIDCMachineCallback = cb
	return mongo.Connect(context.Background(), opts)
}

func connectWithMachineCBAndProperties(uri string, cb driver.OIDCCallback, props map[string]string) (*mongo.Client, error) {
	opts := options.Client().ApplyURI(uri)

	opts.Auth.OIDCMachineCallback = cb
	opts.Auth.AuthMechanismProperties = props
	return mongo.Connect(context.Background(), opts)
}

func main() {
	// be quiet linter
	_ = tokenFile("test_user2")

	hasError := false
	aux := func(test_name string, f func() error) {
		fmt.Printf("%s...", test_name)
		err := f()
		if err != nil {
			fmt.Println("Test Error: ", err)
			fmt.Println("...Failed")
			hasError = true
		} else {
			fmt.Println("...Ok")
		}
	}
	aux("machine_1_1_callbackIsCalled", machine11callbackIsCalled)
	aux("machine_1_2_callbackIsCalledOnlyOneForMultipleConnections", machine12callbackIsCalledOnlyOneForMultipleConnections)
	aux("machine_2_1_validCallbackInputs", machine21validCallbackInputs)
	aux("machine_2_3_oidcCallbackReturnMissingData", machine23oidcCallbackReturnMissingData)
	aux("machine_2_4_invalidClientConfigurationWithCallback", machine24invalidClientConfigurationWithCallback)
	aux("machine_3_1_failureWithCachedTokensFetchANewTokenAndRetryAuth", machine31failureWithCachedTokensFetchANewTokenAndRetryAuth)
	aux("machine_3_2_authFailuresWithoutCachedTokensReturnsAnError", machine32authFailuresWithoutCachedTokensReturnsAnError)
	// fail points do not seem to be working, or I'm using them wrongly
	aux("machine_3_3_UnexpectedErrorCodeDoesNotClearTheCache", machine33UnexpectedErrorCodeDoesNotClearTheCache)
	if hasError {
		log.Fatal("One or more tests failed")
	}
}

func machine11callbackIsCalled() error {
	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	client, err := connectWithMachineCB(uriSingle, func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		t := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("machine_1_1: failed reading token file: %v", err)
		}
		return &driver.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &t,
			RefreshToken: nil,
		}, nil
	})

	defer client.Disconnect(context.Background())

	if err != nil {
		return fmt.Errorf("machine_1_1: failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		return fmt.Errorf("machine_1_1: failed executing Find: %v", err)
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		return fmt.Errorf("machine_1_1: expected callback count to be 1, got %d", callbackCount)
	}
	return callbackFailed
}

func machine12callbackIsCalledOnlyOneForMultipleConnections() error {
	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	client, err := connectWithMachineCB(uriSingle, func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		t := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("machine_1_2: failed reading token file: %v", err)
		}
		return &driver.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &t,
			RefreshToken: nil,
		}, nil
	})

	defer client.Disconnect(context.Background())

	if err != nil {
		return fmt.Errorf("machine_1_2: failed connecting client: %v", err)
	}

	var wg sync.WaitGroup

	var findFailed error = nil
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			coll := client.Database("test").Collection("test")
			_, err := coll.Find(context.Background(), bson.D{})
			if err != nil {
				findFailed = fmt.Errorf("machine_1_2: failed executing Find: %v", err)
			}
		}()
	}

	wg.Wait()
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		return fmt.Errorf("machine_1_2: expected callback count to be 1, got %d", callbackCount)
	}
	if callbackFailed != nil {
		return callbackFailed
	}
	return findFailed
}

func machine21validCallbackInputs() error {
	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	client, err := connectWithMachineCB(uriSingle, func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
		if args.RefreshToken != nil {
			callbackFailed = fmt.Errorf("machine_2_1: expected RefreshToken to be nil, got %v", args.RefreshToken)
		}
		if args.Timeout.Before(time.Now()) {
			callbackFailed = fmt.Errorf("machine_2_1: expected timeout to be in the future, got %v", args.Timeout)
		}
		if args.Version < 1 {
			callbackFailed = fmt.Errorf("machine_2_1: expected Version to be at least 1, got %d", args.Version)
		}
		if args.IDPInfo != nil {
			callbackFailed = fmt.Errorf("machine_2_1: expected IdpID to be nil for Machine flow, got %v", args.IDPInfo)
		}
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		t := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			fmt.Printf("machine_2_1: failed reading token file: %v", err)
		}
		return &driver.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &t,
			RefreshToken: nil,
		}, nil
	})

	defer client.Disconnect(context.Background())

	if err != nil {
		return fmt.Errorf("machine_2_1: failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		return fmt.Errorf("machine_2_1: failed executing Find: %v", err)
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		return fmt.Errorf("machine_2_1: expected callback count to be 1, got %d", callbackCount)
	}
	return callbackFailed
}

func machine23oidcCallbackReturnMissingData() error {
	callbackCount := 0
	countMutex := sync.Mutex{}

	client, err := connectWithMachineCB(uriSingle, func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
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

	defer client.Disconnect(context.Background())

	if err != nil {
		return fmt.Errorf("machine_2_3: failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err == nil {
		return fmt.Errorf("machine_2_3: should have failed to executed Find, but succeeded")
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		return fmt.Errorf("machine_2_3: expected callback count to be 1, got %d", callbackCount)
	}
	return nil
}

func machine24invalidClientConfigurationWithCallback() error {
	_, err := connectWithMachineCBAndProperties(uriSingle, func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
		t := time.Now().Add(time.Hour)
		return &driver.OIDCCredential{
			AccessToken:  "",
			ExpiresAt:    &t,
			RefreshToken: nil,
		}, nil
	},
		map[string]string{"ENVIRONMENT": "test"},
	)
	if err == nil {
		return fmt.Errorf("machine_2_4: succeeded building client when it should fail")
	}
	return nil
}

func machine31failureWithCachedTokensFetchANewTokenAndRetryAuth() error {
	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	client, err := connectWithMachineCB(uriSingle, func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		t := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("machine_3_1: failed reading token file: %v", err)
		}
		return &driver.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &t,
			RefreshToken: nil,
		}, nil
	})

	defer client.Disconnect(context.Background())

	if err != nil {
		return fmt.Errorf("machine_3_1: failed connecting client: %v", err)
	}

	// Poison the cache with a random token
	client.GetAuthenticator().(*auth.OIDCAuthenticator).SetAccessToken("some random happy sunshine string")

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		return fmt.Errorf("machine_3_1: failed executing Find: %v", err)
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		return fmt.Errorf("machine_3_1: expected callback count to be 1, got %d", callbackCount)
	}
	return callbackFailed
}

func machine32authFailuresWithoutCachedTokensReturnsAnError() error {
	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	client, err := connectWithMachineCB(uriSingle, func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		t := time.Now().Add(time.Hour)
		return &driver.OIDCCredential{
			AccessToken:  "this is a bad, bad token",
			ExpiresAt:    &t,
			RefreshToken: nil,
		}, nil
	})

	defer client.Disconnect(context.Background())

	if err != nil {
		return fmt.Errorf("machine_3_2: failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")
	_, err = coll.Find(context.Background(), bson.D{})
	if err == nil {
		return fmt.Errorf("machine_3_2: Find ucceeded when it should fail")
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		return fmt.Errorf("machine_3_2: expected callback count to be 1, got %d", callbackCount)
	}
	return callbackFailed
}

func machine33UnexpectedErrorCodeDoesNotClearTheCache() error {
	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	adminClient, err := connectAdminClinet()

	client, err := connectWithMachineCB(uriSingle, func(ctx context.Context, args *driver.OIDCArgs) (*driver.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		t := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("machine_3_3: failed reading token file: %v", err)
		}
		return &driver.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &t,
			RefreshToken: nil,
		}, nil
	})

	defer client.Disconnect(context.Background())

	if err != nil {
		return fmt.Errorf("machine_3_3: failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")

	err = mtest.SetFailPoint(
		mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 1,
			},
			Data: mtest.FailPointData{
				FailCommands: []string{"saslStart"},
				AppName:      "go-oidc",
				ErrorCode:    20,
			},
		},
		adminClient,
	)

	if err != nil {
		return fmt.Errorf("machine_3_3: failed setting failpoint: %v", err)
	}

	_, err = coll.Find(context.Background(), bson.D{})
	if err == nil {
		return fmt.Errorf("machine_3_3: Find succeeded when it should fail")
	}

	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		return fmt.Errorf("machine_3_3: expected callback count to be 1, got %d", callbackCount)
	}

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		return fmt.Errorf("machine_3_3: failed executing Find: %v", err)
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		return fmt.Errorf("machine_3_3: expected callback count to be 1, got %d", callbackCount)
	}
	return callbackFailed
}
