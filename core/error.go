package core

import "fmt"

// MessageError represents an error with a message.
type MessageError interface {
	// Message gets the basic message of the error.
	Message() string
}

// WrappedError represents an error that contains another error.
type WrappedError interface {
	MessageError

	// Inner gets the inner error if one exists.
	Inner() error
}

func rolledUpErrorMessage(err error) string {
	if wrappedErr, ok := err.(WrappedError); ok {
		inner := wrappedErr.Inner()
		if inner != nil {
			return fmt.Sprintf("%s: %s", wrappedErr.Message(), rolledUpErrorMessage(inner))
		}

		return wrappedErr.Message()
	}

	return err.Error()
}

func wrapError(inner error, message string) error {
	return &wrappedError{message, inner}
}

// WrapError wraps an error with a message.
func wrapErrorf(inner error, format string, args ...interface{}) error {
	return &wrappedError{fmt.Sprintf(format, args), inner}
}

type wrappedError struct {
	message string
	inner   error
}

func (e *wrappedError) Message() string {
	return e.message
}

func (e *wrappedError) Error() string {
	return rolledUpErrorMessage(e)
}

func (e *wrappedError) Inner() error {
	return e.inner
}

func newConnectionError(connectionID string, inner error, message string) *ConnectionError {
	return &ConnectionError{connectionID, "connection error: " + message, inner}
}

// ConnectionError represents an error that in the connection package.
type ConnectionError struct {
	ConnectionID string

	message string
	inner   error
}

// Message gets the basic error message.
func (e *ConnectionError) Message() string {
	return e.message
}

// Error gets a rolled-up error message.
func (e *ConnectionError) Error() string {
	return rolledUpErrorMessage(e)
}

// Inner gets the inner error if one exists.
func (e *ConnectionError) Inner() error {
	return e.inner
}
