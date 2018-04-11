package connection

import "fmt"

// Error represents a connection error.
type Error struct {
	ConnectionID string
	Wrapped      error

	message string
}

// Error implements the error interface.
func (e Error) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("connection(%s) %s: %s", e.ConnectionID, e.message, e.Wrapped.Error())
	}
	return fmt.Sprintf("connection(%s) %s", e.ConnectionID, e.message)
}

// NetworkError represents an error that ocurred while reading from or writing
// to a network socket.
type NetworkError struct {
	ConnectionID string
	Wrapped      error
}

func (ne NetworkError) Error() string {
	return fmt.Sprintf("connection(%s): %s", ne.ConnectionID, ne.Wrapped.Error())
}

// PoolError is an error returned from a Pool method.
type PoolError string

func (pe PoolError) Error() string { return string(pe) }
