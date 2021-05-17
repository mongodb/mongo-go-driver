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

	recordEvents       atomic.Value
	started            []*event.CommandStartedEvent
	succeeded          []*event.CommandSucceededEvent
	failed             []*event.CommandFailedEvent
	pooled             []*event.PoolEvent
	ignoredCommands    map[string]struct{}
	numConnsCheckedOut int32

	// These should not be changed after the clientEntity is initialized
	observedEvents map[monitoringEventType]struct{}
	storedEvents   map[monitoringEventType][]string // maps an entity type to an array of entityIDs for entities that store it

	entityMap *EntityMap
}

func newClientEntity(ctx context.Context, em *EntityMap, entityOptions *entityOptions) (*clientEntity, error) {
	entity := &clientEntity{
		// The "configureFailPoint" command should always be ignored.
		ignoredCommands: map[string]struct{}{
			"configureFailPoint": {},
		},
		observedEvents: make(map[monitoringEventType]struct{}),
		storedEvents:   make(map[monitoringEventType][]string),
		entityMap:      em,
	}
	entity.setRecordEvents(true)

	// Construct a ClientOptions instance by first applying the cluster URI and then the URIOptions map to ensure that
	// the options specified in the test file take precedence.
	uri := getURIForClient(entityOptions)
	clientOpts := options.Client().ApplyURI(uri)
	if entityOptions.URIOptions != nil {
		if err := setClientOptionsFromURIOptions(clientOpts, entityOptions.URIOptions); err != nil {
			return nil, fmt.Errorf("error parsing URI options: %v", err)
		}
	}

	// UseMultipleMongoses requires validation when connecting to a sharded cluster. Options changes and validation are
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
		commandMonitor := &event.CommandMonitor{
			Started:   entity.processStartedEvent,
			Succeeded: entity.processSucceededEvent,
			Failed:    entity.processFailedEvent,
		}
		poolMonitor := &event.PoolMonitor{
			Event: entity.processPoolEvent,
		}
		clientOpts.SetMonitor(commandMonitor).SetPoolMonitor(poolMonitor)

		for _, eventTypeStr := range entityOptions.ObserveEvents {
			eventType, ok := monitoringEventTypeFromString(eventTypeStr)
			if !ok {
				return nil, fmt.Errorf("unrecognized observed event type %q", eventTypeStr)
			}
			entity.observedEvents[eventType] = struct{}{}
		}
		for _, eventsAsEntity := range entityOptions.StoreEventsAsEntities {
			for _, eventTypeStr := range eventsAsEntity.Events {
				eventType, ok := monitoringEventTypeFromString(eventTypeStr)
				if !ok {
					return nil, fmt.Errorf("unrecognized stored event type %q", eventTypeStr)
				}
				entity.storedEvents[eventType] = append(entity.storedEvents[eventType], eventsAsEntity.EventListID)
			}
		}
	}
	if entityOptions.ServerAPIOptions != nil {
		if err := entityOptions.ServerAPIOptions.ServerAPIVersion.Validate(); err != nil {
			return nil, err
		}
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

func getURIForClient(opts *entityOptions) string {
	if mtest.ClusterTopologyKind() != mtest.LoadBalanced {
		return mtest.ClusterURI()
	}

	// For load-balanced deployments, UseMultipleMongoses is used to determine the load balancer URI. If set to false,
	// the LB fronts a single server. If unset or explicitly true, the LB fronts multiple mongos servers.
	switch {
	case opts.UseMultipleMongoses != nil && !*opts.UseMultipleMongoses:
		return mtest.SingleMongosLoadBalancerURI()
	default:
		return mtest.MultiMongosLoadBalancerURI()
	}
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

func (c *clientEntity) poolEvents() []*event.PoolEvent {
	return c.pooled
}

func (c *clientEntity) numberConnectionsCheckedOut() int32 {
	return c.numConnsCheckedOut
}

func getSecondsSinceEpoch() float64 {
	return float64(time.Now().UnixNano()) / float64(time.Second/time.Nanosecond)
}

func (c *clientEntity) processStartedEvent(_ context.Context, evt *event.CommandStartedEvent) {
	if !c.getRecordEvents() {
		return
	}
	if _, ok := c.observedEvents[commandStartedEvent]; ok {
		c.started = append(c.started, evt)
	}
	eventListIDs, ok := c.storedEvents[commandStartedEvent]
	if !ok {
		return
	}
	bsonBuilder := bsoncore.NewDocumentBuilder().
		AppendString("name", "CommandStartedEvent").
		AppendDouble("observedAt", getSecondsSinceEpoch()).
		AppendString("databaseName", evt.DatabaseName).
		AppendString("commandName", evt.CommandName).
		AppendInt64("requestId", evt.RequestID).
		AppendString("connectionId", evt.ConnectionID)
	if evt.ServiceID != nil {
		bsonBuilder.AppendString("serviceId", evt.ServiceID.String())
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
	if _, ok := c.observedEvents[commandSucceededEvent]; ok {
		c.succeeded = append(c.succeeded, evt)
	}
	eventListIDs, ok := c.storedEvents["CommandSucceededEvent"]
	if !ok {
		return
	}
	bsonBuilder := bsoncore.NewDocumentBuilder().
		AppendString("name", "CommandSucceededEvent").
		AppendDouble("observedAt", getSecondsSinceEpoch()).
		AppendString("commandName", evt.CommandName).
		AppendInt64("requestId", evt.RequestID).
		AppendString("connectionId", evt.ConnectionID)
	if evt.ServiceID != nil {
		bsonBuilder.AppendString("serviceId", evt.ServiceID.String())
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
	if _, ok := c.observedEvents[commandFailedEvent]; ok {
		c.failed = append(c.failed, evt)
	}
	eventListIDs, ok := c.storedEvents["CommandFailedEvent"]
	if !ok {
		return
	}
	bsonBuilder := bsoncore.NewDocumentBuilder().
		AppendString("name", "CommandFailedEvent").
		AppendDouble("observedAt", getSecondsSinceEpoch()).
		AppendInt64("durationNanos", evt.DurationNanos).
		AppendString("commandName", evt.CommandName).
		AppendInt64("requestId", evt.RequestID).
		AppendString("connectionId", evt.ConnectionID).
		AppendString("failure", evt.Failure)
	if evt.ServiceID != nil {
		bsonBuilder.AppendString("serviceId", evt.ServiceID.String())
	}
	doc := bson.Raw(bsonBuilder.Build())
	for _, id := range eventListIDs {
		c.entityMap.appendEventsEntity(id, doc)
	}
}

func getPoolEventDocument(evt *event.PoolEvent, eventType monitoringEventType) bson.Raw {
	bsonBuilder := bsoncore.NewDocumentBuilder().
		AppendString("name", string(eventType)).
		AppendDouble("observedAt", getSecondsSinceEpoch()).
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
	if evt.ServiceID != nil {
		bsonBuilder.AppendString("serviceId", evt.ServiceID.String())
	}
	return bson.Raw(bsonBuilder.Build())
}

func (c *clientEntity) processPoolEvent(evt *event.PoolEvent) {
	if !c.getRecordEvents() {
		return
	}

	// Update the connection counter. This happens even if we're not storing any events.
	switch evt.Type {
	case event.GetSucceeded:
		c.numConnsCheckedOut++
	case event.ConnectionReturned:
		c.numConnsCheckedOut--
	}

	eventType := monitoringEventTypeFromPoolEvent(evt)
	if _, ok := c.observedEvents[eventType]; ok {
		c.pooled = append(c.pooled, evt)
	}
	if eventListIDs, ok := c.storedEvents[eventType]; ok {
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
		case "appname":
			clientOpts.SetAppName(value.(string))
		case "heartbeatFrequencyMS":
			clientOpts.SetHeartbeatInterval(time.Duration(value.(int32)) * time.Millisecond)
		case "loadBalanced":
			clientOpts.SetLoadBalanced(value.(bool))
		case "maxPoolSize":
			clientOpts.SetMaxPoolSize(uint64(value.(int32)))
		case "readConcernLevel":
			clientOpts.SetReadConcern(readconcern.New(readconcern.Level(value.(string))))
		case "retryReads":
			clientOpts.SetRetryReads(value.(bool))
		case "retryWrites":
			clientOpts.SetRetryWrites(value.(bool))
		case "w":
			wc.W = value
			wcSet = true
		case "waitQueueTimeoutMS":
			return newSkipTestError("the waitQueueTimeoutMS client option is not supported")
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
