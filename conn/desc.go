package conn

// Desc contains a description of a connection.
type Desc struct {
	Endpoint            Endpoint
	GitVersion          string
	Version             Version
	MaxBSONObjectSize   uint32
	MaxMessageSizeBytes uint32
	MaxWriteBatchSize   uint16
	WireVersion         Range
	ReadOnly            bool
}
