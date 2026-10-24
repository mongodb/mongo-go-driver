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

// StartStreamProcessor performs a startStreamProcessor operation against an
// Atlas Stream Processing workspace.
//
// Wire shape:
//
//	{
//	  startStreamProcessor: <name>,
//	  workers: <Int32>,
//	  options: {
//	    clearCheckpoints: <bool>,
//	    startAtOperationTime: <Timestamp>,
//	    tier: <string>,
//	    enableAutoScaling: <bool>
//	  },
//	  failover: {
//	    region: <string>,
//	    mode: <string>,
//	    dryRun: <bool>
//	  }
//	}
//
// The startAfter option is reserved by the spec for future use and is
// intentionally not serialized to the wire.
type StartStreamProcessor struct {
	authenticator driver.Authenticator
	session       *session.Client
	clock         *session.ClusterClock
	monitor       *event.CommandMonitor
	crypt         driver.Crypt
	database      string
	deployment    driver.Deployment
	selector      description.ServerSelector
	serverAPI     *driver.ServerAPIOptions

	name              string
	workers           *int32
	clearCheckpoints  *bool
	startAtOperation  *startAtOperationTime
	tier              *string
	enableAutoScaling *bool

	failoverRegion *string
	failoverMode   *string
	failoverDryRun *bool
}

type startAtOperationTime struct {
	t uint32
	i uint32
}

// NewStartStreamProcessor constructs a new StartStreamProcessor.
func NewStartStreamProcessor(name string) *StartStreamProcessor {
	return &StartStreamProcessor{name: name}
}

// Execute runs this operation and returns an error if it does not execute successfully.
func (ssp *StartStreamProcessor) Execute(ctx context.Context) error {
	if ssp.deployment == nil {
		return errors.New("the StartStreamProcessor operation must have a Deployment set before Execute can be called")
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
		Name:           driverutil.StartStreamProcessorOp,
		Authenticator:  ssp.authenticator,
	}.Execute(ctx)
}

func (ssp *StartStreamProcessor) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "startStreamProcessor", ssp.name)
	if ssp.workers != nil {
		dst = bsoncore.AppendInt32Element(dst, "workers", *ssp.workers)
	}
	if ssp.clearCheckpoints != nil || ssp.startAtOperation != nil || ssp.tier != nil || ssp.enableAutoScaling != nil {
		var optsIdx int32
		optsIdx, dst = bsoncore.AppendDocumentElementStart(dst, "options")
		if ssp.clearCheckpoints != nil {
			dst = bsoncore.AppendBooleanElement(dst, "clearCheckpoints", *ssp.clearCheckpoints)
		}
		if ssp.startAtOperation != nil {
			dst = bsoncore.AppendTimestampElement(dst, "startAtOperationTime", ssp.startAtOperation.t, ssp.startAtOperation.i)
		}
		if ssp.tier != nil {
			dst = bsoncore.AppendStringElement(dst, "tier", *ssp.tier)
		}
		if ssp.enableAutoScaling != nil {
			dst = bsoncore.AppendBooleanElement(dst, "enableAutoScaling", *ssp.enableAutoScaling)
		}
		var err error
		dst, err = bsoncore.AppendDocumentEnd(dst, optsIdx)
		if err != nil {
			return nil, err
		}
	}
	if ssp.failoverRegion != nil || ssp.failoverMode != nil || ssp.failoverDryRun != nil {
		if ssp.failoverRegion == nil {
			return nil, errors.New("failover requires a target region")
		}
		var failIdx int32
		failIdx, dst = bsoncore.AppendDocumentElementStart(dst, "failover")
		dst = bsoncore.AppendStringElement(dst, "region", *ssp.failoverRegion)
		if ssp.failoverMode != nil {
			dst = bsoncore.AppendStringElement(dst, "mode", *ssp.failoverMode)
		}
		if ssp.failoverDryRun != nil {
			dst = bsoncore.AppendBooleanElement(dst, "dryRun", *ssp.failoverDryRun)
		}
		var err error
		dst, err = bsoncore.AppendDocumentEnd(dst, failIdx)
		if err != nil {
			return nil, err
		}
	}
	return dst, nil
}

// Workers sets the workers field on the command.
func (ssp *StartStreamProcessor) Workers(w int32) *StartStreamProcessor {
	if ssp == nil {
		ssp = new(StartStreamProcessor)
	}
	ssp.workers = &w
	return ssp
}

// ClearCheckpoints sets the options.clearCheckpoints flag.
func (ssp *StartStreamProcessor) ClearCheckpoints(b bool) *StartStreamProcessor {
	if ssp == nil {
		ssp = new(StartStreamProcessor)
	}
	ssp.clearCheckpoints = &b
	return ssp
}

