// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package operation

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/driverutil"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// StartSampleStreamProcessor performs a startSampleStreamProcessor operation.
// On success the server returns a cursorId; documents are retrieved by a
// subsequent GetMoreSampleStreamProcessor call.
type StartSampleStreamProcessor struct {
	authenticator driver.Authenticator
	session       *session.Client
	clock         *session.ClusterClock
	monitor       *event.CommandMonitor
	crypt         driver.Crypt
	database      string
	deployment    driver.Deployment
	selector      description.ServerSelector
	serverAPI     *driver.ServerAPIOptions

	name     string
	limit    *int32
	cursorID int64
}

// NewStartSampleStreamProcessor constructs a new StartSampleStreamProcessor.
func NewStartSampleStreamProcessor(name string) *StartSampleStreamProcessor {
	return &StartSampleStreamProcessor{name: name}
}

// CursorID returns the cursor ID returned by the server. Zero before Execute
// succeeds.
func (op *StartSampleStreamProcessor) CursorID() int64 { return op.cursorID }

func (op *StartSampleStreamProcessor) processResponse(_ context.Context, resp bsoncore.Document, _ driver.ResponseInfo) error {
	val, err := resp.LookupErr("cursorId")
	if err != nil {
		return fmt.Errorf("startSampleStreamProcessor response is missing cursorId: %w", err)
	}
	id, ok := val.AsInt64OK()
	if !ok {
		return errors.New("startSampleStreamProcessor response cursorId is not a number")
	}
	op.cursorID = id
	return nil
}

// Execute runs this operation and returns an error if it does not execute successfully.
func (op *StartSampleStreamProcessor) Execute(ctx context.Context) error {
	if op.deployment == nil {
		return errors.New("the StartSampleStreamProcessor operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:         op.command,
		ProcessResponseFn: op.processResponse,
		Client:            op.session,
		Clock:             op.clock,
		CommandMonitor:    op.monitor,
		Crypt:             op.crypt,
		Database:          op.database,
		Deployment:        op.deployment,
		Selector:          op.selector,
		ServerAPI:         op.serverAPI,
		Name:              driverutil.StartSampleStreamProcessorOp,
		Authenticator:     op.authenticator,
	}.Execute(ctx)
}

func (op *StartSampleStreamProcessor) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "startSampleStreamProcessor", op.name)
	if op.limit != nil {
		dst = bsoncore.AppendInt32Element(dst, "limit", *op.limit)
	}
	return dst, nil
}

// Limit sets the maximum number of documents to sample.
func (op *StartSampleStreamProcessor) Limit(l int32) *StartSampleStreamProcessor {
	if op == nil {
		op = new(StartSampleStreamProcessor)
	}
	op.limit = &l
	return op
}

// Session sets the session for this operation.
func (op *StartSampleStreamProcessor) Session(s *session.Client) *StartSampleStreamProcessor {
	if op == nil {
		op = new(StartSampleStreamProcessor)
	}
	op.session = s
	return op
}

// ClusterClock sets the cluster clock for this operation.
func (op *StartSampleStreamProcessor) ClusterClock(c *session.ClusterClock) *StartSampleStreamProcessor {
	if op == nil {
		op = new(StartSampleStreamProcessor)
	}
	op.clock = c
	return op
}

// CommandMonitor sets the monitor to use for APM events.
func (op *StartSampleStreamProcessor) CommandMonitor(m *event.CommandMonitor) *StartSampleStreamProcessor {
	if op == nil {
		op = new(StartSampleStreamProcessor)
	}
	op.monitor = m
	return op
}

// Crypt sets the Crypt object to use for automatic encryption and decryption.
func (op *StartSampleStreamProcessor) Crypt(c driver.Crypt) *StartSampleStreamProcessor {
	if op == nil {
		op = new(StartSampleStreamProcessor)
	}
	op.crypt = c
	return op
}

// Database sets the database to run this operation against.
func (op *StartSampleStreamProcessor) Database(database string) *StartSampleStreamProcessor {
	if op == nil {
		op = new(StartSampleStreamProcessor)
	}
	op.database = database
	return op
}

// Deployment sets the deployment to use for this operation.
func (op *StartSampleStreamProcessor) Deployment(d driver.Deployment) *StartSampleStreamProcessor {
	if op == nil {
		op = new(StartSampleStreamProcessor)
	}
	op.deployment = d
	return op
}

// ServerSelector sets the selector used to retrieve a server.
func (op *StartSampleStreamProcessor) ServerSelector(s description.ServerSelector) *StartSampleStreamProcessor {
	if op == nil {
		op = new(StartSampleStreamProcessor)
	}
	op.selector = s
	return op
}

// ServerAPI sets the server API version for this operation.
func (op *StartSampleStreamProcessor) ServerAPI(s *driver.ServerAPIOptions) *StartSampleStreamProcessor {
	if op == nil {
		op = new(StartSampleStreamProcessor)
	}
	op.serverAPI = s
	return op
}

// Authenticator sets the authenticator to use for this operation.
func (op *StartSampleStreamProcessor) Authenticator(a driver.Authenticator) *StartSampleStreamProcessor {
	if op == nil {
		op = new(StartSampleStreamProcessor)
	}
	op.authenticator = a
	return op
}
