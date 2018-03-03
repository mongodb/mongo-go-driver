package description

type TopologyKind uint32

const (
	Single                TopologyKind = 1
	ReplicaSet            TopologyKind = 2
	ReplicaSetNoPrimary   TopologyKind = 4 + ReplicaSet
	ReplicaSetWithPrimary TopologyKind = 8 + ReplicaSet
	Sharded               TopologyKind = 256
)

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
