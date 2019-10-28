// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package event // import "go.mongodb.org/mongo-driver/event"

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/mongo/driver/address"
	"go.mongodb.org/mongo-driver/x/mongo/driver/uuid"
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

// ServerAddress represents a server's address.
// This type will be a string value of the server's canonical address.
type ServerAddress string

// TopologyID represents a unique topology.
type TopologyID uuid.UUID

// ServerKind represents the type of a server.
type ServerKind uint32

// These constants are the possible types of servers.
const (
	Standalone  ServerKind = 1
	RSMember    ServerKind = 2
	RSPrimary   ServerKind = 4 + RSMember
	RSSecondary ServerKind = 8 + RSMember
	RSArbiter   ServerKind = 16 + RSMember
	RSGhost     ServerKind = 32 + RSMember
	Mongos      ServerKind = 256
)

// String implements the fmt.Stringer interface.
func (kind ServerKind) String() string {
	switch kind {
	case Standalone:
		return "Standalone"
	case RSMember:
		return "RSOther"
	case RSPrimary:
		return "RSPrimary"
	case RSSecondary:
		return "RSSecondary"
	case RSArbiter:
		return "RSArbiter"
	case RSGhost:
		return "RSGhost"
	case Mongos:
		return "Mongos"
	}

	return "Unknown"
}

// TopologyKind represents a specific topology configuration.
type TopologyKind uint32

// These constants are the available topology configurations.
const (
	Single                TopologyKind = 1
	ReplicaSet            TopologyKind = 2
	ReplicaSetNoPrimary   TopologyKind = 4 + ReplicaSet
	ReplicaSetWithPrimary TopologyKind = 8 + ReplicaSet
	Sharded               TopologyKind = 256
)

// String implements the fmt.Stringer interface.
func (kind TopologyKind) String() string {
	switch kind {
	case Single:
		return "Single"
	case ReplicaSet:
		return "ReplicaSet"
	case ReplicaSetNoPrimary:
		return "ReplicaSetNoPrimary"
	case ReplicaSetWithPrimary:
		return "ReplicaSetWithPrimary"
	case Sharded:
		return "Sharded"
	}

	return "Unknown"
}

// ServerDescription describes a server to a user.
type ServerDescription struct {
	Address  ServerAddress
	Arbiters []ServerAddress
	Hosts    []ServerAddress
	Passives []ServerAddress
	Primary  ServerAddress
	SetName  string
	Kind     ServerKind
}

// TopologyDescription describes the current topology.
type TopologyDescription struct {
	Kind    TopologyKind
	Servers []ServerDescription
	SetName string

	HasReadableServer func(readpref.Mode) bool
	HasWritableServer func() bool
}

// ServerDescriptionChangedEvent represents a server description change.
type ServerDescriptionChangedEvent struct {
	Address             ServerAddress
	ID                  TopologyID
	PreviousDescription ServerDescription
	NewDescription      ServerDescription
}

// ServerOpeningEvent is an event generated when the server is initialized.
type ServerOpeningEvent struct {
	Address ServerAddress
	ID      TopologyID
}

// ServerClosedEvent is an event generated when the server is closed.
type ServerClosedEvent struct {
	Address ServerAddress
	ID      TopologyID
}

// TopologyDescriptionChangedEvent represents a topology description change.
type TopologyDescriptionChangedEvent struct {
	ID                  TopologyID
	PreviousDescription TopologyDescription
	NewDescription      TopologyDescription
}

// TopologyOpeningEvent is an event generated when the topology is initialized.
type TopologyOpeningEvent struct {
	ID TopologyID
}

// TopologyClosedEvent is an event generated when the topology is closed.
type TopologyClosedEvent struct {
	ID TopologyID
}

// ServerHeartbeatStartedEvent is an event generated when the ismaster command is started.
type ServerHeartbeatStartedEvent struct {
	ConnectionID string
}

// ServerHeartbeatSucceededEvent is an event generated when the ismaster succeeds.
type ServerHeartbeatSucceededEvent struct {
	Duration     int64
	Reply        ServerDescription
	ConnectionID string
}

// ServerHeartbeatFailedEvent is an event generated when the ismaster fails.
type ServerHeartbeatFailedEvent struct {
	Duration     int64
	Reply        error
	ConnectionID string
}

// SdamMonitor represents a monitor that is triggered for different SDAM events.
type SdamMonitor struct {
	ServerDescriptionChanged   func(context.Context, *ServerDescriptionChangedEvent)
	ServerOpening              func(context.Context, *ServerOpeningEvent)
	ServerClosed               func(context.Context, *ServerClosedEvent)
	TopologyDescriptionChanged func(context.Context, *TopologyDescriptionChangedEvent)
	TopologyOpening            func(context.Context, *TopologyOpeningEvent)
	TopologyClosed             func(context.Context, *TopologyClosedEvent)
	ServerHeartbeatStarted     func(context.Context, *ServerHeartbeatStartedEvent)
	ServerHeartbeatSucceeded   func(context.Context, *ServerHeartbeatSucceededEvent)
	ServerHeartbeatFailed      func(context.Context, *ServerHeartbeatFailedEvent)
}

// AddressToServerAddress is a helper method
// that transforms []address.Address to []event.ServerAddress
func AddressToServerAddress(addresses []address.Address) []ServerAddress {
	var serverAddresses []ServerAddress
	for _, a := range addresses {
		serverAddresses = append(serverAddresses, ServerAddress(a.String()))
	}

	return serverAddresses
}
