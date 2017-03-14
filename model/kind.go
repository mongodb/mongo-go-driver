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