// StartAtOperationTime sets the options.startAtOperationTime BSON timestamp.
func (ssp *StartStreamProcessor) StartAtOperationTime(t, i uint32) *StartStreamProcessor {
	if ssp == nil {
		ssp = new(StartStreamProcessor)
	}
	ssp.startAtOperation = &startAtOperationTime{t: t, i: i}
	return ssp
}

// Tier sets the options.tier value.
func (ssp *StartStreamProcessor) Tier(tier string) *StartStreamProcessor {
	if ssp == nil {
		ssp = new(StartStreamProcessor)
	}
	ssp.tier = &tier
	return ssp
}

// EnableAutoScaling sets the options.enableAutoScaling flag.
func (ssp *StartStreamProcessor) EnableAutoScaling(b bool) *StartStreamProcessor {
	if ssp == nil {
		ssp = new(StartStreamProcessor)
	}
	ssp.enableAutoScaling = &b
	return ssp
}

// FailoverRegion sets the failover.region value. Required when any failover
// field is set.
func (ssp *StartStreamProcessor) FailoverRegion(region string) *StartStreamProcessor {
	if ssp == nil {
		ssp = new(StartStreamProcessor)
	}
	ssp.failoverRegion = &region
	return ssp
}

// FailoverMode sets the failover.mode value (e.g. "GRACEFUL", "FORCED").
func (ssp *StartStreamProcessor) FailoverMode(mode string) *StartStreamProcessor {
	if ssp == nil {
		ssp = new(StartStreamProcessor)
	}
	ssp.failoverMode = &mode
	return ssp
}

// FailoverDryRun sets the failover.dryRun flag.
func (ssp *StartStreamProcessor) FailoverDryRun(b bool) *StartStreamProcessor {
	if ssp == nil {
		ssp = new(StartStreamProcessor)
	}
	ssp.failoverDryRun = &b
	return ssp
}

// Session sets the session for this operation.
func (ssp *StartStreamProcessor) Session(s *session.Client) *StartStreamProcessor {
	if ssp == nil {
		ssp = new(StartStreamProcessor)
	}
	ssp.session = s
	return ssp
}

// ClusterClock sets the cluster clock for this operation.
func (ssp *StartStreamProcessor) ClusterClock(c *session.ClusterClock) *StartStreamProcessor {
	if ssp == nil {
		ssp = new(StartStreamProcessor)
	}
	ssp.clock = c
	return ssp
}

// CommandMonitor sets the monitor to use for APM events.
func (ssp *StartStreamProcessor) CommandMonitor(m *event.CommandMonitor) *StartStreamProcessor {
	if ssp == nil {
		ssp = new(StartStreamProcessor)
	}
	ssp.monitor = m
	return ssp
}

// Crypt sets the Crypt object to use for automatic encryption and decryption.
func (ssp *StartStreamProcessor) Crypt(c driver.Crypt) *StartStreamProcessor {
	if ssp == nil {
		ssp = new(StartStreamProcessor)
	}
	ssp.crypt = c
	return ssp
}

// Database sets the database to run this operation against.
func (ssp *StartStreamProcessor) Database(database string) *StartStreamProcessor {
	if ssp == nil {
		ssp = new(StartStreamProcessor)
	}
	ssp.database = database
	return ssp
}

// Deployment sets the deployment to use for this operation.
func (ssp *StartStreamProcessor) Deployment(d driver.Deployment) *StartStreamProcessor {
	if ssp == nil {
		ssp = new(StartStreamProcessor)
	}
	ssp.deployment = d
	return ssp
}

// ServerSelector sets the selector used to retrieve a server.
func (ssp *StartStreamProcessor) ServerSelector(s description.ServerSelector) *StartStreamProcessor {
	if ssp == nil {
		ssp = new(StartStreamProcessor)
	}
	ssp.selector = s
	return ssp
}

// ServerAPI sets the server API version for this operation.
func (ssp *StartStreamProcessor) ServerAPI(s *driver.ServerAPIOptions) *StartStreamProcessor {
	if ssp == nil {
		ssp = new(StartStreamProcessor)
	}
	ssp.serverAPI = s
	return ssp
}

// Authenticator sets the authenticator to use for this operation.
func (ssp *StartStreamProcessor) Authenticator(a driver.Authenticator) *StartStreamProcessor {
	if ssp == nil {
		ssp = new(StartStreamProcessor)
	}
	ssp.authenticator = a
	return ssp
}
