// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/handshake"
	"go.mongodb.org/mongo-driver/internal/integtest"
	"go.mongodb.org/mongo-driver/internal/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

// There are no automated tests for truncation. Given that, setting the
// "MaxDocumentLength" to 10_000 will ensure that the default truncation
// length does not interfere with tests with commands/replies that
// exceed the default truncation length.
const defaultMaxDocumentLen = 10_000

// Security-sensitive commands that should be ignored in command monitoring by default.
var securitySensitiveCommands = []string{
	"authenticate", "saslStart", "saslContinue", "getnonce",
	"createUser", "updateUser", "copydbgetnonce", "copydbsaslstart", "copydb",
}

// clientEntity is a wrapper for a mongo.Client object that also holds additional information required during test
// execution.
type clientEntity struct {
	*mongo.Client
	disconnected bool

	recordEvents                atomic.Value
	started                     []*event.CommandStartedEvent
	succeeded                   []*event.CommandSucceededEvent
	failed                      []*event.CommandFailedEvent
	pooled                      []*event.PoolEvent
	serverDescriptionChanged    []*event.ServerDescriptionChangedEvent
	serverHeartbeatFailedEvent  []*event.ServerHeartbeatFailedEvent
	serverHeartbeatStartedEvent []*event.ServerHeartbeatStartedEvent
	serverHeartbeatSucceeded    []*event.ServerHeartbeatSucceededEvent
	topologyDescriptionChanged  []*event.TopologyDescriptionChangedEvent
	ignoredCommands             map[string]struct{}
	observeSensitiveCommands    *bool
	numConnsCheckedOut          int32

	// These should not be changed after the clientEntity is initialized
	observedEvents                      map[monitoringEventType]struct{}
	storedEvents                        map[monitoringEventType][]string
	eventsCount                         map[monitoringEventType]int32
	serverDescriptionChangedEventsCount map[serverDescriptionChangedEventInfo]int32

	eventsCountLock                         sync.RWMutex
	serverDescriptionChangedEventsCountLock sync.RWMutex
	eventProcessMu                          sync.RWMutex

	entityMap *EntityMap

	logQueue chan orderedLogMessage
}

func newClientEntity(ctx context.Context, em *EntityMap, entityOptions *entityOptions) (*clientEntity, error) {
	// The "configureFailPoint" command should always be ignored.
	ignoredCommands := map[string]struct{}{
		"configureFailPoint": {},
	}
	// If not observing sensitive commands, add security-sensitive commands
	// to ignoredCommands by default.
	if entityOptions.ObserveSensitiveCommands == nil || !*entityOptions.ObserveSensitiveCommands {
		for _, cmd := range securitySensitiveCommands {
			ignoredCommands[cmd] = struct{}{}
		}
	}
	entity := &clientEntity{
		ignoredCommands:                     ignoredCommands,
		observedEvents:                      make(map[monitoringEventType]struct{}),
		storedEvents:                        make(map[monitoringEventType][]string),
		eventsCount:                         make(map[monitoringEventType]int32),
		serverDescriptionChangedEventsCount: make(map[serverDescriptionChangedEventInfo]int32),
		entityMap:                           em,
		observeSensitiveCommands:            entityOptions.ObserveSensitiveCommands,
	}
	entity.setRecordEvents(true)

	// Construct a ClientOptions instance by first applying the cluster URI and then the URIOptions map to ensure that
	// the options specified in the test file take precedence.
	uri := getURIForClient(entityOptions)
	clientOpts := options.Client().ApplyURI(uri)
	if entityOptions.URIOptions != nil {
		if err := setClientOptionsFromURIOptions(clientOpts, entityOptions.URIOptions); err != nil {
			return nil, fmt.Errorf("error parsing URI options: %w", err)
		}
	}

	if olm := entityOptions.ObserveLogMessages; olm != nil {
		expectedLogMessagesCount := expectedLogMessagesCount(ctx, entityOptions.ID)
		ignoreLogMessages := ignoreLogMessages(ctx, entityOptions.ID)

		clientLogger := newLogger(olm, expectedLogMessagesCount, ignoreLogMessages)

		wrap := func(str string) options.LogLevel {
			return options.LogLevel(logger.ParseLevel(str))
		}

		// Assign the log queue to the entity so that it can be used to
		// retrieve log messages.
		entity.logQueue = clientLogger.logQueue

		// Update the client options to add the clientLogger.
		clientOpts.LoggerOptions = options.Logger().
			SetComponentLevel(options.LogComponentCommand, wrap(olm.Command)).
			SetComponentLevel(options.LogComponentTopology, wrap(olm.Topology)).
			SetComponentLevel(options.LogComponentServerSelection, wrap(olm.ServerSelection)).
			SetComponentLevel(options.LogComponentConnection, wrap(olm.Connection)).
			SetMaxDocumentLength(defaultMaxDocumentLen).
			SetSink(clientLogger)
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

		serverMonitor := &event.ServerMonitor{
			ServerDescriptionChanged:   entity.processServerDescriptionChangedEvent,
			ServerHeartbeatFailed:      entity.processServerHeartbeatFailedEvent,
			ServerHeartbeatStarted:     entity.processServerHeartbeatStartedEvent,
			ServerHeartbeatSucceeded:   entity.processServerHeartbeatSucceededEvent,
			TopologyDescriptionChanged: entity.processTopologyDescriptionChangedEvent,
		}

		clientOpts.SetMonitor(commandMonitor).SetPoolMonitor(poolMonitor).SetServerMonitor(serverMonitor)

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
		integtest.AddTestServerAPIVersion(clientOpts)
	}
	for _, cmd := range entityOptions.IgnoredCommands {
		entity.ignoredCommands[cmd] = struct{}{}
	}

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("error creating mongo.Client: %w", err)
	}

	entity.Client = client
	return entity, nil
}

