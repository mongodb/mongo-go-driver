package cluster

// Type represents a type of the cluster.
type Type uint32

// Type constants.
const (
	Unknown               Type = iota
	Single                Type = 1
	ReplicaSet            Type = 2
	ReplicaSetNoPrimary   Type = 4 + ReplicaSet
	ReplicaSetWithPrimary Type = 8 + ReplicaSet
	Sharded               Type = 256
)
