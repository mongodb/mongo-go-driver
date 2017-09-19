package auth

import (
	"context"

	"github.com/10gen/mongo-go-driver/yamgo/internal/feature"
	"github.com/10gen/mongo-go-driver/yamgo/private/conn"
)

func newDefaultAuthenticator(cred *Cred) (Authenticator, error) {
	return &DefaultAuthenticator{
		Cred: cred,
	}, nil
}

// DefaultAuthenticator uses SCRAM-SHA-1 or MONGODB-CR depending
// on the server version.
type DefaultAuthenticator struct {
	Cred *Cred
}

// Auth authenticates the connection.
func (a *DefaultAuthenticator) Auth(ctx context.Context, c conn.Connection) error {
	var actual Authenticator
	var err error
	if err = feature.ScramSHA1(c.Model().Version); err != nil {
		actual, err = newMongoDBCRAuthenticator(a.Cred)
	} else {
		actual, err = newScramSHA1Authenticator(a.Cred)
	}

	if err != nil {
		return err
	}

	return actual.Auth(ctx, c)
}
