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

// GetMoreSampleStreamProcessor performs a getMoreSampleStreamProcessor
// operation. The response carries a cursorId (0 means exhausted) and a
// nextBatch of documents.
type GetMoreSampleStreamProcessor struct {
	authenticator driver.Authenticator
	session       *session.Client
	clock         *session.ClusterClock
	monitor       *event.CommandMonitor
	crypt         driver.Crypt
	database      string
	deployment    driver.Deployment
	selector      description.ServerSelector
	serverAPI     *driver.ServerAPIOptions

	name      string
	cursorID  int64
	batchSize *int32

	resultCursorID int64
	resultBatch    []bsoncore.Document
}

// NewGetMoreSampleStreamProcessor constructs a new GetMoreSampleStreamProcessor.
func NewGetMoreSampleStreamProcessor(name string, cursorID int64) *GetMoreSampleStreamProcessor {
	return &GetMoreSampleStreamProcessor{name: name, cursorID: cursorID}
}

// ResultCursorID returns the cursor ID from the most recent response. 0 means
// the cursor is exhausted; callers MUST NOT issue another getMore.
func (op *GetMoreSampleStreamProcessor) ResultCursorID() int64 { return op.resultCursorID }

// ResultBatch returns the documents from the most recent response. The
// returned slice references memory owned by the operation; callers that need
// to retain documents past the next Execute should copy them.
func (op *GetMoreSampleStreamProcessor) ResultBatch() []bsoncore.Document { return op.resultBatch }

func (op *GetMoreSampleStreamProcessor) processResponse(_ context.Context, resp bsoncore.Document, _ driver.ResponseInfo) error {
	idVal, err := resp.LookupErr("cursorId")
	if err != nil {
		return fmt.Errorf("getMoreSampleStreamProcessor response is missing cursorId: %w", err)
	}
	id, ok := idVal.AsInt64OK()
	if !ok {
		return errors.New("getMoreSampleStreamProcessor response cursorId is not a number")
	}
	op.resultCursorID = id

	op.resultBatch = op.resultBatch[:0]
	batchVal, err := resp.LookupErr("nextBatch")
	if err != nil {
		// nextBatch may be absent for an exhausted cursor; treat as empty.
		return nil
	}
	arr, ok := batchVal.ArrayOK()
	if !ok {
		return errors.New("getMoreSampleStreamProcessor response nextBatch is not an array")
	}
	vals, err := bsoncore.Document(arr).Elements()
	if err != nil {
		return fmt.Errorf("getMoreSampleStreamProcessor response nextBatch is malformed: %w", err)
	}
	for _, elem := range vals {
		doc, ok := elem.Value().DocumentOK()
		if !ok {
			return errors.New("getMoreSampleStreamProcessor response nextBatch element is not a document")
		}
		op.resultBatch = append(op.resultBatch, doc)
	}
	return nil
}

// Execute runs this operation and returns an error if it does not execute successfully.
func (op *GetMoreSampleStreamProcessor) Execute(ctx context.Context) error {
	if op.deployment == nil {
		return errors.New("the GetMoreSampleStreamProcessor operation must have a Deployment set before Execute can be called")
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
		Name:              driverutil.GetMoreSampleStreamProcessorOp,
		Authenticator:     op.authenticator,
	}.Execute(ctx)
}

func (op *GetMoreSampleStreamProcessor) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "getMoreSampleStreamProcessor", op.name)
	dst = bsoncore.AppendInt64Element(dst, "cursorId", op.cursorID)
	if op.batchSize != nil {
		dst = bsoncore.AppendInt32Element(dst, "batchSize", *op.batchSize)
	}
	return dst, nil
}

// BatchSize sets the desired batch size for this fetch.
func (op *GetMoreSampleStreamProcessor) BatchSize(n int32) *GetMoreSampleStreamProcessor {
	if op == nil {
		op = new(GetMoreSampleStreamProcessor)
	}
	op.batchSize = &n
	return op
}

// Session sets the session for this operation.
func (op *GetMoreSampleStreamProcessor) Session(s *session.Client) *GetMoreSampleStreamProcessor {
	if op == nil {
		op = new(GetMoreSampleStreamProcessor)
	}
	op.session = s
	return op
}

// ClusterClock sets the cluster clock for this operation.
func (op *GetMoreSampleStreamProcessor) ClusterClock(c *session.ClusterClock) *GetMoreSampleStreamProcessor {
	if op == nil {
		op = new(GetMoreSampleStreamProcessor)
	}
	op.clock = c
	return op
}

// CommandMonitor sets the monitor to use for APM events.
func (op *GetMoreSampleStreamProcessor) CommandMonitor(m *event.CommandMonitor) *GetMoreSampleStreamProcessor {
	if op == nil {
		op = new(GetMoreSampleStreamProcessor)
	}
	op.monitor = m
	return op
}

// Crypt sets the Crypt object to use for automatic encryption and decryption.
func (op *GetMoreSampleStreamProcessor) Crypt(c driver.Crypt) *GetMoreSampleStreamProcessor {
	if op == nil {
		op = new(GetMoreSampleStreamProcessor)
	}
	op.crypt = c
	return op
}

// Database sets the database to run this operation against.
func (op *GetMoreSampleStreamProcessor) Database(database string) *GetMoreSampleStreamProcessor {
	if op == nil {
		op = new(GetMoreSampleStreamProcessor)
	}
	op.database = database
	return op
}

// Deployment sets the deployment to use for this operation.
func (op *GetMoreSampleStreamProcessor) Deployment(d driver.Deployment) *GetMoreSampleStreamProcessor {
	if op == nil {
		op = new(GetMoreSampleStreamProcessor)
	}
	op.deployment = d
	return op
}

// ServerSelector sets the selector used to retrieve a server.
func (op *GetMoreSampleStreamProcessor) ServerSelector(s description.ServerSelector) *GetMoreSampleStreamProcessor {
	if op == nil {
		op = new(GetMoreSampleStreamProcessor)
	}
	op.selector = s
	return op
}

// ServerAPI sets the server API version for this operation.
func (op *GetMoreSampleStreamProcessor) ServerAPI(s *driver.ServerAPIOptions) *GetMoreSampleStreamProcessor {
	if op == nil {
		op = new(GetMoreSampleStreamProcessor)
	}
	op.serverAPI = s
	return op
}

// Authenticator sets the authenticator to use for this operation.
func (op *GetMoreSampleStreamProcessor) Authenticator(a driver.Authenticator) *GetMoreSampleStreamProcessor {
	if op == nil {
		op = new(GetMoreSampleStreamProcessor)
	}
	op.authenticator = a
	return op
}
