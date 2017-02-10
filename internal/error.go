package internal

import "fmt"

// WrappedError represents an error that contains another error.
type WrappedError interface {
	// Message gets the basic message of the error.
	Message() string
	// Inner gets the inner error if one exists.
	Inner() error
}

// RolledUpErrorMessage gets a flattened error message.
func RolledUpErrorMessage(err error) string {
	if wrappedErr, ok := err.(WrappedError); ok {
		inner := wrappedErr.Inner()
		if inner != nil {
			return fmt.Sprintf("%s: %s", wrappedErr.Message(), RolledUpErrorMessage(inner))
		}

		return wrappedErr.Message()
	}

	return err.Error()
}

// WrapError wraps an error with a message.
func WrapError(inner error, message string) error {
	return &wrappedError{message, inner}
}

// WrapErrorf wraps an error with a message.
func WrapErrorf(inner error, format string, args ...interface{}) error {
	return &wrappedError{fmt.Sprintf(format, args...), inner}
}

type wrappedError struct {
	message string
	inner   error
}

func (e *wrappedError) Message() string {
	return e.message
}

func (e *wrappedError) Error() string {
	return RolledUpErrorMessage(e)
}

func (e *wrappedError) Inner() error {
	return e.inner
}
