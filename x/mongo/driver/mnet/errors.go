package mnet

import "fmt"

// ErrConnectionClosed is returned from an attempt to use an already closed connection.
var ErrConnectionClosed = ConnectionError{ConnectionID: "<closed>", Message: "connection is closed"}

// Error represents a connection error.
type ConnectionError struct {
	ConnectionID string
	Wrapped      error

	// Init will be set to true if this error occurred during connection
	// initialization or during a connection handshake.
	Init    bool
	Message string
}

// Error implements the error interface.
func (e ConnectionError) Error() string {
	message := e.Message
	if e.Init {
		fullMsg := "error occurred during connection handshake"
		if message != "" {
			fullMsg = fmt.Sprintf("%s: %s", fullMsg, message)
		}
		message = fullMsg
	}
	if e.Wrapped != nil && message != "" {
		return fmt.Sprintf("connection(%s) %s: %s", e.ConnectionID, message, e.Wrapped.Error())
	}
	if e.Wrapped != nil {
		return fmt.Sprintf("connection(%s) %s", e.ConnectionID, e.Wrapped.Error())
	}
	return fmt.Sprintf("connection(%s) %s", e.ConnectionID, message)
}

// Unwrap returns the underlying error.
func (e ConnectionError) Unwrap() error {
	return e.Wrapped
}
