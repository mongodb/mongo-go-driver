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
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
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
	observedEvents map[string]struct{}
	storedEvents   map[string][]string

	entityMap *EntityMap
}

func newClientEntity(ctx context.Context, em *EntityMap, entityOptions *entityOptions) (*clientEntity, error) {
	entity := &clientEntity{
		// The "configureFailPoint" command should always be ignored.
		ignoredCommands: map[string]struct{}{
			"configureFailPoint": {},
		},
		observedEvents: make(map[string]struct{}),
		storedEvents:   make(map[string][]string),
		entityMap:      em,
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
			case "commandStartedEvent", "commandSucceededEvent", "commandFailedEvent":
				entity.observedEvents[eventType] = struct{}{}
			default:
				return nil, fmt.Errorf("unrecognized observed event type %s", eventType)
			}
		}
		for _, eventsAsEntity := range entityOptions.StoreEventsAsEntities {
			for _, eventType := range eventsAsEntity.Events {
				switch eventType {
				case "CommandStartedEvent", "CommandSucceededEvent", "CommandFailedEvent",
					"PoolCreatedEvent", "PoolReadyEvent", "PoolClearedEvent", "PoolClosedEvent",
					"ConnectionCreatedEvent", "ConnectionReadyEvent", "ConnectionClosedEvent",
					"ConnectionCheckOutStartedEvent", "ConnectionCheckOutFailedEvent",
					"ConnectionCheckedOutEvent", "ConnectionCheckedInEvent":
					entity.storedEvents[eventType] = append(entity.storedEvents[eventType], eventsAsEntity.EventListID)
				default:
					return nil, fmt.Errorf("unrecognized stored event type %s", eventType)
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

func getSecondsSinceEpoch() int64 {
	return time.Now().Unix()
}

func (c *clientEntity) processStartedEvent(_ context.Context, evt *event.CommandStartedEvent) {
	if !c.getRecordEvents() {
		return
	}
	if _, ok := c.observedEvents["commandStartedEvent"]; ok {
		c.started = append(c.started, evt)
	}
	eventListIDs, ok := c.storedEvents["CommandStartedEvent"]
	if !ok {
		return
	}
	bsonBuilder := bsoncore.NewDocumentBuilder().
		AppendString("name", "CommandStartedEvent").
		AppendInt64("observedAt", getSecondsSinceEpoch()).
		AppendString("databaseName", evt.DatabaseName).
		AppendString("commandName", evt.CommandName).
		AppendInt64("requestId", evt.RequestID).
		AppendString("connectionId", evt.ConnectionID)
	if evt.ServerID != nil {
		bsonBuilder.AppendString("serverId", evt.ServerID.String())
	}
	doc := bson.Raw(bsonBuilder.Build())
	for _, id := range eventListIDs {
		c.entityMap.appendEventsEntity(id, doc)
	}
}

func (c *clientEntity) processSucceededEvent(_ context.Context, evt *event.CommandSucceededEvent) {
	if !c.getRecordEvents() {
		return
	}
	if _, ok := c.observedEvents["commandSucceededEvent"]; ok {
		c.succeeded = append(c.succeeded, evt)
	}
	eventListIDs, ok := c.storedEvents["CommandSucceededEvent"]
	if !ok {
		return
	}
	bsonBuilder := bsoncore.NewDocumentBuilder().
		AppendString("name", "CommandSucceededEvent").
		AppendInt64("observedAt", getSecondsSinceEpoch()).
		AppendString("commandName", evt.CommandName).
		AppendInt64("requestId", evt.RequestID).
		AppendString("connectionId", evt.ConnectionID)
	if evt.ServerID != nil {
		bsonBuilder.AppendString("serverId", evt.ServerID.String())
	}
	doc := bson.Raw(bsonBuilder.Build())
	for _, id := range eventListIDs {
		c.entityMap.appendEventsEntity(id, doc)
	}
}

func (c *clientEntity) processFailedEvent(_ context.Context, evt *event.CommandFailedEvent) {
	if !c.getRecordEvents() {
		return
	}
	if _, ok := c.observedEvents["commandFailedEvent"]; ok {
		c.failed = append(c.failed, evt)
	}
	eventListIDs, ok := c.storedEvents["CommandFailedEvent"]
	if !ok {
		return
	}
	bsonBuilder := bsoncore.NewDocumentBuilder().
		AppendString("name", "CommandFailedEvent").
		AppendInt64("observedAt", getSecondsSinceEpoch()).
		AppendInt64("durationNanos", evt.DurationNanos).
		AppendString("commandName", evt.CommandName).
		AppendInt64("requestId", evt.RequestID).
		AppendString("connectionId", evt.ConnectionID).
		AppendString("failure", evt.Failure)
	if evt.ServerID != nil {
		bsonBuilder.AppendString("serverId", evt.ServerID.String())
	}
	doc := bson.Raw(bsonBuilder.Build())
	for _, id := range eventListIDs {
		c.entityMap.appendEventsEntity(id, doc)
	}
}

func getPoolEventDocument(evt *event.PoolEvent, evtName string) bson.Raw {
	bsonBuilder := bsoncore.NewDocumentBuilder().
		AppendString("name", evtName).
		AppendInt64("observedAt", getSecondsSinceEpoch()).
		AppendString("address", evt.Address)
	if evt.ConnectionID != 0 {
		bsonBuilder.AppendString("connectionId", fmt.Sprint(evt.ConnectionID))
	}
	if evt.PoolOptions != nil {
		optionsDoc := bsoncore.NewDocumentBuilder().
			AppendString("maxPoolSize", fmt.Sprint(evt.PoolOptions.MaxPoolSize)).
			AppendString("minPoolSize", fmt.Sprint(evt.PoolOptions.MinPoolSize)).
			AppendString("maxIdleTimeMS", fmt.Sprint(evt.PoolOptions.WaitQueueTimeoutMS)).
			Build()
		bsonBuilder.AppendDocument("poolOptions", optionsDoc)
	}
	if evt.Reason != "" {
		bsonBuilder.AppendString("reason", evt.Reason)
	}
	return bson.Raw(bsonBuilder.Build())
}

func (c *clientEntity) processPoolEvent(evt *event.PoolEvent) {
	if !c.getRecordEvents() {
		return
	}
	var eventType string
	switch evt.Type {
	// TODO GODRIVER-1827: add storePoolReady and storeConnectionCheckOutStarted events
	case event.PoolCreated:
		eventType = "PoolCreatedEvent"
	case event.PoolCleared:
		eventType = "PoolClearedEvent"
	case event.PoolClosedEvent:
		eventType = "PoolClosedEvent"
	case event.ConnectionCreated:
		eventType = "ConnectionCreatedEvent"
	case event.ConnectionReady:
		eventType = "ConnectionReadyEvent"
	case event.ConnectionClosed:
		eventType = "ConnectionClosedEvent"
	case event.GetFailed:
		eventType = "ConnectionCheckOutFailedEvent"
	case event.GetSucceeded:
		eventType = "ConnectionCheckedOutEvent"
	case event.ConnectionReturned:
		eventType = "ConnectionCheckedInEvent"
	}
	eventListIDs, ok := c.storedEvents[eventType]
	if ok {
		eventBSON := getPoolEventDocument(evt, eventType)
		for _, id := range eventListIDs {
			c.entityMap.appendEventsEntity(id, eventBSON)
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
