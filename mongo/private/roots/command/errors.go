package command

import (
	"errors"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson"
)

var (
	// ErrUnknownCommandFailure occurs when a command fails for an unknown reason.
	ErrUnknownCommandFailure = errors.New("unknown command failure")
	// ErrNoCommandResponse occurs when the server sent no response document to a command.
	ErrNoCommandResponse = errors.New("no command response document")
	// ErrMultiDocCommandResponse occurs when the server sent multiple documents in response to a command.
	ErrMultiDocCommandResponse = errors.New("command returned multiple documents")
	// ErrNoDocCommandResponse occurs when the server indicated a response existed, but none was found.
	ErrNoDocCommandResponse = errors.New("command returned no documents")
)

// CommandFailureError is an error representing a command failure as a document.
type CommandFailureError struct {
	Message  string
	Response bson.Reader
}

// Error implements the error interface.
func (e CommandFailureError) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Response)
}

// CommandResponseError is an error parsing the response to a command.
type CommandResponseError struct {
	Message string
	Wrapped error
}

// NewCommandResponseError creates a CommandResponseError.
func NewCommandResponseError(msg string, err error) CommandResponseError {
	return CommandResponseError{Message: msg, Wrapped: err}
}

// Error implements the error interface.
func (e CommandResponseError) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("%s: %s", e.Message, e.Wrapped)
	}
	return fmt.Sprintf("%s", e.Message)
}

// CommandError is a command execution error from the database.
type CommandError struct {
	Code    int32
	Message string
	Name    string
}

// Error implements the error interface.
func (e CommandError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("(%v) %v", e.Name, e.Message)
	}
	return e.Message
}
