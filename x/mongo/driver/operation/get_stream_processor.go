// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package operation

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/internal/driverutil"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// GetStreamProcessor performs a getStreamProcessor operation. The full
// response document is captured raw and returned via Result so the caller can
// decode whatever fields it needs (tolerating unknown server-internal fields).
type GetStreamProcessor struct {
	authenticator  driver.Authenticator
	session        *session.Client
	clock          *session.ClusterClock
	monitor        *event.CommandMonitor
	crypt          driver.Crypt
	database       string
	deployment     driver.Deployment
	selector       description.ServerSelector
	serverAPI      *driver.ServerAPIOptions
	readPreference *readpref.ReadPref
	retry          *driver.RetryMode

	name   string
	result bsoncore.Document
}

// NewGetStreamProcessor constructs a new GetStreamProcessor.
func NewGetStreamProcessor(name string) *GetStreamProcessor {
	return &GetStreamProcessor{name: name}
}

// Result returns the raw server response from the most recent successful
// Execute call.
func (gsp *GetStreamProcessor) Result() bsoncore.Document { return gsp.result }

func (gsp *GetStreamProcessor) processResponse(_ context.Context, resp bsoncore.Document, _ driver.ResponseInfo) error {
	gsp.result = resp
	return nil
}

// Execute runs this operation and returns an error if it does not execute successfully.
func (gsp *GetStreamProcessor) Execute(ctx context.Context) error {
	if gsp.deployment == nil {
		return errors.New("the GetStreamProcessor operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:         gsp.command,
		ProcessResponseFn: gsp.processResponse,
		RetryMode:         gsp.retry,
		Type:              driver.Read,
		Client:            gsp.session,
		Clock:             gsp.clock,
		CommandMonitor:    gsp.monitor,
		Crypt:             gsp.crypt,
		Database:          gsp.database,
		Deployment:        gsp.deployment,
		ReadPreference:    gsp.readPreference,
		Selector:          gsp.selector,
		ServerAPI:         gsp.serverAPI,
		Name:              driverutil.GetStreamProcessorOp,
		Authenticator:     gsp.authenticator,
	}.Execute(ctx)
}

func (gsp *GetStreamProcessor) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "getStreamProcessor", gsp.name)
	return dst, nil
}

// Retry configures the retry mode for this operation.
func (gsp *GetStreamProcessor) Retry(retry driver.RetryMode) *GetStreamProcessor {
	if gsp == nil {
		gsp = new(GetStreamProcessor)
	}
	gsp.retry = &retry
	return gsp
}

// ReadPreference sets the read preference for this operation.
func (gsp *GetStreamProcessor) ReadPreference(rp *readpref.ReadPref) *GetStreamProcessor {
	if gsp == nil {
		gsp = new(GetStreamProcessor)
	}
	gsp.readPreference = rp
	return gsp
}

// Session sets the session for this operation.
func (gsp *GetStreamProcessor) Session(s *session.Client) *GetStreamProcessor {
	if gsp == nil {
		gsp = new(GetStreamProcessor)
	}
	gsp.session = s
	return gsp
}

// ClusterClock sets the cluster clock for this operation.
func (gsp *GetStreamProcessor) ClusterClock(c *session.ClusterClock) *GetStreamProcessor {
	if gsp == nil {
		gsp = new(GetStreamProcessor)
	}
	gsp.clock = c
	return gsp
}

// CommandMonitor sets the monitor to use for APM events.
func (gsp *GetStreamProcessor) CommandMonitor(m *event.CommandMonitor) *GetStreamProcessor {
	if gsp == nil {
		gsp = new(GetStreamProcessor)
	}
	gsp.monitor = m
	return gsp
}

// Crypt sets the Crypt object to use for automatic encryption and decryption.
func (gsp *GetStreamProcessor) Crypt(c driver.Crypt) *GetStreamProcessor {
	if gsp == nil {
		gsp = new(GetStreamProcessor)
	}
	gsp.crypt = c
	return gsp
}

// Database sets the database to run this operation against.
func (gsp *GetStreamProcessor) Database(database string) *GetStreamProcessor {
	if gsp == nil {
		gsp = new(GetStreamProcessor)
	}
	gsp.database = database
	return gsp
}

// Deployment sets the deployment to use for this operation.
func (gsp *GetStreamProcessor) Deployment(d driver.Deployment) *GetStreamProcessor {
	if gsp == nil {
		gsp = new(GetStreamProcessor)
	}
	gsp.deployment = d
	return gsp
}

// ServerSelector sets the selector used to retrieve a server.
func (gsp *GetStreamProcessor) ServerSelector(s description.ServerSelector) *GetStreamProcessor {
	if gsp == nil {
		gsp = new(GetStreamProcessor)
	}
	gsp.selector = s
	return gsp
}

// ServerAPI sets the server API version for this operation.
func (gsp *GetStreamProcessor) ServerAPI(s *driver.ServerAPIOptions) *GetStreamProcessor {
	if gsp == nil {
		gsp = new(GetStreamProcessor)
	}
	gsp.serverAPI = s
	return gsp
}

// Authenticator sets the authenticator to use for this operation.
func (gsp *GetStreamProcessor) Authenticator(a driver.Authenticator) *GetStreamProcessor {
	if gsp == nil {
		gsp = new(GetStreamProcessor)
	}
	gsp.authenticator = a
	return gsp
}
