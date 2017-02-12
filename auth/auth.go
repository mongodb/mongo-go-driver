package auth

import (
	"fmt"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/desc"
)

// NewDialer returns a connection dialer that will open and authenticate the connection.
func NewDialer(dialer conn.Dialer, authenticator Authenticator) conn.Dialer {
	return func(endpoint desc.Endpoint, opts ...conn.Option) (conn.ConnectionCloser, error) {
		return Dial(dialer, authenticator, endpoint, opts...)
	}
}

// Dial opens a connection and authenticates it.
func Dial(dialer conn.Dialer, authenticator Authenticator, endpoint desc.Endpoint, opts ...conn.Option) (conn.ConnectionCloser, error) {
	conn, err := dialer(endpoint, opts...)
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
	Auth(conn.Connection) error
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
