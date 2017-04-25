//+build gssapi
//+build windows linux darwin

package auth

import (
	"context"
	"fmt"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal/auth/gssapi"
)

// GSSAPI is the mechanism name for GSSAPI.
const GSSAPI = "GSSAPI"

func newGSSAPIAuthenticator(cred *Cred) (Authenticator, error) {
	if cred.Source != "" && cred.Source != "$external" {
		return nil, fmt.Errorf("GSSAPI source must be empty or $external")
	}

	return &GSSAPIAuthenticator{
		Username:    cred.Username,
		Password:    cred.Password,
		PasswordSet: cred.PasswordSet,
		Props:       cred.Props,
	}, nil
}

// GSSAPIAuthenticator uses the GSSAPI algorithm over SASL to authenticate a connection.
type GSSAPIAuthenticator struct {
	Username    string
	Password    string
	PasswordSet bool
	Props       map[string]string
}

// Auth authenticates the connection.
func (a *GSSAPIAuthenticator) Auth(ctx context.Context, c conn.Connection) error {
	client, err := gssapi.New(c.Model().Addr.String(), a.Username, a.Password, a.PasswordSet, a.Props)

	if err != nil {
		return err
	}
	return conductSaslConversation(ctx, c, "$external", client)
}
