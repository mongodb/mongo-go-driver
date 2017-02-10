package desc

// Connection contains information about a connection.
type Connection struct {
	GitVersion          string
	Version             Version
	MaxBSONObjectSize   uint32
	MaxMessageSizeBytes uint32
	MaxWriteBatchSize   uint16
	WireVersion         Range
	ReadOnly            bool
}
