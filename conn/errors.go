package conn

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/mgo.v2/bson"
)

var (
	ErrUnknownCommandFailure   = errors.New("unknown command failure")
	ErrNoCommandResponse       = errors.New("no command response document")
	ErrMultiDocCommandResponse = errors.New("command returned multiple documents")
	ErrNoDocCommandResponse    = errors.New("command returned no documents")
)

// CommandFailureError is an error with a failure response as a document.
type CommandFailureError struct {
	Msg      string
	Response bson.D
}

func (e *CommandFailureError) Error() string {
	return fmt.Sprintf("%s: %v", e.Msg, e.Response)
}

// Message retrieves the message of the error.
func (e *CommandFailureError) Message() string {
	return e.Msg
}

// CommandResponseError is an error in the response to a command.
type CommandResponseError struct {
	Message string
}

func NewCommandResponseError(msg string) *CommandResponseError {
	return &CommandResponseError{msg}
}

func (e *CommandResponseError) Error() string {
	return e.Message
}

// CommandError is an error in the execution of a command.
type CommandError struct {
	Code    int32
	Message string
	Name    string
}

func (e *CommandError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("(%v) %v", e.Name, e.Message)
	}
	return e.Message
}

func IsNsNotFound(err error) bool {
	e, ok := err.(*CommandError)
	return ok && (e.Code == 26)
}

func IsCommandNotFound(err error) bool {
	e, ok := err.(*CommandError)
	return ok && (e.Code == 59 || e.Code == 13390 || strings.HasPrefix(e.Message, "no such cmd:"))
}
