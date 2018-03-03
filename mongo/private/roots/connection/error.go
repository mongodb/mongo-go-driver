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

type NetworkError struct {
	ConnectionID string
	Wrapped      error
}

func (ne NetworkError) Error() string {
	return fmt.Sprintf("connection(%s): %s", ne.Wrapped.Error())
}
