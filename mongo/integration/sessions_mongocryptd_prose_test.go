// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"strconv"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

type mongocryptdProcess struct {
	cmd *exec.Cmd
}

// start will start a mongocryptd server in the background on the OS.
func (p *mongocryptdProcess) start(port int) error {
	args := []string{
		"mongocryptd",
		"--port", strconv.Itoa(port),
	}

	p.cmd = exec.Command(args[0], args[1:]...) //nolint:gosec
	p.cmd.Stderr = p.cmd.Stdout

	return p.cmd.Start()
}

// close will kill the underlying process on the command.
func (p *mongocryptdProcess) close() error {
	if p.cmd.Process == nil {
		return nil
	}

	if err := p.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to terminate mongocryptd process: %w", err)
	}

	// Release instead of wait to avoid blocking in CI.
	err := p.cmd.Process.Release()
	if err != nil {
		return fmt.Errorf("failed to release mongocryptd: %w", err)
	}

	return nil
}

// newTestSessionMongocryptdProseClient will Create the client using the mongo
// API rather than mtest. mtest will attempt to create a collection as
// a database operation, which will not work on a mongocryptd server. A
// mongocryptd server does not support operations on a database.
func newTestSessionMongocryptdProseClient(mt *mtest.T) *mongo.Client {
	const mongocryptdPort = 27022

	// Monitor the lsid value on commands. If an operation run in any
	// subtests contains an lsid, then the Go Driver wire message
	// construction has incorrectly interpreted that
	// LogicalSessionTimeoutMinutes was returned by the server on handshake.
	cmdMonitor := &event.CommandMonitor{
		Started: func(_ context.Context, evt *event.CommandStartedEvent) {
			_, err := evt.Command.LookupErr("lsid")
			assert.ErrorIs(mt, err, bsoncore.ErrElementNotFound)
		},
	}

	uri := &url.URL{
		Scheme: "mongodb",
		Host:   net.JoinHostPort("localhost", strconv.Itoa(mongocryptdPort)),
	}

	proc := mongocryptdProcess{}

	// Start a mongocryptd server.
	err := proc.start(mongocryptdPort)
	require.NoError(mt, err, "failed to create a mongocryptd process: %v", err)

	mt.Cleanup(func() {
		err := proc.close()
		require.NoError(mt, err, "failed to close mongocryptd: %v", err)
	})

	clientOpts := options.
		Client().
		ApplyURI(uri.String()).
		SetMonitor(cmdMonitor)

	ctx := context.Background()

	client, err := mongo.Connect(ctx, clientOpts)
	require.NoError(mt, err, "could not connect to mongocryptd: %v", err)

	return client

}

func TestSessionsMongocryptdProse(t *testing.T) {
	mtOpts := mtest.NewOptions().
		MinServerVersion("4.2").
		Topologies(mtest.ReplicaSet, mtest.Sharded).
		CreateCollection(false).
		CreateClient(false)

	// Create a new instance of mtest (MongoDB testing framework) for this
	// test and configure it to control server versions.
	mt := mtest.New(t, mtOpts)

	proseTest18 := "18. implicit session is ignored if connection does not support sessions"
	mt.RunOpts(proseTest18, mtOpts, func(mt *mtest.T) {
		client := newTestSessionMongocryptdProseClient(mt)

		mt.Cleanup(func() {
			err := client.Disconnect(context.Background())
			require.NoError(mt, err, "mongocryptd client could not disconnect: %v", err)
		})

		coll := client.Database("db").Collection("coll")

		// Send a read command to the server (e.g., findOne), ignoring
		// any errors from the server response
		mt.RunOpts("read", mtOpts, func(_ *mtest.T) {
			_ = coll.FindOne(context.Background(), bson.D{{"x", 1}})
		})

		// Send a write command to the server (e.g., insertOne),
		// ignoring any errors from the server response
		mt.RunOpts("write", mtOpts, func(_ *mtest.T) {
			_, _ = coll.InsertOne(context.Background(), bson.D{{"x", 1}})
		})
	})

	proseTest19 := "19. explicit session raises an error if connection does not support sessions"
	mt.RunOpts(proseTest19, mtOpts, func(mt *mtest.T) {
		client := newTestSessionMongocryptdProseClient(mt)

		mt.Cleanup(func() {
			err := client.Disconnect(context.Background())
			require.NoError(mt, err, "mongocryptd client could not disconnect: %v", err)
		})

		// Create a new explicit session by calling startSession (this
		// MUST NOT error).
		session, err := client.StartSession()
		require.NoError(mt, err, "expected error to be nil, got %v", err)

		defer session.EndSession(context.Background())

		sessionCtx := mongo.NewSessionContext(context.TODO(), session)

		err = session.StartTransaction()
		require.NoError(mt, err, "expected error to be nil, got %v", err)

		coll := client.Database("db").Collection("coll")

		// Attempt to send a read command to the server (e.g., findOne)
		// with the explicit session passed in.
		mt.RunOpts("read", mtOpts, func(mt *mtest.T) {
			// Assert that a client-side error is generated
			// indicating that sessions are not supported
			res := coll.FindOne(sessionCtx, bson.D{{"x", 1}})
			assert.EqualError(mt, res.Err(), "current topology does not support sessions")
		})

		// Attempt to send a write command to the server (e.g.,
		// ``insertOne``) with the explicit session passed in.
		mt.RunOpts("write", mtOpts, func(mt *mtest.T) {
			// Assert that a client-side error is generated
			// indicating that sessions are not supported.
			res := coll.FindOne(sessionCtx, bson.D{{"x", 1}})
			assert.EqualError(mt, res.Err(), "current topology does not support sessions")
		})
	})
}
