// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package oidcauth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/auth"
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

func connectAdminClient() (*mongo.Client, error) {
	return mongo.Connect(options.Client().ApplyURI(uriAdmin))
}

func connectWithMachineCB(uri string, cb options.OIDCCallback) (*mongo.Client, error) {
	cred := options.Credential{
		AuthMechanism:       "MONGODB-OIDC",
		OIDCMachineCallback: cb,
	}
	optsBuilder := options.Client().ApplyURI(uri).SetAuth(cred)
	return mongo.Connect(optsBuilder)
}

func connectWithHumanCB(uri string, cb options.OIDCCallback) (*mongo.Client, error) {
	cred := options.Credential{
		AuthMechanism:     "MONGODB-OIDC",
		OIDCHumanCallback: cb,
	}
	opts := options.Client().ApplyURI(uri).SetAuth(cred)
	return mongo.Connect(opts)
}

func connectWithHumanCBAndUser(uri string, principal string, cb options.OIDCCallback) (*mongo.Client, error) {
	username := principal
	switch principal {
	case "test_user1", "test_user2":
		username = explicitUser(principal)
	}
	cred := options.Credential{
		AuthMechanism:       "MONGODB-OIDC",
		OIDCMachineCallback: cb,
		Username:            username,
	}
	opts := options.Client().ApplyURI(uri).SetAuth(cred)
	return mongo.Connect(opts)
}

func connectWithHumanCBAndMonitor(uri string, cb options.OIDCCallback, m *event.CommandMonitor) (*mongo.Client, error) {
	cred := options.Credential{
		AuthMechanism:     "MONGODB-OIDC",
		OIDCHumanCallback: cb,
	}
	opts := options.Client().ApplyURI(uri).SetMonitor(m).SetAuth(cred)
	return mongo.Connect(opts)
}

func connectWithMachineCBAndProperties(uri string, cb options.OIDCCallback, props map[string]string) (*mongo.Client, error) {
	cred := options.Credential{
		AuthMechanism:           "MONGODB-OIDC",
		OIDCMachineCallback:     cb,
		AuthMechanismProperties: props,
	}
	optsBuilder := options.Client().ApplyURI(uri).SetAuth(cred)
	return mongo.Connect(optsBuilder)
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestMachine_1_1_CallbackIsCalled(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	client, err := connectWithMachineCB(uriSingle, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("failed reading token file: %v", err)
		}
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestMachine_1_2_CallbackIsCalledOnlyOnce(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	client, err := connectWithMachineCB(uriSingle, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("failed reading token file: %v", err)
		}
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	var wg sync.WaitGroup

	var findFailed error
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			coll := client.Database("test").Collection("test")
			_, err := coll.Find(context.Background(), bson.D{})
			if err != nil {
				findFailed = fmt.Errorf("failed executing Find: %v", err)
			}
		}()
	}

	wg.Wait()
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
	if findFailed != nil {
		t.Fatal(findFailed)
	}
}

