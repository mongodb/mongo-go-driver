package server

// Type represents a type of server.
type Type uint32

// Type constants.
const (
	Unknown     Type = 0
	Standalone  Type = 1
	RSMember    Type = 2
	RSPrimary   Type = 4 + RSMember
	RSSecondary Type = 8 + RSMember
	RSArbiter   Type = 16 + RSMember
	RSGhost     Type = 32 + RSMember
	Mongos      Type = 256
)
