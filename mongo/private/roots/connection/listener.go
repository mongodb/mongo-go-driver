package connection

// Listener is a generic mongodb network protocol listener. It can return connections
// that speak the mongodb wire protocol.
//
// Multiple goroutines may invoke methods on a Listener simultaneously.
type Listener interface {
	// Accept waits for and returns the next Connection to the listener.
	Accept() (Connection, error)

	// Close closes the listener.
	Close() error

	// Addr returns the listener's network address.
	Addr() Addr
}

// Listen creates a new listener on the provided network and address.
func Listen(network, address string) (Listener, error) { return nil, nil }