func getURIForClient(opts *entityOptions) string {
	if mtest.Serverless() || mtest.ClusterTopologyKind() != mtest.LoadBalanced {
		return mtest.ClusterURI()
	}

	// For non-serverless load-balanced deployments, UseMultipleMongoses is used to determine the load balancer URI. If set to false,
	// the LB fronts a single server. If unset or explicitly true, the LB fronts multiple mongos servers.
	switch {
	case opts.UseMultipleMongoses != nil && !*opts.UseMultipleMongoses:
		return mtest.SingleMongosLoadBalancerURI()
	default:
		return mtest.MultiMongosLoadBalancerURI()
	}
}

// disconnect disconnects the client associated with this entity. It is an
// idempotent operation, unlike the mongo client's disconnect method. This
// property will help avoid unnecessary errors when calling disconnect on a
// client that has already been disconnected, such as the case when the test
// runner is required to run the closure as part of an operation.
func (c *clientEntity) disconnect(ctx context.Context) error {
	if c.disconnected {
		return nil
	}

	if err := c.Client.Disconnect(ctx); err != nil {
		return err
	}

	c.disconnected = true

	return nil
}

func (c *clientEntity) stopListeningForEvents() {
	c.setRecordEvents(false)
}

func (c *clientEntity) isIgnoredEvent(commandName string, eventDoc bson.Raw) bool {
	// Check if command is in ignoredCommands.
	if _, ok := c.ignoredCommands[commandName]; ok {
		return true
	}

	if commandName == "hello" || strings.ToLower(commandName) == handshake.LegacyHelloLowercase {
		// If observeSensitiveCommands is false (or unset) and hello command has been
		// redacted at operation level, hello command should be ignored as it contained
		// speculativeAuthenticate.
		sensitiveCommandsIgnored := c.observeSensitiveCommands == nil || !*c.observeSensitiveCommands
		redacted := len(eventDoc) == 0
		if sensitiveCommandsIgnored && redacted {
			return true
		}
	}
	return false
}

func (c *clientEntity) startedEvents() []*event.CommandStartedEvent {
	var events []*event.CommandStartedEvent
	for _, evt := range c.started {
		if !c.isIgnoredEvent(evt.CommandName, evt.Command) {
			events = append(events, evt)
		}
	}

	return events
}

