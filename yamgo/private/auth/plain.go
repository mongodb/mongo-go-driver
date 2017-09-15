package auth

import (
	"context"
	"fmt"

	"github.com/10gen/mongo-go-driver/yamgo/private/conn"
)

// PLAIN is the mechanism name for PLAIN.
const PLAIN = "PLAIN"

func newPlainAuthenticator(cred *Cred) (Authenticator, error) {
	if cred.Source != "" && cred.Source != "$external" {
		return nil, fmt.Errorf("PLAIN source must be empty or $external")
	}

	return &PlainAuthenticator{
		Username: cred.Username,
		Password: cred.Password,
	}, nil
}

// PlainAuthenticator uses the PLAIN algorithm over SASL to authenticate a connection.
type PlainAuthenticator struct {
	Username string
	Password string
}

// Auth authenticates the connection.
func (a *PlainAuthenticator) Auth(ctx context.Context, c conn.Connection) error {
	return ConductSaslConversation(ctx, c, "$external", &plainSaslClient{
		username: a.Username,
		password: a.Password,
	})
}

type plainSaslClient struct {
	username string
	password string
}

func (c *plainSaslClient) Start() (string, []byte, error) {
	b := []byte("\x00" + c.username + "\x00" + c.password)
	return PLAIN, b, nil
}

func (c *plainSaslClient) Next(challenge []byte) ([]byte, error) {
	return nil, fmt.Errorf("unexpected server challenge")
}

func (c *plainSaslClient) Completed() bool {
	return true
}
