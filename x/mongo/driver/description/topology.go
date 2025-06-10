// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package description

import "fmt"

// TopologyKind represents a specific topology configuration.
type TopologyKind uint32

// These constants are the available topology configurations.
const (
	TopologyKindSingle                TopologyKind = 1
	TopologyKindReplicaSet            TopologyKind = 2
	TopologyKindReplicaSetNoPrimary   TopologyKind = 4 + TopologyKindReplicaSet
	TopologyKindReplicaSetWithPrimary TopologyKind = 8 + TopologyKindReplicaSet
	TopologyKindSharded               TopologyKind = 256
	TopologyKindLoadBalanced          TopologyKind = 512
)

// Topology contains information about a MongoDB cluster.
type Topology struct {
	Servers               []Server
	SetName               string
	Kind                  TopologyKind
	SessionTimeoutMinutes *int64
	CompatibilityErr      error
}

// String implements the Stringer interface.
func (t Topology) String() string {
	var serversStr string
	for _, s := range t.Servers {
		serversStr += "{ " + s.String() + " }, "
	}
	return fmt.Sprintf("Type: %s, Servers: [%s]", t.Kind, serversStr)
}

// String implements the fmt.Stringer interface.
func (kind TopologyKind) String() string {
	switch kind {
	case TopologyKindSingle:
		return "Single"
	case TopologyKindReplicaSet:
		return "ReplicaSet"
	case TopologyKindReplicaSetNoPrimary:
		return "ReplicaSetNoPrimary"
	case TopologyKindReplicaSetWithPrimary:
		return "ReplicaSetWithPrimary"
	case TopologyKindSharded:
		return "Sharded"
	case TopologyKindLoadBalanced:
		return "LoadBalanced"
	}

	return "Unknown"
}
