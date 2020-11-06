// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
)

// ClientEntity is a wrapper for a mongo.Client object that also holds additional information required during test
// execution.
type ClientEntity struct {
	*mongo.Client

	recordEvents    atomic.Value
	started         []*event.CommandStartedEvent
	succeeded       []*event.CommandSucceededEvent
	failed          []*event.CommandFailedEvent
	ignoredCommands map[string]struct{}
}

func NewClientEntity(ctx context.Context, entityOptions *EntityOptions) (*ClientEntity, error) {
	entity := &ClientEntity{
		// The "configureFailPoint" command should always be ignored.
		ignoredCommands: map[string]struct{}{
			"configureFailPoint": {},
		},
	}
	entity.setRecordEvents(true)

	// Construct a ClientOptions instance by first applying the cluster URI and then the URIOptions map to ensure that
	// the options specified in the test file take precedence.
	clientOpts := options.Client().ApplyURI(mtest.ClusterURI())
	if entityOptions.URIOptions != nil {
		if err := setClientOptionsFromURIOptions(clientOpts, entityOptions.URIOptions); err != nil {
			return nil, fmt.Errorf("error parsing URI options: %v", err)
		}
	}
	// UseMultipleMongoses is only relevant if we're connected to a sharded cluster. Options changes and validation are
	// only required if the option is explicitly set. If it's unset, we make no changes because the cluster URI already
	// includes all nodes and we don't enforce any limits on the number of nodes.
	if mtest.ClusterTopologyKind() == mtest.Sharded && entityOptions.UseMultipleMongoses != nil {
		if err := evaluateUseMultipleMongoses(clientOpts, *entityOptions.UseMultipleMongoses); err != nil {
			return nil, err
		}
	}
	if entityOptions.ObserveEvents != nil {
		// Configure a command monitor that listens for the specified event types. We don't take the IgnoredCommands
		// option into account here because it can be overridden at the test level after the entity has already been
		// created, so we store the events for now but account for it when iterating over them later.
		monitor := &event.CommandMonitor{}

		for _, eventType := range entityOptions.ObserveEvents {
			switch eventType {
			case "commandStartedEvent":
				monitor.Started = entity.processStartedEvent
			case "commandSucceededEvent":
				monitor.Succeeded = entity.processSucceededEvent
			case "commandFailedEvent":
				monitor.Failed = entity.processFailedEvent
			default:
				return nil, fmt.Errorf("unrecognized event type %s", eventType)
			}
		}
		clientOpts.SetMonitor(monitor)
	}
	for _, cmd := range entityOptions.IgnoredCommands {
		entity.ignoredCommands[cmd] = struct{}{}
	}

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("error creating mongo.Client: %v", err)
	}

	entity.Client = client
	return entity, nil
}

func (c *ClientEntity) StopListeningForEvents() {
	c.setRecordEvents(false)
}

func (c *ClientEntity) StartedEvents() []*event.CommandStartedEvent {
	var events []*event.CommandStartedEvent
	for _, evt := range c.started {
		if _, ok := c.ignoredCommands[evt.CommandName]; !ok {
			events = append(events, evt)
		}
	}

	return events
}

func (c *ClientEntity) SucceededEvents() []*event.CommandSucceededEvent {
	var events []*event.CommandSucceededEvent
	for _, evt := range c.succeeded {
		if _, ok := c.ignoredCommands[evt.CommandName]; !ok {
			events = append(events, evt)
		}
	}

	return events
}

func (c *ClientEntity) FailedEvents() []*event.CommandFailedEvent {
	var events []*event.CommandFailedEvent
	for _, evt := range c.failed {
		if _, ok := c.ignoredCommands[evt.CommandName]; !ok {
			events = append(events, evt)
		}
	}

	return events
}

func (c *ClientEntity) processStartedEvent(_ context.Context, evt *event.CommandStartedEvent) {
	if c.getRecordEvents() {
		c.started = append(c.started, evt)
	}
}

func (c *ClientEntity) processSucceededEvent(_ context.Context, evt *event.CommandSucceededEvent) {
	if c.getRecordEvents() {
		c.succeeded = append(c.succeeded, evt)
	}
}

func (c *ClientEntity) processFailedEvent(_ context.Context, evt *event.CommandFailedEvent) {
	if c.getRecordEvents() {
		c.failed = append(c.failed, evt)
	}
}

func (c *ClientEntity) setRecordEvents(record bool) {
	c.recordEvents.Store(record)
}

func (c *ClientEntity) getRecordEvents() bool {
	return c.recordEvents.Load().(bool)
}

func setClientOptionsFromURIOptions(clientOpts *options.ClientOptions, uriOpts bson.M) error {
	// A write concern can be constructed across multiple URI options (e.g. "w", "j", and "wTimeoutMS") so we declare an
	// empty writeConcern instance here that can be populated in the loop below.
	var wc writeConcern
	var wcSet bool

	for key, value := range uriOpts {
		switch key {
		case "heartbeatFrequencyMS":
			clientOpts.SetHeartbeatInterval(time.Duration(value.(int32)) * time.Millisecond)
		case "readConcernLevel":
			clientOpts.SetReadConcern(readconcern.New(readconcern.Level(value.(string))))
		case "retryReads":
			clientOpts.SetRetryReads(value.(bool))
		case "retryWrites":
			clientOpts.SetRetryWrites(value.(bool))
		case "w":
			wc.W = value
			wcSet = true
		default:
			return fmt.Errorf("unrecognized URI option %s", key)
		}
	}

	if wcSet {
		converted, err := wc.toWriteConcernOption()
		if err != nil {
			return fmt.Errorf("error creating write concern: %v", err)
		}
		clientOpts.SetWriteConcern(converted)
	}
	return nil
}

func evaluateUseMultipleMongoses(clientOpts *options.ClientOptions, useMultipleMongoses bool) error {
	hosts := mtest.ClusterConnString().Hosts

	if !useMultipleMongoses {
		clientOpts.SetHosts(hosts[:1])
		return nil
	}

	if len(hosts) < 2 {
		return fmt.Errorf("multiple mongoses required but cluster URI %q only contains one host", mtest.ClusterURI())
	}
	return nil
}
