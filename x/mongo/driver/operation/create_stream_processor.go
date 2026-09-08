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

// CreateStreamProcessor performs a createStreamProcessor operation against an
// Atlas Stream Processing workspace.
//
// Wire shape:
//
//	{
//	  createStreamProcessor: <name>,
//	  pipeline: <Document[]>,
//	  options: {
//	    dlq: <Document>,
//	    streamMetaFieldName: <string>,
//	    tier: <string>,
//	    failover: <bool>
//	  }
//	}
//
// The options sub-document is only included if at least one option is set.
type CreateStreamProcessor struct {
	authenticator driver.Authenticator
	session       *session.Client
	clock         *session.ClusterClock
	monitor       *event.CommandMonitor
	crypt         driver.Crypt
	database      string
	deployment    driver.Deployment
	selector      description.ServerSelector
	serverAPI     *driver.ServerAPIOptions

	name                string
	pipeline            bsoncore.Document
	dlq                 bsoncore.Document
	streamMetaFieldName *string
	tier                *string
	failoverEnabled     *bool
}

// NewCreateStreamProcessor constructs a new CreateStreamProcessor for the
// given processor name and aggregation pipeline.
func NewCreateStreamProcessor(name string, pipeline bsoncore.Document) *CreateStreamProcessor {
	return &CreateStreamProcessor{name: name, pipeline: pipeline}
}

// Execute runs this operation and returns an error if it does not execute successfully.
func (csp *CreateStreamProcessor) Execute(ctx context.Context) error {
	if csp.deployment == nil {
		return errors.New("the CreateStreamProcessor operation must have a Deployment set before Execute can be called")
	}

	return driver.Operation{
		CommandFn:      csp.command,
		Client:         csp.session,
		Clock:          csp.clock,
		CommandMonitor: csp.monitor,
		Crypt:          csp.crypt,
		Database:       csp.database,
		Deployment:     csp.deployment,
		Selector:       csp.selector,
		ServerAPI:      csp.serverAPI,
		Name:           driverutil.CreateStreamProcessorOp,
		Authenticator:  csp.authenticator,
	}.Execute(ctx)
}

func (csp *CreateStreamProcessor) command(dst []byte, _ description.SelectedServer) ([]byte, error) {
	dst = bsoncore.AppendStringElement(dst, "createStreamProcessor", csp.name)
	if csp.pipeline != nil {
		dst = bsoncore.AppendArrayElement(dst, "pipeline", csp.pipeline)
	}
	if csp.dlq != nil || csp.streamMetaFieldName != nil || csp.tier != nil || csp.failoverEnabled != nil {
		var optsIdx int32
		optsIdx, dst = bsoncore.AppendDocumentElementStart(dst, "options")
		if csp.dlq != nil {
			dst = bsoncore.AppendDocumentElement(dst, "dlq", csp.dlq)
		}
		if csp.streamMetaFieldName != nil {
			dst = bsoncore.AppendStringElement(dst, "streamMetaFieldName", *csp.streamMetaFieldName)
		}
		if csp.tier != nil {
			dst = bsoncore.AppendStringElement(dst, "tier", *csp.tier)
		}
		if csp.failoverEnabled != nil {
			dst = bsoncore.AppendBooleanElement(dst, "failover", *csp.failoverEnabled)
		}
		var err error
		dst, err = bsoncore.AppendDocumentEnd(dst, optsIdx)
		if err != nil {
			return nil, err
		}
	}
	return dst, nil
}

// DLQ sets the dead-letter-queue configuration document.
func (csp *CreateStreamProcessor) DLQ(dlq bsoncore.Document) *CreateStreamProcessor {
	if csp == nil {
		csp = new(CreateStreamProcessor)
	}
	csp.dlq = dlq
	return csp
}

// StreamMetaFieldName sets the field name to use for stream metadata.
func (csp *CreateStreamProcessor) StreamMetaFieldName(name string) *CreateStreamProcessor {
	if csp == nil {
		csp = new(CreateStreamProcessor)
	}
	csp.streamMetaFieldName = &name
	return csp
}

// Tier sets the compute tier for the new processor.
func (csp *CreateStreamProcessor) Tier(tier string) *CreateStreamProcessor {
	if csp == nil {
		csp = new(CreateStreamProcessor)
	}
	csp.tier = &tier
	return csp
}

// FailoverEnabled sets whether failover should be enabled.
func (csp *CreateStreamProcessor) FailoverEnabled(b bool) *CreateStreamProcessor {
	if csp == nil {
		csp = new(CreateStreamProcessor)
	}
	csp.failoverEnabled = &b
	return csp
}

// Session sets the session for this operation.
func (csp *CreateStreamProcessor) Session(s *session.Client) *CreateStreamProcessor {
	if csp == nil {
		csp = new(CreateStreamProcessor)
	}
	csp.session = s
	return csp
}

// ClusterClock sets the cluster clock for this operation.
func (csp *CreateStreamProcessor) ClusterClock(c *session.ClusterClock) *CreateStreamProcessor {
	if csp == nil {
		csp = new(CreateStreamProcessor)
	}
	csp.clock = c
	return csp
}

// CommandMonitor sets the monitor to use for APM events.
func (csp *CreateStreamProcessor) CommandMonitor(m *event.CommandMonitor) *CreateStreamProcessor {
	if csp == nil {
		csp = new(CreateStreamProcessor)
	}
	csp.monitor = m
	return csp
}

// Crypt sets the Crypt object to use for automatic encryption and decryption.
func (csp *CreateStreamProcessor) Crypt(c driver.Crypt) *CreateStreamProcessor {
	if csp == nil {
		csp = new(CreateStreamProcessor)
	}
	csp.crypt = c
	return csp
}

// Database sets the database to run this operation against.
func (csp *CreateStreamProcessor) Database(database string) *CreateStreamProcessor {
	if csp == nil {
		csp = new(CreateStreamProcessor)
	}
	csp.database = database
	return csp
}

// Deployment sets the deployment to use for this operation.
func (csp *CreateStreamProcessor) Deployment(d driver.Deployment) *CreateStreamProcessor {
	if csp == nil {
		csp = new(CreateStreamProcessor)
	}
	csp.deployment = d
	return csp
}

// ServerSelector sets the selector used to retrieve a server.
func (csp *CreateStreamProcessor) ServerSelector(s description.ServerSelector) *CreateStreamProcessor {
	if csp == nil {
		csp = new(CreateStreamProcessor)
	}
	csp.selector = s
	return csp
}

// ServerAPI sets the server API version for this operation.
func (csp *CreateStreamProcessor) ServerAPI(s *driver.ServerAPIOptions) *CreateStreamProcessor {
	if csp == nil {
		csp = new(CreateStreamProcessor)
	}
	csp.serverAPI = s
	return csp
}

// Authenticator sets the authenticator to use for this operation.
func (csp *CreateStreamProcessor) Authenticator(a driver.Authenticator) *CreateStreamProcessor {
	if csp == nil {
		csp = new(CreateStreamProcessor)
	}
	csp.authenticator = a
	return csp
}
