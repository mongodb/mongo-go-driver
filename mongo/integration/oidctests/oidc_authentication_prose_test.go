// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/auth"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	xauth "go.mongodb.org/mongo-driver/x/mongo/driver/auth"
)

const awsEnv = "AWS_WEB_IDENTITY_TOKEN_FILE"

type result struct {
	expiresInSeconds int
	err              error
}

type callback struct {
	t *testing.T

	path string

	requestResult []result
	refreshResult []result

	requestCbcount int32
	refreshCbcount int32
}

func (cb *callback) OnRequest(ctx context.Context, _ string, _ auth.IDPServerInfo) (auth.IDPServerResp, error) {
	cb.t.Helper()
	cb.validateContextDeadline(ctx, 5*time.Minute)
	i := atomic.AddInt32(&cb.requestCbcount, 1) - 1
	assert.Greater(cb.t, len(cb.requestResult), int(i), "expect more request results")
	return cb.callback(cb.requestResult[i])
}

func (cb *callback) OnRefresh(ctx context.Context, _ string, _ auth.IDPServerInfo, _ auth.IDPRefreshInfo) (auth.IDPServerResp, error) {
	cb.t.Helper()
	cb.validateContextDeadline(ctx, 5*time.Minute)
	i := atomic.AddInt32(&cb.refreshCbcount, 1) - 1
	assert.Greater(cb.t, len(cb.refreshResult), int(i), "expect more refresh results")
	return cb.callback(cb.refreshResult[i])
}

func (cb *callback) callback(res result) (auth.IDPServerResp, error) {
	cb.t.Helper()
	var resp auth.IDPServerResp
	if cb.path == "" {
		path, ok := os.LookupEnv(awsEnv)
		assert.True(cb.t, ok, "expect env: %s", awsEnv)
		cb.path = path
	}
	b, err := ioutil.ReadFile(cb.path)
	assert.Nil(cb.t, err, "ReadFile error: %v", err)
	resp.AccessToken = string(b)
	resp.ExpiresInSeconds = &res.expiresInSeconds
	resp.RefreshToken = &resp.AccessToken
	return resp, res.err
}

func (cb *callback) validateContextDeadline(ctx context.Context, timeout time.Duration) {
	deadline, ok := ctx.Deadline()
	assert.True(cb.t, ok, "Expect a context with deadline")
	timeout = timeout.Round(time.Minute)
	assert.Equal(cb.t, timeout, time.Until(deadline).Round(time.Minute), "Expect timeout in %s", timeout.String())
}

