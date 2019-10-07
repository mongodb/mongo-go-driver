// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

func setupConvenientTransactions(t *testing.T) *Client {
	cs := testutil.ConnString(t)
	clientOpts := options.Client().ApplyURI(cs.Original).SetReadPreference(readpref.Primary()).
		SetWriteConcern(writeconcern.New(writeconcern.WMajority()))
	client, err := Connect(ctx, clientOpts)
	assert.Nil(t, err, "Connect error: %v", err)

	version, err := getServerVersion(client.Database("admin"))
	assert.Nil(t, err, "getServerVersion error: %v", err)
	topoKind := client.deployment.(*topology.Topology).Kind()
	if compareVersions(t, version, "4.1") < 0 || topoKind == description.Single {
		t.Skip("skipping standalones and versions < 4.1")
	}

	// pin to a single mongos if necessary
	if topoKind != description.Sharded {
		return client
	}
	client, err = Connect(ctx, clientOpts.SetHosts([]string{cs.Hosts[0]}))
	assert.Nil(t, err, "Connect error: %v", err)
	return client
}

func TestConvenientTransactions(t *testing.T) {
	client := setupConvenientTransactions(t)
	db := client.Database("TestConvenientTransactions")
	dbAdmin := client.Database("admin")

	defer func() {
		err := dbAdmin.RunCommand(ctx, bson.D{
			{"killAllSessions", bson.A{}},
		}).Err()
		if err != nil {
			if ce, ok := err.(CommandError); !ok || ce.Code != errorInterrupted {
				t.Fatalf("killAllSessions error: %v", err)
			}
		}

		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	}()

	t.Run("callback raises custom error", func(t *testing.T) {
		coll := db.Collection(t.Name())
		_, err := coll.InsertOne(ctx, bson.D{{"x", 1}})
		assert.Nil(t, err, "InsertOne error: %v", err)

		sess, err := client.StartSession()
		assert.Nil(t, err, "StartSession error: %v", err)
		defer sess.EndSession(context.Background())

		testErr := errors.New("test error")
		_, err = sess.WithTransaction(context.Background(), func(sessCtx SessionContext) (interface{}, error) {
			return nil, testErr
		})
		assert.Equal(t, testErr, err, "expected error %v, got %v", testErr, err)
	})
	t.Run("callback returns value", func(t *testing.T) {
		coll := db.Collection(t.Name())
		_, err := coll.InsertOne(ctx, bson.D{{"x", 1}})
		assert.Nil(t, err, "InsertOne error: %v", err)

		sess, err := client.StartSession()
		assert.Nil(t, err, "StartSession error: %v", err)
		defer sess.EndSession(context.Background())

		res, err := sess.WithTransaction(context.Background(), func(sessCtx SessionContext) (interface{}, error) {
			return false, nil
		})
		assert.Nil(t, err, "WithTransaction error: %v", err)
		resBool, ok := res.(bool)
		assert.True(t, ok, "expected result type %T, got %T", false, res)
		assert.False(t, resBool, "expected result false, got %v", resBool)
	})
	t.Run("retry timeout enforced", func(t *testing.T) {
		withTransactionTimeout = time.Second

		coll := db.Collection(t.Name())
		_, err := coll.InsertOne(ctx, bson.D{{"x", 1}})
		assert.Nil(t, err, "InsertOne error: %v", err)

		t.Run("transient transaction error", func(t *testing.T) {
			sess, err := client.StartSession()
			assert.Nil(t, err, "StartSession error: %v", err)
			defer sess.EndSession(context.Background())

			_, err = sess.WithTransaction(context.Background(), func(sessCtx SessionContext) (interface{}, error) {
				return nil, CommandError{Name: "test Error", Labels: []string{driver.TransientTransactionError}}
			})
			assert.NotNil(t, err, "expected WithTransaction error, got nil")
			cmdErr, ok := err.(CommandError)
			assert.True(t, ok, "expected error type %T, got %T", CommandError{}, err)
			assert.True(t, cmdErr.HasErrorLabel(driver.TransientTransactionError),
				"expected error with label %v, got %v", driver.TransientTransactionError, cmdErr)
		})
		t.Run("unknown transaction commit result", func(t *testing.T) {
			//set failpoint
			failpoint := bson.D{{"configureFailPoint", "failCommand"},
				{"mode", "alwaysOn"},
				{"data", bson.D{
					{"failCommands", bson.A{"commitTransaction"}},
					{"closeConnection", true},
				}},
			}
			err = dbAdmin.RunCommand(ctx, failpoint).Err()
			assert.Nil(t, err, "error setting failpoint: %v", err)
			defer func() {
				err = dbAdmin.RunCommand(ctx, bson.D{
					{"configureFailPoint", "failCommand"},
					{"mode", "off"},
				}).Err()
				assert.Nil(t, err, "error turning off failpoint: %v", err)
			}()

			sess, err := client.StartSession()
			assert.Nil(t, err, "StartSession error: %v", err)
			defer sess.EndSession(context.Background())

			_, err = sess.WithTransaction(context.Background(), func(sessCtx SessionContext) (interface{}, error) {
				_, err := coll.InsertOne(sessCtx, bson.D{{"x", 1}})
				return nil, err
			})
			assert.NotNil(t, err, "expected WithTransaction error, got nil")
			cmdErr, ok := err.(CommandError)
			assert.True(t, ok, "expected error type %T, got %T", CommandError{}, err)
			assert.True(t, cmdErr.HasErrorLabel(driver.UnknownTransactionCommitResult),
				"expected error with label %v, got %v", driver.UnknownTransactionCommitResult, cmdErr)
		})
		t.Run("commit transient transaction error", func(t *testing.T) {
			//set failpoint
			failpoint := bson.D{{"configureFailPoint", "failCommand"},
				{"mode", "alwaysOn"},
				{"data", bson.D{
					{"failCommands", bson.A{"commitTransaction"}},
					{"errorCode", 251},
				}},
			}
			err = dbAdmin.RunCommand(ctx, failpoint).Err()
			assert.Nil(t, err, "error setting failpoint: %v", err)
			defer func() {
				err = dbAdmin.RunCommand(ctx, bson.D{
					{"configureFailPoint", "failCommand"},
					{"mode", "off"},
				}).Err()
				assert.Nil(t, err, "error turning off failpoint: %v", err)
			}()

			sess, err := client.StartSession()
			assert.Nil(t, err, "StartSession error: %v", err)
			defer sess.EndSession(context.Background())

			_, err = sess.WithTransaction(context.Background(), func(sessCtx SessionContext) (interface{}, error) {
				_, err := coll.InsertOne(sessCtx, bson.D{{"x", 1}})
				return nil, err
			})
			assert.NotNil(t, err, "expected WithTransaction error, got nil")
			cmdErr, ok := err.(CommandError)
			assert.True(t, ok, "expected error type %T, got %T", CommandError{}, err)
			assert.True(t, cmdErr.HasErrorLabel(driver.TransientTransactionError),
				"expected error with label %v, got %v", driver.TransientTransactionError, cmdErr)
		})
	})
}