func TestMachine_2_1_ValidCallbackInputs(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	client, err := connectWithMachineCB(uriSingle, func(ctx context.Context, args *options.OIDCArgs) (*options.OIDCCredential, error) {
		if args.RefreshToken != nil {
			callbackFailed = fmt.Errorf("expected RefreshToken to be nil, got %v", args.RefreshToken)
		}
		timeout, ok := ctx.Deadline()
		if !ok {
			callbackFailed = fmt.Errorf("expected context to have deadline, got %v", ctx)
		}
		if timeout.Before(time.Now()) {
			callbackFailed = fmt.Errorf("expected timeout to be in the future, got %v", timeout)
		}
		if args.Version < 1 {
			callbackFailed = fmt.Errorf("expected Version to be at least 1, got %d", args.Version)
		}
		if args.IDPInfo != nil {
			callbackFailed = fmt.Errorf("expected IdpID to be nil for Machine flow, got %v", args.IDPInfo)
		}
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			return nil, fmt.Errorf("failed reading token file: %w", err)
		}
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestMachine_2_3_OIDCCallbackReturnMissingData(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	countMutex := sync.Mutex{}

	client, err := connectWithMachineCB(uriSingle, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		return &options.OIDCCredential{
			AccessToken:  "",
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err == nil {
		t.Fatal("should have failed to execute Find, but succeeded")
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
}

func TestMachine_2_4_InvalidClientConfigurationWithCallback(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	_, err := connectWithMachineCBAndProperties(uriSingle, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		expiry := time.Now().Add(time.Hour)
		return &options.OIDCCredential{
			AccessToken:  "",
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	},
		map[string]string{"ENVIRONMENT": "test"},
	)
	if err == nil {
		t.Fatal("succeeded building client when it should fail")
	}
}

func TestMachine_2_5_InvalidUseOfAllowedHosts(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "azure" {
		t.Skip("Skipping: test only runs when OIDC_ENV=azure")
	}

	_, err := connectWithMachineCBAndProperties(uriSingle, func(_ context.Context, _ *options.OIDCArgs) (*options.OIDCCredential, error) {
		expiry := time.Now().Add(time.Hour)
		return &options.OIDCCredential{
			AccessToken:  "",
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	},
		map[string]string{
			"ENVIRONMENT":   "azure",
			"ALLOWED_HOSTS": "",
		},
	)
	if err == nil {
		t.Fatal("succeeded building client when it should fail")
	}
}

func TestMachine_3_1_FailureWithCachedTokens(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	client, err := connectWithMachineCB(uriSingle, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("failed reading token file: %v", err)
		}
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	// Poison the cache with a random token
	clientElem := reflect.ValueOf(client).Elem()
	authenticatorField := clientElem.FieldByName("authenticator")
	authenticatorField = reflect.NewAt(
		authenticatorField.Type(),
		unsafe.Pointer(authenticatorField.UnsafeAddr())).Elem()
	// this is the only usage of the x packages in the test, showing the public interface is
	// correct.
	authenticatorField.Interface().(*auth.OIDCAuthenticator).SetAccessToken("some random happy sunshine string")

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestMachine_3_2_AuthFailuresWithoutCachedTokens(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	client, err := connectWithMachineCB(uriSingle, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		return &options.OIDCCredential{
			AccessToken:  "this is a bad, bad token",
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")
	_, err = coll.Find(context.Background(), bson.D{})
	if err == nil {
		t.Fatal("Find succeeded when it should fail")
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestMachine_3_3_UnexpectedErrorCodeDoesNotClearCache(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	adminClient, err := connectAdminClient()

	defer func() { _ = adminClient.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting admin client: %v", err)
	}

	client, err := connectWithMachineCB(uriSingle, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("failed reading token file: %v", err)
		}
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")

	res := adminClient.Database("admin").RunCommand(context.Background(), bson.D{
		{Key: "configureFailPoint", Value: "failCommand"},
		{Key: "mode", Value: bson.D{
			{Key: "times", Value: 1},
		}},
		{Key: "data", Value: bson.D{
			{Key: "failCommands", Value: bson.A{
				"saslStart",
			}},
			{Key: "errorCode", Value: 20},
		}},
	})

	if res.Err() != nil {
		t.Fatalf("failed setting failpoint: %v", res.Err())
	}

	_, err = coll.Find(context.Background(), bson.D{})
	if err == nil {
		t.Fatal("Find succeeded when it should fail")
	}

	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestMachine_4_1_ReauthenticationSucceeds(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	adminClient, err := connectAdminClient()
	defer func() { _ = adminClient.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting admin client: %v", err)
	}

	client, err := connectWithMachineCB(uriSingle, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("failed reading token file: %v", err)
		}
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")
	res := adminClient.Database("admin").RunCommand(context.Background(), bson.D{
		{Key: "configureFailPoint", Value: "failCommand"},
		{Key: "mode", Value: bson.D{
			{Key: "times", Value: 1},
		}},
		{Key: "data", Value: bson.D{
			{Key: "failCommands", Value: bson.A{
				"find",
			}},
			{Key: "errorCode", Value: 391},
		}},
	})

	if res.Err() != nil {
		t.Fatalf("failed setting failpoint: %v", res.Err())
	}

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 2 {
		t.Fatalf("expected callback count to be 2, got %d", callbackCount)
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestMachine_4_2_ReadCommandsFailIfReauthenticationFails(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	firstCall := true
	countMutex := sync.Mutex{}

	adminClient, err := connectAdminClient()
	defer func() { _ = adminClient.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting admin client: %v", err)
	}

	client, err := connectWithMachineCB(uriSingle, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		if firstCall {
			firstCall = false
			tokenFile := tokenFile("test_user1")
			accessToken, err := os.ReadFile(tokenFile)
			if err != nil {
				callbackFailed = fmt.Errorf("failed reading token file: %v", err)
			}
			return &options.OIDCCredential{
				AccessToken:  string(accessToken),
				ExpiresAt:    &expiry,
				RefreshToken: nil,
			}, nil
		}
		return &options.OIDCCredential{
			AccessToken:  "this is a bad, bad token",
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil

	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")
	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}

	res := adminClient.Database("admin").RunCommand(context.Background(), bson.D{
		{Key: "configureFailPoint", Value: "failCommand"},
		{Key: "mode", Value: bson.D{
			{Key: "times", Value: 1},
		}},
		{Key: "data", Value: bson.D{
			{Key: "failCommands", Value: bson.A{
				"find",
			}},
			{Key: "errorCode", Value: 391},
		}},
	})

	if res.Err() != nil {
		t.Fatalf("failed setting failpoint: %v", res.Err())
	}

	_, err = coll.Find(context.Background(), bson.D{})
	if err == nil {
		t.Fatal("Find succeeded when it should fail")
	}

	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 2 {
		t.Fatalf("expected callback count to be 2, got %d", callbackCount)
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestMachine_4_3_WriteCommandsFailIfReauthenticationFails(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	firstCall := true
	countMutex := sync.Mutex{}

	adminClient, err := connectAdminClient()
	defer func() { _ = adminClient.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting admin client: %v", err)
	}

	client, err := connectWithMachineCB(uriSingle, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		if firstCall {
			firstCall = false
			tokenFile := tokenFile("test_user1")
			accessToken, err := os.ReadFile(tokenFile)
			if err != nil {
				callbackFailed = fmt.Errorf("failed reading token file: %v", err)
			}
			return &options.OIDCCredential{
				AccessToken:  string(accessToken),
				ExpiresAt:    &expiry,
				RefreshToken: nil,
			}, nil
		}
		return &options.OIDCCredential{
			AccessToken:  "this is a bad, bad token",
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")
	_, err = coll.InsertOne(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Insert: %v", err)
	}

	res := adminClient.Database("admin").RunCommand(context.Background(), bson.D{
		{Key: "configureFailPoint", Value: "failCommand"},
		{Key: "mode", Value: bson.D{
			{Key: "times", Value: 1},
		}},
		{Key: "data", Value: bson.D{
			{Key: "failCommands", Value: bson.A{
				"insert",
			}},
			{Key: "errorCode", Value: 391},
		}},
	})

	if res.Err() != nil {
		t.Fatalf("failed setting failpoint: %v", res.Err())
	}

	_, err = coll.InsertOne(context.Background(), bson.D{})
	if err == nil {
		t.Fatal("Insert succeeded when it should fail")
	}

	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 2 {
		t.Fatalf("expected callback count to be 2, got %d", callbackCount)
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestMachine_5_1_AzureWithNoUsername(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "azure" {
		t.Skip("Skipping: test only runs when OIDC_ENV=azure")
	}

	opts := options.Client().ApplyURI(uriSingle)
	if opts == nil {
		t.Fatalf("failed parsing uri: %q", uriSingle)
	}
	client, err := mongo.Connect(opts)
	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}
}

func TestMachine_5_2_AzureWithBadUsername(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "azure" {
		t.Skip("Skipping: test only runs when OIDC_ENV=azure")
	}

	opts := options.Client().ApplyURI(uriSingle)

	if opts == nil {
		t.Fatalf("failed parsing uri: %q", uriSingle)
	}
	if opts.Auth == nil || opts.Auth.AuthMechanism != "MONGODB-OIDC" {
		t.Fatal("expected URI to contain MONGODB-OIDC auth information")
	}

	opts.Auth.Username = "bad"

	client, err := mongo.Connect(opts)
	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err == nil {
		t.Fatal("Find succeeded when it should fail")
	}
}

func TestMachine_6_1_GCPWithNoUsername(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "gcp" {
		t.Skip("Skipping: test only runs when OIDC_ENV=gcp")
	}

	opts := options.Client().ApplyURI(uriSingle)
	client, err := mongo.Connect(opts)
	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}
}

// TestMachine_K8s tests the "k8s" Kubernetes OIDC environment. There is no specified
// prose test for "k8s", so this test simply checks that you can run a "find"
// with the provided conn string.
func TestMachine_K8s(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "k8s" {
		t.Skip("Skipping: test only runs when OIDC_ENV=k8s")
	}

	if !strings.Contains(uriSingle, "ENVIRONMENT:k8s") {
		t.Fatal("expected MONGODB_URI_SINGLE to specify ENVIRONMENT:k8s for Kubernetes test")
	}

	opts := options.Client().ApplyURI(uriSingle)
	client, err := mongo.Connect(opts)
	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}
}

func TestHuman_1_1_SinglePrincipalImplicitUsername(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	client, err := connectWithHumanCB(uriSingle, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("failed reading token file: %v", err)
		}
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestHuman_1_2_SinglePrincipalExplicitUsername(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	client, err := connectWithHumanCBAndUser(uriSingle, "test_user1", func(_ context.Context, _ *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("failed reading token file: %v", err)
		}
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	})
	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestHuman_1_3_MultiplePrincipalUser1(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}
	cb := func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("failed reading token file: %v", err)
		}
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	}
	cred := options.Credential{
		AuthMechanism:     "MONGODB-OIDC",
		OIDCHumanCallback: cb,
		Username:          explicitUser("test_user1"),
	}
	opts := options.Client().ApplyURI(uriMulti).SetAuth(cred)
	client, err := mongo.Connect(opts)
	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestHuman_1_4_MultiplePrincipalUser2(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}
	cb := func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user2")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("failed reading token file: %v", err)
		}
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	}
	cred := options.Credential{
		AuthMechanism:     "MONGODB-OIDC",
		OIDCHumanCallback: cb,
		Username:          explicitUser("test_user2"),
	}
	opts := options.Client().ApplyURI(uriMulti).SetAuth(cred)
	client, err := mongo.Connect(opts)
	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestHuman_1_5_MultiplePrincipalNoUser(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	client, err := connectWithHumanCB(uriMulti, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("failed reading token file: %v", err)
		}
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	})
	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err == nil {
		t.Fatal("Find succeeded when it should fail")
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 0 {
		t.Fatalf("expected callback count to be 0, got %d", callbackCount)
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestHuman_1_6_AllowedHostsBlocked(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	var callbackFailed error

	t.Run("empty_allowed_hosts", func(t *testing.T) {
		cb := func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
			expiry := time.Now().Add(time.Hour)
			tokenFile := tokenFile("test_user1")
			accessToken, err := os.ReadFile(tokenFile)
			if err != nil {
				callbackFailed = fmt.Errorf("failed reading token file: %v", err)
			}
			return &options.OIDCCredential{
				AccessToken:  string(accessToken),
				ExpiresAt:    &expiry,
				RefreshToken: nil,
			}, nil
		}
		cred := options.Credential{
			AuthMechanism:           "MONGODB-OIDC",
			OIDCHumanCallback:       cb,
			AuthMechanismProperties: map[string]string{"ALLOWED_HOSTS": ""},
		}
		opts := options.Client().ApplyURI(uriMulti).SetAuth(cred)
		client, err := mongo.Connect(opts)
		if err != nil {
			t.Fatalf("failed connecting client: %v", err)
		}
		defer func() { _ = client.Disconnect(context.Background()) }()

		coll := client.Database("test").Collection("test")

		_, err = coll.Find(context.Background(), bson.D{})
		if err == nil {
			t.Fatal("Find succeeded when it should fail with empty 'ALLOWED_HOSTS'")
		}
	})

	t.Run("mismatched_allowed_hosts", func(t *testing.T) {
		cb := func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
			expiry := time.Now().Add(time.Hour)
			tokenFile := tokenFile("test_user1")
			accessToken, err := os.ReadFile(tokenFile)
			if err != nil {
				callbackFailed = fmt.Errorf("failed reading token file: %v", err)
			}
			return &options.OIDCCredential{
				AccessToken:  string(accessToken),
				ExpiresAt:    &expiry,
				RefreshToken: nil,
			}, nil
		}
		cred := options.Credential{
			AuthMechanism:           "MONGODB-OIDC",
			OIDCHumanCallback:       cb,
			AuthMechanismProperties: map[string]string{"ALLOWED_HOSTS": "example.com"},
		}
		opts := options.Client().ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC&ignored=example.com").SetAuth(cred)
		client, err := mongo.Connect(opts)
		if err != nil {
			t.Fatalf("failed connecting client: %v", err)
		}
		defer func() { _ = client.Disconnect(context.Background()) }()

		coll := client.Database("test").Collection("test")

		_, err = coll.Find(context.Background(), bson.D{})
		if err == nil {
			t.Fatal("Find succeeded when it should fail with 'ALLOWED_HOSTS' 'example.com'")
		}
	})

	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestHuman_1_7_AllowedHostsInConnectionStringIgnored(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	uri := "mongodb+srv://example.com/?authMechanism=MONGODB-OIDC&authMechanismProperties=ALLOWED_HOSTS:%5B%22example.com%22%5D"
	opts := options.Client().ApplyURI(uri)
	err := opts.Validate()
	if err == nil {
		t.Fatal("succeeded in applying URI which should produce an error")
	}
}

func TestHuman_1_8_MachineIDPHumanCallback(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	if _, ok := os.LookupEnv("OIDC_IS_LOCAL"); !ok {
		t.Skip("Skipping: test only runs when OIDC_IS_LOCAL is set")
	}
	callbackCount := 0

	var callbackFailed error
	countMutex := sync.Mutex{}

	client, err := connectWithHumanCBAndUser(uriSingle, "test_machine", func(_ context.Context, _ *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_machine")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("failed reading token file: %v", err)
		}
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestHuman_2_1_ValidCallbackInputs(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	client, err := connectWithHumanCB(uriSingle, func(_ context.Context, args *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		if args.Version != 1 {
			callbackFailed = fmt.Errorf("expected version to be 1, got %d", args.Version)
		}
		if args.IDPInfo == nil {
			callbackFailed = fmt.Errorf("expected IDPInfo to be non-nil, previous error: (%v)", callbackFailed)
		}
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("failed reading token file: %v, previous error: (%v)", err, callbackFailed)
		}
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestHuman_2_2_CallbackReturnsMissingData(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	countMutex := sync.Mutex{}

	client, err := connectWithHumanCB(uriSingle, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		return &options.OIDCCredential{}, nil
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err == nil {
		t.Fatal("Find succeeded when it should fail")
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
}

func TestHuman_2_3_RefreshTokenIsPassedToCallback(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	adminClient, err := connectAdminClient()
	if err != nil {
		t.Fatalf("failed connecting admin client: %v", err)
	}
	defer func() { _ = adminClient.Disconnect(context.Background()) }()

	client, err := connectWithHumanCB(uriSingle, func(_ context.Context, args *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		if callbackCount == 1 && args.RefreshToken != nil {
			callbackFailed = fmt.Errorf("expected refresh token to be nil first time, got %v, previous error: (%v)", args.RefreshToken, callbackFailed)
		}
		if callbackCount == 2 && args.RefreshToken == nil {
			callbackFailed = fmt.Errorf("expected refresh token to be non-nil second time, got %v, previous error: (%v)", args.RefreshToken, callbackFailed)
		}
		expiry := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("failed reading token file: %v", err)
		}
		rt := "this is fake"
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: &rt,
		}, nil
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	res := adminClient.Database("admin").RunCommand(context.Background(), bson.D{
		{Key: "configureFailPoint", Value: "failCommand"},
		{Key: "mode", Value: bson.D{
			{Key: "times", Value: 1},
		}},
		{Key: "data", Value: bson.D{
			{Key: "failCommands", Value: bson.A{
				"find",
			}},
			{Key: "errorCode", Value: 391},
		}},
	})

	if res.Err() != nil {
		t.Fatal("failed to set failpoint")
	}

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}
	countMutex.Lock()
	defer countMutex.Unlock()
	if callbackCount != 2 {
		t.Fatalf("expected callback count to be 2, got %d", callbackCount)
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestHuman_3_1_UsesSpeculativeAuth(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	adminClient, err := connectAdminClient()
	if err != nil {
		t.Fatalf("failed connecting admin client: %v", err)
	}
	defer func() { _ = adminClient.Disconnect(context.Background()) }()

	client, err := connectWithHumanCB(uriSingle, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		// the callback should not even be called due to spec auth.
		return &options.OIDCCredential{}, nil
	})

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	// We deviate from the Prose test since the failPoint on find with no error code does not seem to
	// work. Rather we put an access token in the cache to force speculative auth.
	tokenFile := tokenFile("test_user1")
	accessToken, err := os.ReadFile(tokenFile)
	if err != nil {
		t.Fatalf("failed reading token file: %v", err)
	}
	clientElem := reflect.ValueOf(client).Elem()
	authenticatorField := clientElem.FieldByName("authenticator")
	authenticatorField = reflect.NewAt(
		authenticatorField.Type(),
		unsafe.Pointer(authenticatorField.UnsafeAddr())).Elem()
	// This is the only usage of the x packages in the test, showing the public interface is
	// correct.
	authenticatorField.Interface().(*auth.OIDCAuthenticator).SetAccessToken(string(accessToken))

	res := adminClient.Database("admin").RunCommand(context.Background(), bson.D{
		{Key: "configureFailPoint", Value: "failCommand"},
		{Key: "mode", Value: bson.D{
			{Key: "times", Value: 1},
		}},
		{Key: "data", Value: bson.D{
			{Key: "failCommands", Value: bson.A{
				"saslStart",
			}},
			{Key: "errorCode", Value: 18},
		}},
	})

	if res.Err() != nil {
		t.Fatal("failed to set failpoint")
	}

	coll := client.Database("test").Collection("test")
	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}
}

func TestHuman_3_2_DoesNotUseSpeculativeAuth(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	var callbackFailed error

	adminClient, err := connectAdminClient()
	if err != nil {
		t.Fatalf("failed connecting admin client: %v", err)
	}
	defer func() { _ = adminClient.Disconnect(context.Background()) }()

	client, err := connectWithHumanCB(uriSingle, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		expiry := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("failed reading token file: %v", err)
		}
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	res := adminClient.Database("admin").RunCommand(context.Background(), bson.D{
		{Key: "configureFailPoint", Value: "failCommand"},
		{Key: "mode", Value: bson.D{
			{Key: "times", Value: 1},
		}},
		{Key: "data", Value: bson.D{
			{Key: "failCommands", Value: bson.A{
				"saslStart",
			}},
			{Key: "errorCode", Value: 18},
		}},
	})

	if res.Err() != nil {
		t.Fatal("failed to set failpoint")
	}

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err == nil {
		t.Fatal("Find succeeded when it should fail")
	}
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestHuman_4_1_ReauthenticationSucceeds(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	adminClient, err := connectAdminClient()
	if err != nil {
		t.Fatalf("failed connecting admin client: %v", err)
	}

	defer func() { _ = adminClient.Disconnect(context.Background()) }()

	clearChannels := func(s chan *event.CommandStartedEvent, succ chan *event.CommandSucceededEvent, f chan *event.CommandFailedEvent) {
		for len(s) > 0 {
			<-s
		}
		for len(succ) > 0 {
			<-succ
		}
		for len(f) > 0 {
			<-f
		}
	}

	started := make(chan *event.CommandStartedEvent, 100)
	succeeded := make(chan *event.CommandSucceededEvent, 100)
	failed := make(chan *event.CommandFailedEvent, 100)

	monitor := event.CommandMonitor{
		Started: func(_ context.Context, e *event.CommandStartedEvent) {
			started <- e
		},
		Succeeded: func(_ context.Context, e *event.CommandSucceededEvent) {
			succeeded <- e
		},
		Failed: func(_ context.Context, e *event.CommandFailedEvent) {
			failed <- e
		},
	}

	client, err := connectWithHumanCBAndMonitor(uriSingle, func(_ context.Context, _ *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("failed reading token file: %v", err)
		}
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	}, &monitor)
	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()
	clearChannels(started, succeeded, failed)

	coll := client.Database("test").Collection("test")
	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatal("Find failed when it should succeed")
	}
	countMutex.Lock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
	countMutex.Unlock()
	clearChannels(started, succeeded, failed)

	res := adminClient.Database("admin").RunCommand(context.Background(), bson.D{
		{Key: "configureFailPoint", Value: "failCommand"},
		{Key: "mode", Value: bson.D{
			{Key: "times", Value: 1},
		}},
		{Key: "data", Value: bson.D{
			{Key: "failCommands", Value: bson.A{
				"find",
			}},
			{Key: "errorCode", Value: 391},
		}},
	})

	if res.Err() != nil {
		t.Fatalf("failed setting failpoint: %v", res.Err())
	}

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatal("Second find failed when it should succeed")
	}
	countMutex.Lock()
	if callbackCount != 2 {
		t.Fatalf("expected callback count to be 2, got %d", callbackCount)
	}
	countMutex.Unlock()

	if len(started) != 2 {
		t.Fatalf("expected 2 finds started, found %d", len(started))
	}
	for len(started) > 0 {
		ste := <-started
		if ste.CommandName != "find" {
			t.Fatalf("found unexpected command started %s", ste.CommandName)
		}
	}
	if len(succeeded) != 1 {
		t.Fatalf("expected 1 finds succeed, found %d", len(succeeded))
	}
	for len(succeeded) > 0 {
		sue := <-succeeded
		if sue.CommandName != "find" {
			t.Fatalf("found unexpected command succeeded %s", sue.CommandName)
		}
	}
	if len(failed) != 1 {
		t.Fatalf("expected 1 finds failed, found %d", len(failed))
	}
	for len(failed) > 0 {
		fe := <-failed
		if fe.CommandName != "find" {
			t.Fatalf("found unexpected command failed %s", fe.CommandName)
		}
	}

	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestHuman_4_2_ReauthenticationSucceedsNoRefreshToken(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	adminClient, err := connectAdminClient()
	if err != nil {
		t.Fatalf("failed connecting admin client: %v", err)
	}
	defer func() { _ = adminClient.Disconnect(context.Background()) }()

	client, err := connectWithHumanCB(uriSingle, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("failed reading token file: %v", err)
		}
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: nil,
		}, nil
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}

	countMutex.Lock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
	countMutex.Unlock()

	res := adminClient.Database("admin").RunCommand(context.Background(), bson.D{
		{Key: "configureFailPoint", Value: "failCommand"},
		{Key: "mode", Value: bson.D{
			{Key: "times", Value: 1},
		}},
		{Key: "data", Value: bson.D{
			{Key: "failCommands", Value: bson.A{
				"find",
			}},
			{Key: "errorCode", Value: 391},
		}},
	})

	if res.Err() != nil {
		t.Fatal("failed to set failpoint")
	}

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}

	countMutex.Lock()
	if callbackCount != 2 {
		t.Fatalf("expected callback count to be 2, got %d", callbackCount)
	}
	countMutex.Unlock()
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestHuman_4_3_ReauthenticationSucceedsAfterRefreshFails(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	adminClient, err := connectAdminClient()
	if err != nil {
		t.Fatalf("failed connecting admin client: %v", err)
	}
	defer func() { _ = adminClient.Disconnect(context.Background()) }()

	client, err := connectWithHumanCB(uriSingle, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		expiry := time.Now().Add(time.Hour)
		tokenFile := tokenFile("test_user1")
		accessToken, err := os.ReadFile(tokenFile)
		if err != nil {
			callbackFailed = fmt.Errorf("failed reading token file: %v", err)
		}
		refreshToken := "bad token"
		return &options.OIDCCredential{
			AccessToken:  string(accessToken),
			ExpiresAt:    &expiry,
			RefreshToken: &refreshToken,
		}, nil
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}

	countMutex.Lock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
	countMutex.Unlock()

	res := adminClient.Database("admin").RunCommand(context.Background(), bson.D{
		{Key: "configureFailPoint", Value: "failCommand"},
		{Key: "mode", Value: bson.D{
			{Key: "times", Value: 1},
		}},
		{Key: "data", Value: bson.D{
			{Key: "failCommands", Value: bson.A{
				"find",
			}},
			{Key: "errorCode", Value: 391},
		}},
	})

	if res.Err() != nil {
		t.Fatal("failed to set failpoint")
	}

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}

	countMutex.Lock()
	if callbackCount != 2 {
		t.Fatalf("expected callback count to be 2, got %d", callbackCount)
	}
	countMutex.Unlock()
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}

func TestHuman_4_4_ReauthenticationFails(t *testing.T) {
	if os.Getenv("OIDC_ENV") != "" {
		t.Skip("Skipping: test only runs when OIDC_ENV is empty")
	}

	callbackCount := 0
	var callbackFailed error
	countMutex := sync.Mutex{}

	adminClient, err := connectAdminClient()
	if err != nil {
		t.Fatalf("failed connecting admin client: %v", err)
	}
	defer func() { _ = adminClient.Disconnect(context.Background()) }()

	client, err := connectWithHumanCB(uriSingle, func(context.Context, *options.OIDCArgs) (*options.OIDCCredential, error) {
		countMutex.Lock()
		defer countMutex.Unlock()
		callbackCount++
		badToken := "bad token"
		expiry := time.Now().Add(time.Hour)
		if callbackCount == 1 {
			tokenFile := tokenFile("test_user1")
			accessToken, err := os.ReadFile(tokenFile)
			if err != nil {
				callbackFailed = fmt.Errorf("failed reading token file: %v", err)
			}
			return &options.OIDCCredential{
				AccessToken:  string(accessToken),
				ExpiresAt:    &expiry,
				RefreshToken: &badToken,
			}, nil
		}
		return &options.OIDCCredential{
			AccessToken:  badToken,
			ExpiresAt:    &expiry,
			RefreshToken: &badToken,
		}, errors.New("failed to refresh token")
	})

	defer func() { _ = client.Disconnect(context.Background()) }()

	if err != nil {
		t.Fatalf("failed connecting client: %v", err)
	}

	coll := client.Database("test").Collection("test")

	_, err = coll.Find(context.Background(), bson.D{})
	if err != nil {
		t.Fatalf("failed executing Find: %v", err)
	}

	countMutex.Lock()
	if callbackCount != 1 {
		t.Fatalf("expected callback count to be 1, got %d", callbackCount)
	}
	countMutex.Unlock()

	res := adminClient.Database("admin").RunCommand(context.Background(), bson.D{
		{Key: "configureFailPoint", Value: "failCommand"},
		{Key: "mode", Value: bson.D{
			{Key: "times", Value: 1},
		}},
		{Key: "data", Value: bson.D{
			{Key: "failCommands", Value: bson.A{
				"find",
			}},
			{Key: "errorCode", Value: 391},
		}},
	})

	if res.Err() != nil {
		t.Fatal("failed to set failpoint")
	}

	_, err = coll.Find(context.Background(), bson.D{})
	if err == nil {
		t.Fatal("Find succeeded when it should fail")
	}

	countMutex.Lock()
	if callbackCount != 3 {
		t.Fatalf("expected callback count to be 3, got %d", callbackCount)
	}
	countMutex.Unlock()
	if callbackFailed != nil {
		t.Fatal(callbackFailed)
	}
}