func TestCallbackDrivenAuthProse(t *testing.T) {
	t.Setenv(awsEnv, "/tmp/tokens/test_user1")

	mtOpts := mtest.NewOptions().MinServerVersion("7.0").CreateClient(false)
	mt := mtest.New(t, mtOpts)
	defer mt.Close()

	mt.RunOpts("1.1 Single Principal Implicit Username", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		callback := &callback{
			t:             mt.T,
			requestResult: []result{{expiresInSeconds: 60, err: nil}},
		}
		clientOpts := options.Client().
			ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC")
		clientOpts.Auth.OidcOnRequest = callback.OnRequest
		client, err := mongo.Connect(ctx, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		coll := client.Database("test").Collection("test")
		_, err = coll.Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)
	})
	mt.RunOpts("1.2 Single Principal Explicit Username", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		callback := &callback{
			t:             mt.T,
			requestResult: []result{{expiresInSeconds: 60, err: nil}},
		}
		clientOpts := options.Client().
			ApplyURI("mongodb://test_user1@localhost/?authMechanism=MONGODB-OIDC")
		clientOpts.Auth.OidcOnRequest = callback.OnRequest
		client, err := mongo.Connect(ctx, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		coll := client.Database("test").Collection("test")
		_, err = coll.Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)
	})
	mt.RunOpts("1.3 Multiple Principal User 1", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		callback := &callback{
			t:             mt.T,
			requestResult: []result{{expiresInSeconds: 60, err: nil}},
		}
		clientOpts := options.Client().
			ApplyURI("mongodb://test_user1@localhost:27018/?authMechanism=MONGODB-OIDC&directConnection=true&readPreference=secondaryPreferred")
		clientOpts.Auth.OidcOnRequest = callback.OnRequest
		client, err := mongo.Connect(ctx, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		coll := client.Database("test").Collection("test")
		_, err = coll.Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)
	})
	mt.RunOpts("1.4 Multiple Principal User 2", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		callback := &callback{
			t:             mt.T,
			path:          "/tmp/tokens/test_user2",
			requestResult: []result{{expiresInSeconds: 60, err: nil}},
		}
		clientOpts := options.Client().
			ApplyURI("mongodb://test_user2@localhost:27018/?authMechanism=MONGODB-OIDC&directConnection=true&readPreference=secondaryPreferred")
		clientOpts.Auth.OidcOnRequest = callback.OnRequest
		client, err := mongo.Connect(ctx, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		coll := client.Database("test").Collection("test")
		_, err = coll.Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)
	})
	mt.RunOpts("1.5 Multiple Principal No User", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		callback := &callback{
			t:             mt.T,
			requestResult: []result{{expiresInSeconds: 60, err: nil}},
		}
		clientOpts := options.Client().
			ApplyURI("mongodb://localhost:27018/?authMechanism=MONGODB-OIDC&directConnection=true&readPreference=secondaryPreferred")
		clientOpts.Auth.OidcOnRequest = callback.OnRequest
		client, err := mongo.Connect(ctx, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		coll := client.Database("test").Collection("test")
		_, err = coll.Find(ctx, bson.M{})
		assert.NotNil(mt, err, "expect failure")
	})
	mt.RunOpts("1.6 Allowed Hosts Blocked", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		callback := &callback{
			t:             mt.T,
			requestResult: []result{{expiresInSeconds: 60, err: nil}},
		}
		clientOpts := options.Client().
			ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC&ignored=example.com")
		clientOpts.Auth.OidcOnRequest = callback.OnRequest
		clientOpts.Auth.AuthMechanismProperties["ALLOWED_HOSTS"] = `["example.com"]`
		client, err := mongo.Connect(ctx, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		coll := client.Database("test").Collection("test")
		_, err = coll.Find(ctx, bson.M{})
		assert.ErrorContains(mt, err, "OIDC host is not allowed")
	})
	mt.RunOpts("1.7 Lock Avoids Extra Callback Calls", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		callback := &callback{
			t:             mt.T,
			requestResult: []result{{expiresInSeconds: 60, err: nil}},
			refreshResult: []result{{expiresInSeconds: 301, err: nil}},
		}
		f := func(clientOpts *options.ClientOptions) {
			client, err := mongo.Connect(ctx, clientOpts)
			assert.Nil(mt, err, "Connect error: %v", err)
			coll := client.Database("test").Collection("test")
			_, err = coll.Find(ctx, bson.M{})
			assert.Nil(mt, err, "Find error: %v", err)
		}
		var wg sync.WaitGroup
		t := func() {
			defer wg.Done()
			clientOpts := options.Client().
				ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC")
			clientOpts.Auth.OidcOnRequest = callback.OnRequest
			clientOpts.Auth.OidcOnRefresh = callback.OnRefresh
			f(clientOpts)
			f(clientOpts)
		}
		wg.Add(2)
		go t()
		go t()
		wg.Wait()
		assert.Equal(mt, int32(1), callback.requestCbcount, "request count is expected to be %d, but is %d", 1, callback.requestCbcount)
		assert.Equal(mt, int32(1), callback.refreshCbcount, "refresh count is expected to be %d, but is %d", 1, callback.refreshCbcount)
	})
}

func TestAwsAutomaticAuthProse(t *testing.T) {
	t.Setenv(awsEnv, "/tmp/tokens/test_user1")

	mtOpts := mtest.NewOptions().MinServerVersion("7.0").CreateClient(false)
	mt := mtest.New(t, mtOpts)
	defer mt.Close()

	mt.RunOpts("2.1 Single Principal", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		clientOpts := options.Client().
			ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC&authMechanismProperties=PROVIDER_NAME:aws")
		client, err := mongo.Connect(ctx, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		coll := client.Database("test").Collection("test")
		_, err = coll.Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)
	})
	mt.RunOpts("2.2 Multiple Principal User 1", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		clientOpts := options.Client().
			ApplyURI("mongodb://localhost:27018/?authMechanism=MONGODB-OIDC&authMechanismProperties=PROVIDER_NAME:aws&directConnection=true&readPreference=secondaryPreferred")
		client, err := mongo.Connect(ctx, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		coll := client.Database("test").Collection("test")
		_, err = coll.Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)
	})
	mt.RunOpts("2.3 Multiple Principal User 2", mtOpts, func(mt *mtest.T) {
		mt.T.Setenv(awsEnv, "/tmp/tokens/test_user2")
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		clientOpts := options.Client().
			ApplyURI("mongodb://localhost:27018/?authMechanism=MONGODB-OIDC&authMechanismProperties=PROVIDER_NAME:aws&directConnection=true&readPreference=secondaryPreferred")
		client, err := mongo.Connect(ctx, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		coll := client.Database("test").Collection("test")
		_, err = coll.Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)
	})
	mt.RunOpts("2.4 Allowed Hosts Ignored", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		clientOpts := options.Client().
			ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC&authMechanismProperties=PROVIDER_NAME:aws")
		clientOpts.Auth.AuthMechanismProperties["ALLOWED_HOSTS"] = ""
		client, err := mongo.Connect(ctx, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		coll := client.Database("test").Collection("test")
		_, err = coll.Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)
	})
}

