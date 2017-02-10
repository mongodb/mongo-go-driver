package core

// Server represents a server.
type Server interface {
	// Connection gets a connection to the server.
	Connection() (Connection, error)
}
