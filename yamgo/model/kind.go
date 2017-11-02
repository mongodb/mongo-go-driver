// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package model

// Unknown is an unknown cluster or server kind.
const Unknown = 0

// ClusterKind represents a type of the cluster.
type ClusterKind uint32

// ClusterKind constants.
const (
	Single                ClusterKind = 1
	ReplicaSet            ClusterKind = 2
	ReplicaSetNoPrimary   ClusterKind = 4 + ReplicaSet
	ReplicaSetWithPrimary ClusterKind = 8 + ReplicaSet
	Sharded               ClusterKind = 256
)

// ServerKind represents a type of server.
type ServerKind uint32

// ServerKind constants.
const (
	Standalone  ServerKind = 1
	RSMember    ServerKind = 2
	RSPrimary   ServerKind = 4 + RSMember
	RSSecondary ServerKind = 8 + RSMember
	RSArbiter   ServerKind = 16 + RSMember
	RSGhost     ServerKind = 32 + RSMember
	Mongos      ServerKind = 256
)

func (kind ClusterKind) String() string {
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