func TestCallbackValidationProse(t *testing.T) {
	t.Setenv(awsEnv, "/tmp/tokens/test_user1")

	mtOpts := mtest.NewOptions().MinServerVersion("7.0").CreateClient(false)
	mt := mtest.New(t, mtOpts)
	defer mt.Close()

	mt.RunOpts("3.1 Valid Callbacks", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		callback := &callback{
			t:             mt.T,
			requestResult: []result{{expiresInSeconds: 60, err: nil}},
			refreshResult: []result{{expiresInSeconds: 60, err: nil}},
		}
		f := func(clientOpts *options.ClientOptions) {
			client, err := mongo.Connect(ctx, clientOpts)
			assert.Nil(mt, err, "Connect error: %v", err)
			coll := client.Database("test").Collection("test")
			_, err = coll.Find(ctx, bson.M{})
			assert.Nil(mt, err, "Find error: %v", err)
		}
		clientOpts := options.Client().
			ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC")
		clientOpts.Auth.OidcOnRequest = callback.OnRequest
		clientOpts.Auth.OidcOnRefresh = callback.OnRefresh
		f(clientOpts)
		assert.Equal(mt, int32(1), callback.requestCbcount, "request count is expected to be %d, but is %d", 1, callback.requestCbcount)
		f(clientOpts)
		assert.Equal(mt, int32(1), callback.refreshCbcount, "refresh count is expected to be %d, but is %d", 1, callback.refreshCbcount)
	})
	mt.RunOpts("3.2 Request Callback Returns Null", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		callback := &callback{
			t:             mt.T,
			requestResult: []result{{expiresInSeconds: 60, err: errors.New("request error")}},
		}
		clientOpts := options.Client().
			ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC")
		clientOpts.Auth.OidcOnRequest = callback.OnRequest
		clientOpts.Auth.OidcOnRefresh = callback.OnRefresh
		client, err := mongo.Connect(ctx, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		coll := client.Database("test").Collection("test")
		_, err = coll.Find(ctx, bson.M{})
		assert.ErrorContains(mt, err, "auth error")
		assert.Equal(mt, int32(1), callback.requestCbcount, "request count is expected to be %d, but is %d", 1, callback.requestCbcount)
		assert.Equal(mt, int32(0), callback.refreshCbcount, "refresh count is expected to be %d, but is %d", 0, callback.refreshCbcount)
	})
	mt.RunOpts("3.3 Refresh Callback Returns Null", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		callback := &callback{
			t:             mt.T,
			requestResult: []result{{expiresInSeconds: 60, err: nil}},
			refreshResult: []result{{expiresInSeconds: 60, err: errors.New("refresh error")}},
		}
		clientOpts := options.Client().
			ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC")
		clientOpts.Auth.OidcOnRequest = callback.OnRequest
		clientOpts.Auth.OidcOnRefresh = callback.OnRefresh
		func(clientOpts *options.ClientOptions) {
			client, err := mongo.Connect(ctx, clientOpts)
			assert.Nil(mt, err, "Connect error: %v", err)
			coll := client.Database("test").Collection("test")
			_, err = coll.Find(ctx, bson.M{})
			assert.Nil(mt, err, "Find error: %v", err)
		}(clientOpts)
		assert.Equal(mt, int32(1), callback.requestCbcount, "request count is expected to be %d, but is %d", 1, callback.requestCbcount)
		func(clientOpts *options.ClientOptions) {
			client, err := mongo.Connect(ctx, clientOpts)
			assert.Nil(mt, err, "Connect error: %v", err)
			coll := client.Database("test").Collection("test")
			_, err = coll.Find(ctx, bson.M{})
			assert.ErrorContains(mt, err, "refresh error")
		}(clientOpts)
		assert.Equal(mt, int32(1), callback.refreshCbcount, "refresh count is expected to be %d, but is %d", 1, callback.refreshCbcount)
	})
	mt.RunOpts("3.4 Request Callback Returns Invalid Data", mtOpts, func(mt *mtest.T) {
		mt.Skip("N/A for Go Driver")
	})
	mt.RunOpts("3.5 Refresh Callback Returns Missing Data", mtOpts, func(mt *mtest.T) {
		mt.Skip("N/A for Go Driver")
	})
	mt.RunOpts("3.6 Refresh Callback Returns Extra Data", mtOpts, func(mt *mtest.T) {
		mt.Skip("N/A for Go Driver")
	})
}

