package auth

import (
	"context"
	"fmt"

	"github.com/10gen/mongo-go-driver/conn"
)

// AuthenticatorFactory constructs an authenticator.
type AuthenticatorFactory func(db, username, password string, props map[string]string) (Authenticator, error)

var authFactories = make(map[string]AuthenticatorFactory)

func init() {
	RegisterAuthenticatorFactory("", newDefaultAuthenticator)
	RegisterAuthenticatorFactory(scramSHA1, newScramSHA1Authenticator)
	RegisterAuthenticatorFactory(mongodbCR, newMongoDBCRAuthenticator)
	RegisterAuthenticatorFactory(plain, newPlainAuthenticator)
}

// CreateAuthenticator creates an authenticator.
func CreateAuthenticator(name, db, username, password string, props map[string]string) (Authenticator, error) {
	if f, ok := authFactories[name]; ok {
		return f(db, username, password, props)
	}

	return nil, fmt.Errorf("unknown authenticator: %s", name)
}

// RegisterAuthenticatorFactory registers the authenticator factory.
func RegisterAuthenticatorFactory(name string, factory AuthenticatorFactory) {
	authFactories[name] = factory
}

// Dialer returns a connection dialer that will open and authenticate the connection.
func Dialer(dialer conn.Dialer, authenticator Authenticator) conn.Dialer {
	return func(ctx context.Context, endpoint conn.Endpoint, opts ...conn.Option) (conn.Connection, error) {
		return Dial(dialer, authenticator, ctx, endpoint, opts...)
	}
}

// Dial opens a connection and authenticates it.
func Dial(dialer conn.Dialer, authenticator Authenticator, ctx context.Context, endpoint conn.Endpoint, opts ...conn.Option) (conn.Connection, error) {
	conn, err := dialer(ctx, endpoint, opts...)
	if err != nil {
		if conn != nil {
			conn.Close()
		}
		return nil, err
	}

	err = authenticator.Auth(ctx, conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

// Authenticator handles authenticating a connection.
type Authenticator interface {
	// Auth authenticates the connection.
	Auth(context.Context, conn.Connection) error
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
