// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package event // import "go.mongodb.org/mongo-driver/event"

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/description"
)

// CommandStartedEvent represents an event generated when a command is sent to a server.
type CommandStartedEvent struct {
	Command      bson.Raw
	DatabaseName string
	CommandName  string
	RequestID    int64
	ConnectionID string
}

// CommandFinishedEvent represents a generic command finishing.
type CommandFinishedEvent struct {
	DurationNanos int64
	CommandName   string
	RequestID     int64
	ConnectionID  string
}

// CommandSucceededEvent represents an event generated when a command's execution succeeds.
type CommandSucceededEvent struct {
	CommandFinishedEvent
	Reply bson.Raw
}

// CommandFailedEvent represents an event generated when a command's execution fails.
type CommandFailedEvent struct {
	CommandFinishedEvent
	Failure string
}

// CommandMonitor represents a monitor that is triggered for different events.
type CommandMonitor struct {
	Started   func(context.Context, *CommandStartedEvent)
	Succeeded func(context.Context, *CommandSucceededEvent)
	Failed    func(context.Context, *CommandFailedEvent)
}

// strings for pool command monitoring reasons
const (
	ReasonIdle              = "idle"
	ReasonPoolClosed        = "poolClosed"
	ReasonStale             = "stale"
	ReasonConnectionErrored = "connectionError"
	ReasonTimedOut          = "timeout"
)

// strings for pool command monitoring types
const (
	ConnectionClosed   = "ConnectionClosed"
	PoolCreated        = "ConnectionPoolCreated"
	ConnectionCreated  = "ConnectionCreated"
	GetFailed          = "ConnectionCheckOutFailed"
	GetSucceeded       = "ConnectionCheckedOut"
	ConnectionReturned = "ConnectionCheckedIn"
	PoolCleared        = "ConnectionPoolCleared"
	PoolClosedEvent    = "ConnectionPoolClosed"
)

// MonitorPoolOptions contains pool options as formatted in pool events
type MonitorPoolOptions struct {
	MaxPoolSize        uint64 `json:"maxPoolSize"`
	MinPoolSize        uint64 `json:"minPoolSize"`
	WaitQueueTimeoutMS uint64 `json:"maxIdleTimeMS"`
}

// PoolEvent contains all information summarizing a pool event
type PoolEvent struct {
	Type         string              `json:"type"`
	Address      string              `json:"address"`
	ConnectionID uint64              `json:"connectionId"`
	PoolOptions  *MonitorPoolOptions `json:"options"`
	Reason       string              `json:"reason"`
}

// PoolMonitor is a function that allows the user to gain access to events occurring in the pool
type PoolMonitor struct {
	Event func(*PoolEvent)
}

// ServerDescription describes a server to a user.
type ServerDescription struct {
	Address  address.Address
	Arbiters []string
	Hosts    []string
	Passives []string
	Primary  address.Address
	SetName  string
	Kind     description.ServerKind
}

// TopologyDescription describes the current topology.
type TopologyDescription struct {
	Kind    description.TopologyKind
	Servers []ServerDescription
	SetName string
}

// ServerDescriptionChangedEvent represents a server description change.
type ServerDescriptionChangedEvent struct {
	Address             address.Address
	ID                  primitive.ObjectID
	PreviousDescription ServerDescription
	NewDescription      ServerDescription
}

// ServerOpeningEvent is an event generated when the server is initialized.
type ServerOpeningEvent struct {
	Address address.Address
	ID      primitive.ObjectID
}

// ServerClosedEvent is an event generated when the server is closed.
type ServerClosedEvent struct {
	Address address.Address
	ID      primitive.ObjectID
}

// TopologyDescriptionChangedEvent represents a topology description change.
type TopologyDescriptionChangedEvent struct {
	ID                  primitive.ObjectID
	PreviousDescription TopologyDescription
	NewDescription      TopologyDescription
}

// TopologyOpeningEvent is an event generated when the topology is initialized.
type TopologyOpeningEvent struct {
	ID primitive.ObjectID
}

// TopologyClosedEvent is an event generated when the topology is closed.
type TopologyClosedEvent struct {
	ID primitive.ObjectID
}

// ServerHeartbeatStartedEvent is an event generated when the ismaster command is started.
type ServerHeartbeatStartedEvent struct {
	ConnectionID string
	Awaited      bool
}

// ServerHeartbeatSucceededEvent is an event generated when the ismaster succeeds.
type ServerHeartbeatSucceededEvent struct {
	Duration     int64
	Reply        ServerDescription
	ConnectionID string
	Awaited      bool
}

// ServerHeartbeatFailedEvent is an event generated when the ismaster fails.
type ServerHeartbeatFailedEvent struct {
	Duration     int64
	Reply        error
	ConnectionID string
	Awaited      bool
}

// SdamMonitor represents a monitor that is triggered for different SDAM events.
type SdamMonitor struct {
	ServerDescriptionChanged   func(*ServerDescriptionChangedEvent)
	ServerOpening              func(*ServerOpeningEvent)
	ServerClosed               func(*ServerClosedEvent)
	TopologyDescriptionChanged func(*TopologyDescriptionChangedEvent)
	TopologyOpening            func(*TopologyOpeningEvent)
	TopologyClosed             func(*TopologyClosedEvent)
	ServerHeartbeatStarted     func(*ServerHeartbeatStartedEvent)
	ServerHeartbeatSucceeded   func(*ServerHeartbeatSucceededEvent)
	ServerHeartbeatFailed      func(*ServerHeartbeatFailedEvent)
}