func TestCachedCredentialsProse(t *testing.T) {
	t.Setenv(awsEnv, "/tmp/tokens/test_user1")

	mtOpts := mtest.NewOptions().MinServerVersion("7.0").CreateClient(false)
	mt := mtest.New(t, mtOpts)
	defer mt.Close()

	mt.RunOpts("4.1 Cache with refresh", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		callback := &callback{
			t:             mt.T,
			requestResult: []result{{expiresInSeconds: 60, err: nil}},
			refreshResult: []result{{expiresInSeconds: 60, err: nil}},
		}
		f := func(clientOpts *options.ClientOptions) {
			client, err := mongo.Connect(ctx, clientOpts)
			assert.Nil(mt, err, "Connect error: %v", err)
			coll := client.Database("test").Collection("test")
			_, err = coll.Find(ctx, bson.M{})
			assert.Nil(mt, err, "Find error: %v", err)
		}
		clientOpts := options.Client().
			ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC")
		clientOpts.Auth.OidcOnRequest = callback.OnRequest
		clientOpts.Auth.OidcOnRefresh = callback.OnRefresh
		f(clientOpts)
		assert.Equal(mt, int32(1), callback.requestCbcount, "request count is expected to be %d, but is %d", 1, callback.requestCbcount)
		_, ok := xauth.OidcCache.Load("@localhost:27017")
		assert.True(mt, ok, "credentials are not cached")
		f(clientOpts)
		assert.Equal(mt, int32(1), callback.refreshCbcount, "refresh count is expected to be %d, but is %d", 1, callback.refreshCbcount)
	})
	mt.RunOpts("4.2 Cache with no refresh", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		callback := &callback{
			t: mt.T,
			requestResult: []result{
				{expiresInSeconds: 60, err: nil},
				{expiresInSeconds: 60, err: nil},
			},
		}
		func() {
			clientOpts := options.Client().
				ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC")
			clientOpts.Auth.OidcOnRequest = callback.OnRequest
			clientOpts.Auth.OidcOnRefresh = callback.OnRefresh
			client, err := mongo.Connect(ctx, clientOpts)
			assert.Nil(mt, err, "Connect error: %v", err)
			coll := client.Database("test").Collection("test")
			_, err = coll.Find(ctx, bson.M{})
			assert.Nil(mt, err, "Find error: %v", err)
		}()
		assert.Equal(mt, int32(1), callback.requestCbcount, "request count is expected to be %d, but is %d", 1, callback.requestCbcount)
		func() {
			clientOpts := options.Client().
				ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC")
			clientOpts.Auth.OidcOnRequest = callback.OnRequest
			client, err := mongo.Connect(ctx, clientOpts)
			assert.Nil(mt, err, "Connect error: %v", err)
			coll := client.Database("test").Collection("test")
			_, err = coll.Find(ctx, bson.M{})
			assert.Nil(mt, err, "Find error: %v", err)
		}()
		assert.Equal(mt, int32(2), callback.requestCbcount, "request count is expected to be  %d, but is %d", 2, callback.requestCbcount)
	})
	mt.RunOpts("4.3 Cache key includes callback", mtOpts, func(mt *mtest.T) {
		mt.Skip("N/A for Go Driver")
	})
	mt.RunOpts("4.4 Error clears cache", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		callback := &callback{
			t:             mt.T,
			requestResult: []result{{expiresInSeconds: 300, err: nil}},
			refreshResult: []result{{expiresInSeconds: 300, err: errors.New("refresh error")}},
		}
		clientOpts := options.Client().
			ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC")
		clientOpts.Auth.OidcOnRequest = callback.OnRequest
		clientOpts.Auth.OidcOnRefresh = callback.OnRefresh
		func(clientOpts *options.ClientOptions) {
			client, err := mongo.Connect(ctx, clientOpts)
			assert.Nil(mt, err, "Connect error: %v", err)
			coll := client.Database("test").Collection("test")
			_, err = coll.Find(ctx, bson.M{})
			assert.Nil(mt, err, "Find error: %v", err)
		}(clientOpts)
		func(clientOpts *options.ClientOptions) {
			client, err := mongo.Connect(ctx, clientOpts)
			assert.Nil(mt, err, "Connect error: %v", err)
			coll := client.Database("test").Collection("test")
			_, err = coll.Find(ctx, bson.M{})
			assert.ErrorContains(mt, err, "refresh error")
		}(clientOpts)
		v, ok := xauth.OidcCache.Load("@localhost:27017")
		assert.True(mt, ok, "credentials are not cached")
		entry := v.(*xauth.CacheEntry)
		assert.Equal(mt, &xauth.Auth{}, entry.Auth, "cached credentials are not cleared")
	})
	mt.RunOpts("4.5 AWS Automatic workflow does not use cache", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		clientOpts := options.Client().
			ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC&authMechanismProperties=PROVIDER_NAME:aws")
		client, err := mongo.Connect(ctx, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		coll := client.Database("test").Collection("test")
		_, err = coll.Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)
		_, ok := xauth.OidcCache.Load("@localhost:27017")
		assert.False(mt, ok, "credentials are not expected to be cached")
	})
}

