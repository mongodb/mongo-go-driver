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

// StopStreamProcessor performs a stopStreamProcessor operation against an
// Atlas Stream Processing workspace.
type StopStreamProcessor struct {
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

// NewStopStreamProcessor constructs and returns a new StopStreamProcessor.
func NewStopStreamProcessor(name string) *StopStreamProcessor {
	return &StopStreamProcessor{name: name}
}

// Execute runs this operation and returns an error if it does not execute successfully.
func (ssp *StopStreamProcessor) Execute(ctx context.Context) error {
	if ssp.deployment == nil {
		return errors.New("the StopStreamProcessor operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:      ssp.command,
		Client:         ssp.session,
		Clock:          ssp.clock,
		CommandMonitor: ssp.monitor,
		Crypt:          ssp.crypt,
		Database:       ssp.database,
		Deployment:     ssp.deployment,
		Selector:       ssp.selector,
		ServerAPI:      ssp.serverAPI,
		Name:           driverutil.StopStreamProcessorOp,
		Authenticator:  ssp.authenticator,
	}.Execute(ctx)
}

func (ssp *StopStreamProcessor) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "stopStreamProcessor", ssp.name)
	return dst, nil
}

// Session sets the session for this operation.
func (ssp *StopStreamProcessor) Session(s *session.Client) *StopStreamProcessor {
	if ssp == nil {
		ssp = new(StopStreamProcessor)
	}
	ssp.session = s
	return ssp
}

// ClusterClock sets the cluster clock for this operation.
func (ssp *StopStreamProcessor) ClusterClock(c *session.ClusterClock) *StopStreamProcessor {
	if ssp == nil {
		ssp = new(StopStreamProcessor)
	}
	ssp.clock = c
	return ssp
}

// CommandMonitor sets the monitor to use for APM events.
func (ssp *StopStreamProcessor) CommandMonitor(m *event.CommandMonitor) *StopStreamProcessor {
	if ssp == nil {
		ssp = new(StopStreamProcessor)
	}
	ssp.monitor = m
	return ssp
}

// Crypt sets the Crypt object to use for automatic encryption and decryption.
func (ssp *StopStreamProcessor) Crypt(c driver.Crypt) *StopStreamProcessor {
	if ssp == nil {
		ssp = new(StopStreamProcessor)
	}
	ssp.crypt = c
	return ssp
}

// Database sets the database to run this operation against.
func (ssp *StopStreamProcessor) Database(database string) *StopStreamProcessor {
	if ssp == nil {
		ssp = new(StopStreamProcessor)
	}
	ssp.database = database
	return ssp
}

// Deployment sets the deployment to use for this operation.
func (ssp *StopStreamProcessor) Deployment(d driver.Deployment) *StopStreamProcessor {
	if ssp == nil {
		ssp = new(StopStreamProcessor)
	}
	ssp.deployment = d
	return ssp
}

// ServerSelector sets the selector used to retrieve a server.
func (ssp *StopStreamProcessor) ServerSelector(s description.ServerSelector) *StopStreamProcessor {
	if ssp == nil {
		ssp = new(StopStreamProcessor)
	}
	ssp.selector = s
	return ssp
}

// ServerAPI sets the server API version for this operation.
func (ssp *StopStreamProcessor) ServerAPI(s *driver.ServerAPIOptions) *StopStreamProcessor {
	if ssp == nil {
		ssp = new(StopStreamProcessor)
	}
	ssp.serverAPI = s
	return ssp
}

// Authenticator sets the authenticator to use for this operation.
func (ssp *StopStreamProcessor) Authenticator(a driver.Authenticator) *StopStreamProcessor {
	if ssp == nil {
		ssp = new(StopStreamProcessor)
	}
	ssp.authenticator = a
	return ssp
}
