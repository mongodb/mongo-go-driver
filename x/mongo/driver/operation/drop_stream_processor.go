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
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// DropStreamProcessor performs a dropStreamProcessor operation against an
// Atlas Stream Processing workspace.
type DropStreamProcessor struct {
	authenticator driver.Authenticator
	session       *session.Client
	clock         *session.ClusterClock
	monitor       *event.CommandMonitor
	crypt         driver.Crypt
	database      string
	deployment    driver.Deployment
	selector      description.ServerSelector
	serverAPI     *driver.ServerAPIOptions
	name          string
}

// NewDropStreamProcessor constructs and returns a new DropStreamProcessor.
func NewDropStreamProcessor(name string) *DropStreamProcessor {
	return &DropStreamProcessor{name: name}
}

// Execute runs this operation and returns an error if it does not execute successfully.
func (dsp *DropStreamProcessor) Execute(ctx context.Context) error {
	if dsp.deployment == nil {
		return errors.New("the DropStreamProcessor operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:      dsp.command,
		Client:         dsp.session,
		Clock:          dsp.clock,
		CommandMonitor: dsp.monitor,
		Crypt:          dsp.crypt,
		Database:       dsp.database,
		Deployment:     dsp.deployment,
		Selector:       dsp.selector,
		ServerAPI:      dsp.serverAPI,
		Name:           driverutil.DropStreamProcessorOp,
		Authenticator:  dsp.authenticator,
	}.Execute(ctx)
}

func (dsp *DropStreamProcessor) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "dropStreamProcessor", dsp.name)
	return dst, nil
}

// Session sets the session for this operation.
func (dsp *DropStreamProcessor) Session(s *session.Client) *DropStreamProcessor {
	if dsp == nil {
		dsp = new(DropStreamProcessor)
	}
	dsp.session = s
	return dsp
}

// ClusterClock sets the cluster clock for this operation.
func (dsp *DropStreamProcessor) ClusterClock(c *session.ClusterClock) *DropStreamProcessor {
	if dsp == nil {
		dsp = new(DropStreamProcessor)
	}
	dsp.clock = c
	return dsp
}

// CommandMonitor sets the monitor to use for APM events.
func (dsp *DropStreamProcessor) CommandMonitor(m *event.CommandMonitor) *DropStreamProcessor {
	if dsp == nil {
		dsp = new(DropStreamProcessor)
	}
	dsp.monitor = m
	return dsp
}

// Crypt sets the Crypt object to use for automatic encryption and decryption.
func (dsp *DropStreamProcessor) Crypt(c driver.Crypt) *DropStreamProcessor {
	if dsp == nil {
		dsp = new(DropStreamProcessor)
	}
	dsp.crypt = c
	return dsp
}

// Database sets the database to run this operation against.
func (dsp *DropStreamProcessor) Database(database string) *DropStreamProcessor {
	if dsp == nil {
		dsp = new(DropStreamProcessor)
	}
	dsp.database = database
	return dsp
}

// Deployment sets the deployment to use for this operation.
func (dsp *DropStreamProcessor) Deployment(d driver.Deployment) *DropStreamProcessor {
	if dsp == nil {
		dsp = new(DropStreamProcessor)
	}
	dsp.deployment = d
	return dsp
}

// ServerSelector sets the selector used to retrieve a server.
func (dsp *DropStreamProcessor) ServerSelector(s description.ServerSelector) *DropStreamProcessor {
	if dsp == nil {
		dsp = new(DropStreamProcessor)
	}
	dsp.selector = s
	return dsp
}

// ServerAPI sets the server API version for this operation.
func (dsp *DropStreamProcessor) ServerAPI(s *driver.ServerAPIOptions) *DropStreamProcessor {
	if dsp == nil {
		dsp = new(DropStreamProcessor)
	}
	dsp.serverAPI = s
	return dsp
}

// Authenticator sets the authenticator to use for this operation.
func (dsp *DropStreamProcessor) Authenticator(a driver.Authenticator) *DropStreamProcessor {
	if dsp == nil {
		dsp = new(DropStreamProcessor)
	}
	dsp.authenticator = a
	return dsp
}
