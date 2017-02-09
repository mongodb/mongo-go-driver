package auth

import (
	"fmt"

	"github.com/10gen/mongo-go-driver/core"
)

// NewConnectionDialer returns a connection dialer that will authenticate the connection.
func NewConnectionDialer(dialer core.ConnectionDialer, authenticator Authenticator) core.ConnectionDialer {
	return func(opts core.ConnectionOptions) (core.ConnectionCloser, error) {
		return DialConnection(dialer, authenticator, opts)
	}
}

// DialConnection opens a connection that will authenticate the connection.
func DialConnection(dialer core.ConnectionDialer, authenticator Authenticator, opts core.ConnectionOptions) (core.ConnectionCloser, error) {
	conn, err := dialer(opts)
	if err != nil {
		if conn != nil {
			conn.Close()
		}
		return nil, err
	}

	err = authenticator.Auth(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

// Authenticator handles authenticating a connection.
type Authenticator interface {
	// Auth authenticates the connection.
	Auth(core.Connection) error
}

func newError(err error, mech string) error {
	return &Error{
		message: fmt.Sprintf("unable to authenticate using mechanism \"%s\"", mech),
		inner:   err,
	}
}

// Error is an error that occured during authentication.
type Error struct {
	message string
	inner   error
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.message, e.inner)
}

// Inner returns the wrapped error.
func (e *Error) Inner() error {
	return e.inner
}

// Message returns the message.
func (e *Error) Message() string {
	return e.message
}
