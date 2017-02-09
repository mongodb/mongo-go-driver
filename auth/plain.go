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

// Auth authenticates the connection.
func (a *PlainAuthenticator) Auth(c core.Connection) error {
	return conductSaslConversation(c, a.DB, &plainSaslClient{
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
	return plain, b, nil
}

func (c *plainSaslClient) Next(challenge []byte) ([]byte, error) {
	return nil, fmt.Errorf("unexpected server challenge")
}

func (c *plainSaslClient) Completed() bool {
	return true
}
