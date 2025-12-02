// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build go1.13

package integration

import (
	"context"
	"errors"
	"io"
	"regexp"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/failpoint"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/integtest"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func containsPattern(patterns []string, str string) bool {
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(str) {
			return true
		}
	}

	return false
}

func TestErrors(t *testing.T) {
	mt := mtest.New(t, noClientOpts)

	mt.RunOpts("errors are wrapped", noClientOpts, func(mt *mtest.T) {
		mt.Run("network error during application operation", func(mt *mtest.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := mt.Client.Ping(ctx, mtest.PrimaryRp)
			assert.True(mt, errors.Is(err, context.Canceled), "expected error %v, got %v", context.Canceled, err)
		})

		authOpts := mtest.NewOptions().Auth(true).Topologies(mtest.ReplicaSet, mtest.Single).MinServerVersion("4.0")
		mt.RunOpts("network error during auth", authOpts, func(mt *mtest.T) {
			mt.SetFailPoint(failpoint.FailPoint{
				ConfigureFailPoint: "failCommand",
				Mode: failpoint.Mode{
					Times: 1,
				},
				Data: failpoint.Data{
					// Set the fail point for saslContinue rather than saslStart because the driver will use speculative
					// auth on 4.4+ so there won't be an explicit saslStart command.
					FailCommands:    []string{"saslContinue"},
					CloseConnection: true,
				},
			})

			clientOpts := options.Client().ApplyURI(mtest.ClusterURI())
			integtest.AddTestServerAPIVersion(clientOpts)
			client, err := mongo.Connect(clientOpts)
			assert.Nil(mt, err, "Connect error: %v", err)
			defer func() { _ = client.Disconnect(context.Background()) }()

			// A connection getting closed should manifest as an io.EOF error.
			err = client.Ping(context.Background(), mtest.PrimaryRp)
			assert.True(mt, errors.Is(err, io.EOF), "expected error %v, got %v", io.EOF, err)
		})
	})

	mt.RunOpts("network timeouts", noClientOpts, func(mt *mtest.T) {
		mt.Run("context timeouts return DeadlineExceeded", func(mt *mtest.T) {
			_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"x", 1}})
			assert.Nil(mt, err, "InsertOne error: %v", err)

			mt.ClearEvents()
			filter := bson.M{
				"$where": "function() { sleep(1000); return false; }",
			}
			timeoutCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			_, err = mt.Coll.Find(timeoutCtx, filter)

			assert.Error(mt, err)

			errPatterns := []string{
				context.DeadlineExceeded.Error(),
				`^\(MaxTimeMSExpired\) Executor error during find command.*:: caused by :: operation exceeded time limit$`,
			}

			assert.True(t, containsPattern(errPatterns, err.Error()),
				"expected possibleErrors=%v to contain %v, but it didn't",
				errPatterns, err.Error())

			evt := mt.GetStartedEvent()
			assert.Equal(mt, "find", evt.CommandName, "expected command 'find', got %q", evt.CommandName)
		})
	})
	mt.Run("ServerError", func(mt *mtest.T) {
		mtOpts := mtest.NewOptions().MinServerVersion("4.0").Topologies(mtest.ReplicaSet)
		mt.RunOpts("Raw response", mtOpts, func(mt *mtest.T) {
			mt.Run("CommandError", func(mt *mtest.T) {
				// Mock a CommandError via failpoint with an arbitrary code.
				mt.SetFailPoint(failpoint.FailPoint{
					ConfigureFailPoint: "failCommand",
					Mode: failpoint.Mode{
						Times: 1,
					},
					Data: failpoint.Data{
						FailCommands: []string{"find"},
						ErrorCode:    123,
					},
				})

				res := mt.Coll.FindOne(context.Background(), bson.D{})
				assert.NotNil(mt, res.Err(), "expected FindOne error, got nil")
				ce, ok := res.Err().(mongo.CommandError)
				assert.True(mt, ok, "expected FindOne error to be CommandError, got %T", res.Err())

				// Assert that raw response exists and contains error code 123.
				assert.NotNil(mt, ce.Raw, "ce.Raw is nil")
				val, err := ce.Raw.LookupErr("code")
				assert.Nil(mt, err, "expected 'code' field in ce.Raw, got %v", ce.Raw)
				code, ok := val.AsInt64OK()
				assert.True(mt, ok, "expected 'code' to be int64, got %v", val)
				assert.Equal(mt, code, int64(123), "expected 'code' 123, got %d", code)
			})
			mt.Run("WriteError", func(mt *mtest.T) {
				// Mock a WriteError by inserting documents with duplicate _id fields.
				_, err := mt.Coll.InsertOne(context.Background(), bson.D{{"_id", 1}})
				assert.Nil(mt, err, "InsertOne error: %v", err)

				_, err = mt.Coll.InsertOne(context.Background(), bson.D{{"_id", 1}})
				assert.NotNil(mt, err, "expected InsertOne error, got nil")
				we, ok := err.(mongo.WriteException)
				assert.True(mt, ok, "expected InsertOne error to be WriteException, got %T", err)

				assert.NotNil(mt, we.WriteErrors, "expected we.WriteErrors, got nil")
				assert.NotNil(mt, we.WriteErrors[0], "expected at least one WriteError")

				// Assert that raw response exists for the WriteError and contains error code 11000.
				raw := we.WriteErrors[0].Raw
				assert.NotNil(mt, raw, "Raw of WriteError is nil")
				val, err := raw.LookupErr("code")
				assert.Nil(mt, err, "expected 'code' field in Raw field, got %v", raw)
				code, ok := val.AsInt64OK()
				assert.True(mt, ok, "expected 'code' to be int64, got %v", val)
				assert.Equal(mt, code, int64(11000), "expected 'code' 11000, got %d", code)
			})
			mt.Run("WriteException", func(mt *mtest.T) {
				// Mock a WriteException via failpoint with an arbitrary WriteConcernError.
				mt.SetFailPoint(failpoint.FailPoint{
					ConfigureFailPoint: "failCommand",
					Mode: failpoint.Mode{
						Times: 1,
					},
					Data: failpoint.Data{
						FailCommands: []string{"delete"},
						WriteConcernError: &failpoint.WriteConcernError{
							Code: 123,
						},
					},
				})

				_, err := mt.Coll.DeleteMany(context.Background(), bson.D{})
				assert.NotNil(mt, err, "expected DeleteMany error, got nil")
				we, ok := err.(mongo.WriteException)
				assert.True(mt, ok, "expected DeleteMany error to be WriteException, got %T", err)

				// Assert that raw response exists and contains error code 123.
				assert.NotNil(mt, we.Raw, "expected RawResponse, got nil")
				val, err := we.Raw.LookupErr("writeConcernError", "code")
				assert.Nil(mt, err, "expected 'code' field in RawResponse, got %v", we.Raw)
				code, ok := val.AsInt64OK()
				assert.True(mt, ok, "expected 'code' to be int64, got %v", val)
				assert.Equal(mt, code, int64(123), "expected 'code' 123, got %d", code)
			})
		})
	})
}
