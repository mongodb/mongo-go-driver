package auth

import (
	"fmt"

	"github.com/10gen/mongo-go-driver/core"
)

const plain = "PLAIN"

// PlainAuthenticator uses the PLAIN algorithm over SASL to authenticate a connection.
type PlainAuthenticator struct {
	DB       string
	Username string
	Password string
}

// Name returns PLAIN.
func (a *PlainAuthenticator) Name() string {
	return plain
}

// Auth authenticates the connection.
func (a *PlainAuthenticator) Auth(c core.Connection) error {
	return conductSaslConversation(c, a.DB, &plainSaslClient{
		Username: a.Username,
		Password: a.Password,
	})
}

type plainSaslClient struct {
	Username string
	Password string
}

func (c *plainSaslClient) Start() (string, []byte, error) {
	b := []byte("\x00" + c.Username + "\x00" + c.Password)
	return plain, b, nil
}

func (c *plainSaslClient) Next(challenge []byte) ([]byte, error) {
	return nil, fmt.Errorf("unexpected server challenge")
}
