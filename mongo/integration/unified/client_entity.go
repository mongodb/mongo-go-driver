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
	"go.mongodb.org/mongo-driver/internal/testutil"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
)

// clientEntity is a wrapper for a mongo.Client object that also holds additional information required during test
// execution.
type clientEntity struct {
	*mongo.Client

	recordEvents    atomic.Value
	started         []*event.CommandStartedEvent
	succeeded       []*event.CommandSucceededEvent
	failed          []*event.CommandFailedEvent
	ignoredCommands map[string]struct{}

	// These should not be changed after the clientEntity is initialized
	storePoolCreated               []string
	storePoolReady                 []string
	storePoolCleared               []string
	storePoolClosed                []string
	storeConnectionCreated         []string
	storeConnectionReady           []string
	storeConnectionClosed          []string
	storeConnectionCheckOutStarted []string
	storeConnectionCheckOutFailed  []string
	storeConnectionCheckedOut      []string
	storeConnectionCheckedIn       []string
	storeCommandStarted            []string
	storeCommandSucceeded          []string
	storeCommandFailed             []string
	observeCommandStarted          bool
	observeCommandSucceeded        bool
	observeCommandFailed           bool

	entityMap *EntityMap
}

func newClientEntity(ctx context.Context, em *EntityMap, entityOptions *entityOptions) (*clientEntity, error) {
	entity := &clientEntity{
		// The "configureFailPoint" command should always be ignored.
		ignoredCommands: map[string]struct{}{
			"configureFailPoint": {},
		},
		entityMap: em,
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

	if entityOptions.ObserveEvents != nil || entityOptions.StoreEventsAsEntities != nil {
		// Configure a command monitor that listens for the specified event types. We don't take the IgnoredCommands
		// option into account here because it can be overridden at the test level after the entity has already been
		// created, so we store the events for now but account for it when iterating over them later.
		monitor := &event.CommandMonitor{
			Started:   entity.processStartedEvent,
			Succeeded: entity.processSucceededEvent,
			Failed:    entity.processFailedEvent,
		}

		for _, eventType := range entityOptions.ObserveEvents {
			switch eventType {
			case "commandStartedEvent":
				entity.observeCommandStarted = true
			case "commandSucceededEvent":
				entity.observeCommandSucceeded = true
			case "commandFailedEvent":
				entity.observeCommandFailed = true
			default:
				return nil, fmt.Errorf("unrecognized event type %s", eventType)
			}
		}
		for _, eventsAsEntity := range entityOptions.StoreEventsAsEntities {
			for _, eventType := range eventsAsEntity.Events {
				switch eventType {
				case "CommandStartedEvent":
					entity.storeCommandStarted = append(entity.storeCommandStarted, eventsAsEntity.ID)
				case "CommandSucceededEvent":
					entity.storeCommandSucceeded = append(entity.storeCommandSucceeded, eventsAsEntity.ID)
				case "CommandFailedEvent":
					entity.storeCommandFailed = append(entity.storeCommandSucceeded, eventsAsEntity.ID)
				case "PoolCreatedEvent":
					entity.storePoolCreated = append(entity.storePoolCreated, eventsAsEntity.ID)
				case "PoolReadyEvent":
					entity.storePoolReady = append(entity.storePoolReady, eventsAsEntity.ID)
				case "PoolClearedEvent":
					entity.storePoolCleared = append(entity.storePoolCleared, eventsAsEntity.ID)
				case "PoolClosedEvent":
					entity.storePoolClosed = append(entity.storePoolClosed, eventsAsEntity.ID)
				case "ConnectionCreatedEvent":
					entity.storeConnectionCreated = append(entity.storeConnectionCreated, eventsAsEntity.ID)
				case "ConnectionReadyEvent":
					entity.storeConnectionReady = append(entity.storeConnectionReady, eventsAsEntity.ID)
				case "ConnectionClosedEvent":
					entity.storeConnectionClosed = append(entity.storeConnectionClosed, eventsAsEntity.ID)
				case "ConnectionCheckOutStartedEvent":
					entity.storeConnectionCheckOutStarted = append(entity.storeConnectionCheckOutStarted, eventsAsEntity.ID)
				case "ConnectionCheckOutFailedEvent":
					entity.storeConnectionCheckOutFailed = append(entity.storeConnectionCheckOutFailed, eventsAsEntity.ID)
				case "ConnectionCheckedOutEvent":
					entity.storeConnectionCheckedOut = append(entity.storeConnectionCheckedOut, eventsAsEntity.ID)
				case "ConnectionCheckedInEvent":
					entity.storeConnectionCheckedIn = append(entity.storeConnectionCheckedIn, eventsAsEntity.ID)
				default:
					return nil, fmt.Errorf("unrecognized event type %s", eventType)
				}
			}
		}

		poolMonitor := &event.PoolMonitor{
			Event: entity.processPoolEvent,
		}
		clientOpts.SetMonitor(monitor).SetPoolMonitor(poolMonitor)
	}
	if entityOptions.ServerAPIOptions != nil {
		clientOpts.SetServerAPIOptions(entityOptions.ServerAPIOptions.ServerAPIOptions)
	} else {
		testutil.AddTestServerAPIVersion(clientOpts)
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

func (c *clientEntity) stopListeningForEvents() {
	c.setRecordEvents(false)
}

func (c *clientEntity) startedEvents() []*event.CommandStartedEvent {
	var events []*event.CommandStartedEvent
	for _, evt := range c.started {
		if _, ok := c.ignoredCommands[evt.CommandName]; !ok {
			events = append(events, evt)
		}
	}

	return events
}

func (c *clientEntity) succeededEvents() []*event.CommandSucceededEvent {
	var events []*event.CommandSucceededEvent
	for _, evt := range c.succeeded {
		if _, ok := c.ignoredCommands[evt.CommandName]; !ok {
			events = append(events, evt)
		}
	}

	return events
}

func (c *clientEntity) failedEvents() []*event.CommandFailedEvent {
	var events []*event.CommandFailedEvent
	for _, evt := range c.failed {
		if _, ok := c.ignoredCommands[evt.CommandName]; !ok {
			events = append(events, evt)
		}
	}

	return events
}

func (c *clientEntity) processStartedEvent(_ context.Context, evt *event.CommandStartedEvent) {
	if c.getRecordEvents() {
		if c.observeCommandStarted {
			c.started = append(c.started, evt)
		}
		if c.storeCommandStarted != nil {
			record := bson.D{
				{"name", "CommandStartedEvent"},
				{"observedAt", time.Now().Unix()},
				{"databaseName", evt.DatabaseName},
				{"commandName", evt.CommandName},
				{"requestId", evt.RequestID},
				{"connnectionId", evt.ConnectionID},
				{"serverId", evt.ServerID},
			}
			eventBSON, _ := bson.Marshal(record)
			for _, id := range c.storeCommandStarted {
				c.entityMap.appendEventsEntity(id, eventBSON)
			}
		}
	}
}

func (c *clientEntity) processSucceededEvent(_ context.Context, evt *event.CommandSucceededEvent) {
	if c.getRecordEvents() {
		if c.observeCommandSucceeded {
			c.succeeded = append(c.succeeded, evt)
		}
		if c.storeCommandSucceeded != nil {
			record := bson.D{
				{"name", "CommandSucceededEvent"},
				{"observedAt", time.Now().Unix()},
				{"commandName", evt.CommandName},
				{"requestId", evt.RequestID},
				{"connnectionId", evt.ConnectionID},
				{"serverId", evt.ServerID},
			}
			eventBSON, _ := bson.Marshal(record)
			for _, id := range c.storeCommandSucceeded {
				c.entityMap.appendEventsEntity(id, eventBSON)
			}
		}
	}
}

func (c *clientEntity) processFailedEvent(_ context.Context, evt *event.CommandFailedEvent) {
	if c.getRecordEvents() {
		if c.observeCommandFailed {
			c.failed = append(c.failed, evt)
		}
		if c.storeCommandFailed != nil {
			record := bson.D{
				{"name", "CommandFailedEvent"},
				{"observedAt", time.Now().Unix()},
				{"durationNanos", evt.DurationNanos},
				{"commandName", evt.CommandName},
				{"requestId", evt.RequestID},
				{"connnectionId", evt.ConnectionID},
				{"serverId", evt.ServerID},
				{"failure", evt.Failure},
			}
			eventBSON, _ := bson.Marshal(record)
			for _, id := range c.storeCommandFailed {
				c.entityMap.appendEventsEntity(id, eventBSON)
			}
		}
	}
}

func (c *clientEntity) processPoolEvent(evt *event.PoolEvent) {
	switch evt.Type {
	// TODO GODRIVER-1827: add storePoolReady and storeConnectionCheckOutStarted events
	case event.PoolCreated:
		if c.storePoolCreated != nil {
			record := bson.D{
				{"name", evt.Type},
				{"observedAt", time.Now().Unix()},
				{"address", evt.Address},
				{"poolOptions", evt.PoolOptions},
			}
			eventBSON, _ := bson.Marshal(record)
			for _, id := range c.storePoolCreated {
				c.entityMap.appendEventsEntity(id, eventBSON)
			}
		}
	case event.PoolCleared:
		if c.storePoolCleared != nil {
			record := bson.D{
				{"name", evt.Type},
				{"observedAt", time.Now().Unix()},
				{"address", evt.Address},
			}
			eventBSON, _ := bson.Marshal(record)
			for _, id := range c.storePoolCleared {
				c.entityMap.appendEventsEntity(id, eventBSON)
			}
		}
	case event.PoolClosedEvent:
		if c.storePoolClosed != nil {
			record := bson.D{
				{"name", evt.Type},
				{"observedAt", time.Now().Unix()},
				{"address", evt.Address},
			}
			eventBSON, _ := bson.Marshal(record)
			for _, id := range c.storePoolClosed {
				c.entityMap.appendEventsEntity(id, eventBSON)
			}
		}
	case event.ConnectionCreated:
		if c.storeConnectionCreated != nil {
			record := bson.D{
				{"name", evt.Type},
				{"observedAt", time.Now().Unix()},
				{"address", evt.Address},
				{"connnectionId", evt.ConnectionID},
			}
			eventBSON, _ := bson.Marshal(record)
			for _, id := range c.storeConnectionCreated {
				c.entityMap.appendEventsEntity(id, eventBSON)
			}
		}
	case event.ConnectionReady:
		if c.storeConnectionReady != nil {
			record := bson.D{
				{"name", evt.Type},
				{"observedAt", time.Now().Unix()},
				{"address", evt.Address},
				{"connnectionId", evt.ConnectionID},
			}
			eventBSON, _ := bson.Marshal(record)
			for _, id := range c.storeConnectionReady {
				c.entityMap.appendEventsEntity(id, eventBSON)
			}
		}
	case event.ConnectionClosed:
		if c.storeConnectionClosed != nil {
			record := bson.D{
				{"name", evt.Type},
				{"observedAt", time.Now().Unix()},
				{"address", evt.Address},
				{"connnectionId", evt.ConnectionID},
				{"reason", evt.Reason},
			}
			eventBSON, _ := bson.Marshal(record)
			for _, id := range c.storeConnectionClosed {
				c.entityMap.appendEventsEntity(id, eventBSON)
			}
		}
	case event.GetFailed:
		if c.storeConnectionCheckOutFailed != nil {
			record := bson.D{
				{"name", evt.Type},
				{"observedAt", time.Now().Unix()},
				{"address", evt.Address},
				{"reason", evt.Reason},
			}
			eventBSON, _ := bson.Marshal(record)
			for _, id := range c.storeConnectionCheckOutFailed {
				c.entityMap.appendEventsEntity(id, eventBSON)
			}
		}
	case event.GetSucceeded:
		if c.storeConnectionCheckedOut != nil {
			record := bson.D{
				{"name", evt.Type},
				{"observedAt", time.Now().Unix()},
				{"address", evt.Address},
				{"connnectionId", evt.ConnectionID},
			}
			eventBSON, _ := bson.Marshal(record)
			for _, id := range c.storeConnectionCheckedOut {
				c.entityMap.appendEventsEntity(id, eventBSON)
			}
		}
	case event.ConnectionReturned:
		if c.storeConnectionCheckedIn != nil {
			record := bson.D{
				{"name", evt.Type},
				{"observedAt", time.Now().Unix()},
				{"address", evt.Address},
				{"connnectionId", evt.ConnectionID},
			}
			eventBSON, _ := bson.Marshal(record)
			for _, id := range c.storeConnectionCheckedIn {
				c.entityMap.appendEventsEntity(id, eventBSON)
			}
		}
	}
}

func (c *clientEntity) setRecordEvents(record bool) {
	c.recordEvents.Store(record)
}

func (c *clientEntity) getRecordEvents() bool {
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
