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

// GetStreamProcessorStats performs a getStreamProcessorStats operation. The
// full response document is captured raw.
type GetStreamProcessorStats struct {
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

	name    string
	verbose *bool
	result  bsoncore.Document
}

// NewGetStreamProcessorStats constructs a new GetStreamProcessorStats.
func NewGetStreamProcessorStats(name string) *GetStreamProcessorStats {
	return &GetStreamProcessorStats{name: name}
}

// Result returns the raw server response.
func (gsps *GetStreamProcessorStats) Result() bsoncore.Document { return gsps.result }

func (gsps *GetStreamProcessorStats) processResponse(_ context.Context, resp bsoncore.Document, _ driver.ResponseInfo) error {
	gsps.result = resp
	return nil
}

// Execute runs this operation and returns an error if it does not execute successfully.
func (gsps *GetStreamProcessorStats) Execute(ctx context.Context) error {
	if gsps.deployment == nil {
		return errors.New("the GetStreamProcessorStats operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:         gsps.command,
		ProcessResponseFn: gsps.processResponse,
		RetryMode:         gsps.retry,
		Type:              driver.Read,
		Client:            gsps.session,
		Clock:             gsps.clock,
		CommandMonitor:    gsps.monitor,
		Crypt:             gsps.crypt,
		Database:          gsps.database,
		Deployment:        gsps.deployment,
		ReadPreference:    gsps.readPreference,
		Selector:          gsps.selector,
		ServerAPI:         gsps.serverAPI,
		Name:              driverutil.GetStreamProcessorStatsOp,
		Authenticator:     gsps.authenticator,
	}.Execute(ctx)
}

func (gsps *GetStreamProcessorStats) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "getStreamProcessorStats", gsps.name)
	if gsps.verbose != nil {
		var optsIdx int32
		optsIdx, dst = bsoncore.AppendDocumentElementStart(dst, "options")
		dst = bsoncore.AppendBooleanElement(dst, "verbose", *gsps.verbose)
		var err error
		dst, err = bsoncore.AppendDocumentEnd(dst, optsIdx)
		if err != nil {
			return nil, err
		}
	}
	return dst, nil
}

// Verbose sets the options.verbose flag.
func (gsps *GetStreamProcessorStats) Verbose(b bool) *GetStreamProcessorStats {
	if gsps == nil {
		gsps = new(GetStreamProcessorStats)
	}
	gsps.verbose = &b
	return gsps
}

// Retry configures the retry mode for this operation.
func (gsps *GetStreamProcessorStats) Retry(retry driver.RetryMode) *GetStreamProcessorStats {
	if gsps == nil {
		gsps = new(GetStreamProcessorStats)
	}
	gsps.retry = &retry
	return gsps
}

// ReadPreference sets the read preference for this operation.
func (gsps *GetStreamProcessorStats) ReadPreference(rp *readpref.ReadPref) *GetStreamProcessorStats {
	if gsps == nil {
		gsps = new(GetStreamProcessorStats)
	}
	gsps.readPreference = rp
	return gsps
}

// Session sets the session for this operation.
func (gsps *GetStreamProcessorStats) Session(s *session.Client) *GetStreamProcessorStats {
	if gsps == nil {
		gsps = new(GetStreamProcessorStats)
	}
	gsps.session = s
	return gsps
}

// ClusterClock sets the cluster clock for this operation.
func (gsps *GetStreamProcessorStats) ClusterClock(c *session.ClusterClock) *GetStreamProcessorStats {
	if gsps == nil {
		gsps = new(GetStreamProcessorStats)
	}
	gsps.clock = c
	return gsps
}

// CommandMonitor sets the monitor to use for APM events.
func (gsps *GetStreamProcessorStats) CommandMonitor(m *event.CommandMonitor) *GetStreamProcessorStats {
	if gsps == nil {
		gsps = new(GetStreamProcessorStats)
	}
	gsps.monitor = m
	return gsps
}

// Crypt sets the Crypt object to use for automatic encryption and decryption.
func (gsps *GetStreamProcessorStats) Crypt(c driver.Crypt) *GetStreamProcessorStats {
	if gsps == nil {
		gsps = new(GetStreamProcessorStats)
	}
	gsps.crypt = c
	return gsps
}

// Database sets the database to run this operation against.
func (gsps *GetStreamProcessorStats) Database(database string) *GetStreamProcessorStats {
	if gsps == nil {
		gsps = new(GetStreamProcessorStats)
	}
	gsps.database = database
	return gsps
}

// Deployment sets the deployment to use for this operation.
func (gsps *GetStreamProcessorStats) Deployment(d driver.Deployment) *GetStreamProcessorStats {
	if gsps == nil {
		gsps = new(GetStreamProcessorStats)
	}
	gsps.deployment = d
	return gsps
}

// ServerSelector sets the selector used to retrieve a server.
func (gsps *GetStreamProcessorStats) ServerSelector(s description.ServerSelector) *GetStreamProcessorStats {
	if gsps == nil {
		gsps = new(GetStreamProcessorStats)
	}
	gsps.selector = s
	return gsps
}

// ServerAPI sets the server API version for this operation.
func (gsps *GetStreamProcessorStats) ServerAPI(s *driver.ServerAPIOptions) *GetStreamProcessorStats {
	if gsps == nil {
		gsps = new(GetStreamProcessorStats)
	}
	gsps.serverAPI = s
	return gsps
}

// Authenticator sets the authenticator to use for this operation.
func (gsps *GetStreamProcessorStats) Authenticator(a driver.Authenticator) *GetStreamProcessorStats {
	if gsps == nil {
		gsps = new(GetStreamProcessorStats)
	}
	gsps.authenticator = a
	return gsps
}