func (c *clientEntity) succeededEvents() []*event.CommandSucceededEvent {
	var events []*event.CommandSucceededEvent
	for _, evt := range c.succeeded {
		if !c.isIgnoredEvent(evt.CommandName, evt.Reply) {
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

func (c *clientEntity) addEventsCount(eventType monitoringEventType) {
	c.eventsCountLock.Lock()
	defer c.eventsCountLock.Unlock()

	c.eventsCount[eventType]++
}

func (c *clientEntity) addServerDescriptionChangedEventCount(evt serverDescriptionChangedEventInfo) {
	c.serverDescriptionChangedEventsCountLock.Lock()
	defer c.serverDescriptionChangedEventsCountLock.Unlock()

	c.serverDescriptionChangedEventsCount[evt]++
}

func (c *clientEntity) getEventCount(eventType monitoringEventType) int32 {
	c.eventsCountLock.RLock()
	defer c.eventsCountLock.RUnlock()

	return c.eventsCount[eventType]
}

func (c *clientEntity) getServerDescriptionChangedEventCount(evt serverDescriptionChangedEventInfo) int32 {
	c.serverDescriptionChangedEventsCountLock.Lock()
	defer c.serverDescriptionChangedEventsCountLock.Unlock()

	return c.serverDescriptionChangedEventsCount[evt]
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

	c.addEventsCount(commandStartedEvent)

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

	c.addEventsCount(commandSucceededEvent)

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

	c.addEventsCount(commandFailedEvent)

	eventListIDs, ok := c.storedEvents["CommandFailedEvent"]
	if !ok {
		return
	}
	bsonBuilder := bsoncore.NewDocumentBuilder().
		AppendString("name", "CommandFailedEvent").
		AppendDouble("observedAt", getSecondsSinceEpoch()).
		AppendInt64("durationNanos", evt.Duration.Nanoseconds()).
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

	c.addEventsCount(eventType)

	if eventListIDs, ok := c.storedEvents[eventType]; ok {
		eventBSON := getPoolEventDocument(evt, eventType)
		for _, id := range eventListIDs {
			c.entityMap.appendEventsEntity(id, eventBSON)
		}
	}
}

func (c *clientEntity) processServerDescriptionChangedEvent(evt *event.ServerDescriptionChangedEvent) {
	c.eventProcessMu.Lock()
	defer c.eventProcessMu.Unlock()

	if !c.getRecordEvents() {
		return
	}

	if _, ok := c.observedEvents[serverDescriptionChangedEvent]; ok {
		c.serverDescriptionChanged = append(c.serverDescriptionChanged, evt)
	}

	// Record object-specific unified spec test data on an event.
	c.addServerDescriptionChangedEventCount(*newServerDescriptionChangedEventInfo(evt))

	// Record the event generally.
	c.addEventsCount(serverDescriptionChangedEvent)
}

func (c *clientEntity) processServerHeartbeatFailedEvent(evt *event.ServerHeartbeatFailedEvent) {
	c.eventProcessMu.Lock()
	defer c.eventProcessMu.Unlock()

	if !c.getRecordEvents() {
		return
	}

	if _, ok := c.observedEvents[serverHeartbeatFailedEvent]; ok {
		c.serverHeartbeatFailedEvent = append(c.serverHeartbeatFailedEvent, evt)
	}

	c.addEventsCount(serverHeartbeatFailedEvent)
}

func (c *clientEntity) processServerHeartbeatStartedEvent(evt *event.ServerHeartbeatStartedEvent) {
	c.eventProcessMu.Lock()
	defer c.eventProcessMu.Unlock()

	if !c.getRecordEvents() {
		return
	}

	if _, ok := c.observedEvents[serverHeartbeatStartedEvent]; ok {
		c.serverHeartbeatStartedEvent = append(c.serverHeartbeatStartedEvent, evt)
	}

	c.addEventsCount(serverHeartbeatStartedEvent)
}

func (c *clientEntity) processServerHeartbeatSucceededEvent(evt *event.ServerHeartbeatSucceededEvent) {
	c.eventProcessMu.Lock()
	defer c.eventProcessMu.Unlock()

	if !c.getRecordEvents() {
		return
	}

	if _, ok := c.observedEvents[serverHeartbeatSucceededEvent]; ok {
		c.serverHeartbeatSucceeded = append(c.serverHeartbeatSucceeded, evt)
	}

	c.addEventsCount(serverHeartbeatSucceededEvent)
}

func (c *clientEntity) processTopologyDescriptionChangedEvent(evt *event.TopologyDescriptionChangedEvent) {
	c.eventProcessMu.Lock()
	defer c.eventProcessMu.Unlock()

	if !c.getRecordEvents() {
		return
	}

	if _, ok := c.observedEvents[topologyDescriptionChangedEvent]; ok {
		c.topologyDescriptionChanged = append(c.topologyDescriptionChanged, evt)
	}

	c.addEventsCount(topologyDescriptionChangedEvent)
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
		switch strings.ToLower(key) {
		case "appname":
			clientOpts.SetAppName(value.(string))
		case "connecttimeoutms":
			clientOpts.SetConnectTimeout(time.Duration(value.(int32)) * time.Millisecond)
		case "heartbeatfrequencyms":
			clientOpts.SetHeartbeatInterval(time.Duration(value.(int32)) * time.Millisecond)
		case "loadbalanced":
			clientOpts.SetLoadBalanced(value.(bool))
		case "maxidletimems":
			clientOpts.SetMaxConnIdleTime(time.Duration(value.(int32)) * time.Millisecond)
		case "minpoolsize":
			clientOpts.SetMinPoolSize(uint64(value.(int32)))
		case "maxpoolsize":
			clientOpts.SetMaxPoolSize(uint64(value.(int32)))
		case "maxconnecting":
			clientOpts.SetMaxConnecting(uint64(value.(int32)))
		case "readconcernlevel":
			clientOpts.SetReadConcern(readconcern.New(readconcern.Level(value.(string))))
		case "retryreads":
			clientOpts.SetRetryReads(value.(bool))
		case "retrywrites":
			clientOpts.SetRetryWrites(value.(bool))
		case "sockettimeoutms":
			clientOpts.SetSocketTimeout(time.Duration(value.(int32)) * time.Millisecond)
		case "w":
			wc.W = value
			wcSet = true
		case "waitqueuetimeoutms":
			return newSkipTestError("the waitQueueTimeoutMS client option is not supported")
		case "waitqueuesize":
			return newSkipTestError("the waitQueueSize client option is not supported")
		case "timeoutms":
			clientOpts.SetTimeout(time.Duration(value.(int32)) * time.Millisecond)
		case "serverselectiontimeoutms":
			clientOpts.SetServerSelectionTimeout(time.Duration(value.(int32)) * time.Millisecond)
		case "servermonitoringmode":
			clientOpts.SetServerMonitoringMode(value.(string))
		default:
			return fmt.Errorf("unrecognized URI option %s", key)
		}
	}

	if wcSet {
		converted, err := wc.toWriteConcernOption()
		if err != nil {
			return fmt.Errorf("error creating write concern: %w", err)
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
