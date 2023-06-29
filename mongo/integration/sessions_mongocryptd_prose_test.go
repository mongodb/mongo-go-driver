// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

type mongocryptdProcess struct {
	cmd *exec.Cmd
}

// start will start a mongocryptd server in the background on the OS. If a
// process
func (p *mongocryptdProcess) start(ctx context.Context, port int) error {
	args := []string{
		"mongocryptd",
		"--port", strconv.Itoa(port),
	}

	p.cmd = exec.Command(args[0], args[1:]...)
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

// TestSessionsMongocryptdProse tests session prose tests that should use a
// mongocryptd server as the test server (available with server versions 4.2+).
func TestSessionsMongocryptdProse(t *testing.T) {
	t.Parallel()

	const mongocryptdPort = 27022

	ctx := context.Background()

	proc := mongocryptdProcess{}
	procTimeout := time.Duration(100 * time.Millisecond)

	procCtx, procCancel := context.WithTimeout(context.Background(), procTimeout)
	t.Cleanup(procCancel)

	// Start a mongocryptd server.
	err := proc.start(procCtx, mongocryptdPort)
	require.NoError(t, err, "failed to create a mongocryptd process: %v", err)

	t.Cleanup(func() {
		err := proc.close()
		require.NoError(t, err, "failed to close mongocryptd: %v", err)
	})

	cmdMonitor := &event.CommandMonitor{
		Started: func(_ context.Context, evt *event.CommandStartedEvent) {
			// Lookup the session id for the command sent to
			// a mongocryptd server. If the command contains
			// a session ID, then the Go Driver WM
			// construction has incorrectly interpreted that
			// LogicalSessionTimeoutMinutes was returned by
			// the server via a hello command.
			_, err := evt.Command.LookupErr("lsid")
			if !errors.Is(err, bsoncore.ErrElementNotFound) {
				require.NoError(t, err, "expected error to be nil, got %v", err)
			}
		},
	}

	uri := &url.URL{
		Scheme: "mongodb",
		Host:   net.JoinHostPort("localhost", strconv.Itoa(mongocryptdPort)),
	}

	clientOpts := options.Client().ApplyURI(uri.String()).SetMonitor(cmdMonitor)

	client, err := mongo.Connect(ctx, clientOpts)
	require.Nil(t, err, "failed to connect to mongocryptd: %v", err)

	t.Cleanup(func() {
		client.Disconnect(ctx)

		// There is no need to cleanup databases or collections
		// because mongocryptd does not support any kind of
		// data manipulation or typical server functionality. If the
		// result errors were checked for operations performed against
		// this server, they would always return non-nil.
	})

	t.Run("18 implicit session is ignored if connection does not support sessions", func(t *testing.T) {
		coll := client.Database("db").Collection("coll")

		// Send a read command to the server (e.g., findOne), ignoring
		// any errors from the server response
		t.Run("read", func(t *testing.T) { coll.FindOne(ctx, bson.D{{"x", 1}}) })

		// Send a write command to the server (e.g., insertOne),
		// ignoring any errors from the server response
		t.Run("write", func(t *testing.T) { coll.InsertOne(ctx, bson.D{{"x", 1}}) })
	})

	t.Run("19. explicit session raises an error if connection does not support sessions", func(t *testing.T) {
		// Create a new explicit session by calling startSession (this
		// MUST NOT error).
		session, err := client.StartSession()
		require.NoError(t, err, "expected error to be nil, got %v", err)

		defer session.EndSession(ctx)

		sessionCtx := mongo.NewSessionContext(context.TODO(), session)

		if err = session.StartTransaction(); err != nil {
			panic(err)
		}

		coll := client.Database("db").Collection("coll")

		// Attempt to send a read command to the server (e.g., findOne)
		// with the explicit session passed in.
		t.Run("read", func(t *testing.T) {
			// Assert that a client-side error is generated
			// indicating that sessions are not supported
			res := coll.FindOne(sessionCtx, bson.D{{"x", 1}})
			assert.ErrorIs(t, res.Err(), mongo.ErrSessionsNotSupported,
				"expected %v, got %v", mongo.ErrSessionsNotSupported, res.Err())
		})

		// Attempt to send a write command to the server (e.g.,
		// ``insertOne``) with the explicit session passed in.
		t.Run("write", func(t *testing.T) {
			// Assert that a client-side error is generated
			// indicating that sessions are not supported.
			res := coll.FindOne(sessionCtx, bson.D{{"x", 1}})
			assert.ErrorIs(t, res.Err(), mongo.ErrSessionsNotSupported,
				"expected %v, got %v", mongo.ErrSessionsNotSupported, res.Err())
		})
	})

}
