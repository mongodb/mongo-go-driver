package auth

import (
	"context"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal/feature"
)

func newDefaultAuthenticator(db, username, password string, props map[string]string) (Authenticator, error) {
	return &DefaultAuthenticator{
		DB:       db,
		Username: username,
		Password: password,
	}, nil
}

// DefaultAuthenticator uses SCRAM-SHA-1 or MONGODB-CR depending
// on the server version.
type DefaultAuthenticator struct {
	DB       string
	Username string
	Password string
}

// Auth authenticates the connection.
func (a *DefaultAuthenticator) Auth(ctx context.Context, c conn.Connection) error {
	var actual Authenticator
	var err error
	if err = feature.ScramSHA1(c.Desc().Version); err != nil {
		actual, err = newMongoDBCRAuthenticator(a.DB, a.Username, a.Password, nil)
	} else {
		actual, err = newScramSHA1Authenticator(a.DB, a.Username, a.Password, nil)
	}

	if err != nil {
		return err
	}

	return actual.Auth(ctx, c)
}