func TestSpeculativeAuthenticationProse(t *testing.T) {
	t.Setenv(awsEnv, "/tmp/tokens/test_user1")

	mtOpts := mtest.NewOptions().MinServerVersion("7.0").CreateClient(false)
	mt := mtest.New(t, mtOpts)
	defer mt.Close()

	xauth.OidcCache = sync.Map{}
	ctx := context.Background()
	callback := &callback{
		t:             mt.T,
		requestResult: []result{{expiresInSeconds: 301, err: nil}},
	}
	clientOpts := options.Client().
		ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC")
	clientOpts.Auth.OidcOnRequest = callback.OnRequest
	clientOpts.Auth.OidcOnRefresh = callback.OnRefresh
	func(clientOpts *options.ClientOptions) {
		client, err := mongo.Connect(ctx, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		coll := client.Database("test").Collection("test")
		_, err = coll.Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)
	}(clientOpts)
	func() {
		client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost"))
		assert.Nil(mt, err, "Connect error: %v", err)
		err = mtest.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 2,
			},
			Data: mtest.FailPointData{
				FailCommands: []string{"saslStart"},
				ErrorCode:    18,
			},
		}, client)
		assert.Nil(mt, err, "SetFailPoint error: %v", err)
	}()
	defer func() {
		client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost"))
		assert.Nil(mt, err, "Connect error: %v", err)
		err = mtest.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode:               "off",
		}, client)
		assert.Nil(t, err, "error turning off failpoint: %v", err)
	}()
	func(clientOpts *options.ClientOptions) {
		client, err := mongo.Connect(ctx, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		coll := client.Database("test").Collection("test")
		_, err = coll.Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)
	}(clientOpts)
}

