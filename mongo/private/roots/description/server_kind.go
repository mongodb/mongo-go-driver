package description

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