func TestReauthenticationProse(t *testing.T) {
	t.Setenv(awsEnv, "/tmp/tokens/test_user1")

	mtOpts := mtest.NewOptions().MinServerVersion("7.0").CreateClient(false)
	mt := mtest.New(t, mtOpts)
	defer mt.Close()

	mt.RunOpts("6.1 Succeeds", mtOpts, func(mt *mtest.T) {
		// TODO
	})
	mt.RunOpts("6.2 Retries and Succeeds with Cache", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		callback := &callback{
			t: mt.T,
			requestResult: []result{
				{expiresInSeconds: 360, err: nil},
				{expiresInSeconds: 360, err: nil},
				{expiresInSeconds: 360, err: nil},
				{expiresInSeconds: 360, err: nil},
			},
		}
		clientOpts := options.Client().
			ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC")
		clientOpts.Auth.OidcOnRequest = callback.OnRequest

		client, err := mongo.Connect(ctx, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		coll := client.Database("test").Collection("test")
		_, err = coll.Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)

		err = mtest.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 2,
			},
			Data: mtest.FailPointData{
				FailCommands: []string{"find", "saslStart"},
				ErrorCode:    391,
			},
		}, client)
		assert.Nil(mt, err, "SetFailPoint error: %v", err)
		defer func() {
			err = mtest.SetFailPoint(mtest.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode:               "off",
			}, client)
			assert.Nil(t, err, "error turning off failpoint: %v", err)
		}()

		_, err = coll.Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)
	})
	mt.RunOpts("6.3 Retries and Fails with no Cache", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()
		callback := &callback{
			t:             mt.T,
			requestResult: []result{{expiresInSeconds: 360, err: nil}},
		}
		clientOpts := options.Client().
			ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC")
		clientOpts.Auth.OidcOnRequest = callback.OnRequest

		client, err := mongo.Connect(ctx, clientOpts)
		assert.Nil(mt, err, "Connect error: %v", err)
		coll := client.Database("test").Collection("test")
		_, err = coll.Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)
		xauth.OidcCache = sync.Map{}

		err = mtest.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 2,
			},
			Data: mtest.FailPointData{
				FailCommands: []string{"find", "saslStart"},
				ErrorCode:    391,
			},
		}, client)
		assert.Nil(mt, err, "SetFailPoint error: %v", err)
		defer func() {
			err = mtest.SetFailPoint(mtest.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode:               "off",
			}, client)
			assert.Nil(t, err, "error turning off failpoint: %v", err)
		}()

		_, err = coll.Find(ctx, bson.M{})
		assert.ErrorContains(mt, err, "ReauthenticationRequired")
	})
	mt.RunOpts("6.4 Separate Connections Avoid Extra Callback Calls", mtOpts, func(mt *mtest.T) {
		xauth.OidcCache = sync.Map{}
		ctx := context.Background()

		callback := &callback{
			t:             mt.T,
			requestResult: []result{{expiresInSeconds: 360, err: nil}},
			refreshResult: []result{{expiresInSeconds: 360, err: nil}},
		}
		clientOpts := options.Client().
			ApplyURI("mongodb://localhost/?authMechanism=MONGODB-OIDC")
		clientOpts.Auth.OidcOnRequest = callback.OnRequest
		clientOpts.Auth.OidcOnRefresh = callback.OnRefresh

		client1, err := mongo.Connect(ctx, clientOpts.SetAppName("app1"))
		assert.Nil(mt, err, "Connect error: %v", err)
		client2, err := mongo.Connect(ctx, clientOpts.SetAppName("app2"))
		assert.Nil(mt, err, "Connect error: %v", err)

		_, err = client1.Database("test").Collection("test").Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)
		_, err = client2.Database("test").Collection("test").Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)
		assert.Equal(mt, int32(1), callback.requestCbcount, "request count is expected to be %d, but is %d", 1, callback.requestCbcount)
		assert.Equal(mt, int32(0), callback.refreshCbcount, "refresh count is expected to be %d, but is %d", 0, callback.refreshCbcount)

		err = mtest.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 1,
			},
			Data: mtest.FailPointData{
				FailCommands: []string{"find"},
				ErrorCode:    391,
				AppName:      "app1",
			},
		}, client1)
		assert.Nil(mt, err, "SetFailPoint error: %v", err)
		defer func() {
			err = mtest.SetFailPoint(mtest.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode:               "off",
			}, client1)
			assert.Nil(t, err, "error turning off failpoint: %v", err)
		}()
		_, err = client1.Database("test").Collection("test").Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)
		assert.Equal(mt, int32(1), callback.requestCbcount, "request count is expected to be %d, but is %d", 1, callback.requestCbcount)
		assert.Equal(mt, int32(1), callback.refreshCbcount, "refresh count is expected to be %d, but is %d", 1, callback.refreshCbcount)

		err = mtest.SetFailPoint(mtest.FailPoint{
			ConfigureFailPoint: "failCommand",
			Mode: mtest.FailPointMode{
				Times: 1,
			},
			Data: mtest.FailPointData{
				FailCommands: []string{"find"},
				ErrorCode:    391,
				AppName:      "app2",
			},
		}, client2)
		assert.Nil(mt, err, "SetFailPoint error: %v", err)
		defer func() {
			err = mtest.SetFailPoint(mtest.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode:               "off",
			}, client2)
			assert.Nil(t, err, "error turning off failpoint: %v", err)
		}()
		_, err = client2.Database("test").Collection("test").Find(ctx, bson.M{})
		assert.Nil(mt, err, "Find error: %v", err)
		assert.Equal(mt, int32(1), callback.requestCbcount, "request count is expected to be %d, but is %d", 1, callback.requestCbcount)
		assert.Equal(mt, int32(1), callback.refreshCbcount, "refresh count is expected to be %d, but is %d", 1, callback.refreshCbcount)
	})
}
