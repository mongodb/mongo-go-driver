package driver

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

const oidcMech = "MONGODB-OIDC"
const tokenResourceProp = "TOKEN_RESOURCE"
const environmentProp = "ENVIRONMENT"
const principalProp = "PRINCIPAL"
const allowedHostsProp = "ALLOWED_HOSTS"
const azureEnvironmentValue = "azure"
const gcpEnvironmentValue = "gcp"

// OIDCAuthenticator is synchronized and handles caching of the access token, refreshToken,
// and IDPInfo. It also provides a mechanism to refresh the access token, but this functionality
// is only for the OIDC Human flow.
type OIDCAuthenticator struct {
	mu sync.Mutex // Guards all of the info in the OIDCAuthenticator struct.

	AuthMechanismProperties map[string]string

	accessToken  string
	refreshToken *string
	idpInfo      *IDPInfo
}

func NewOIDCAuthenticator() *OIDCAuthenticator {
	return &OIDCAuthenticator{
		AuthMechanismProperties: make(map[string]string),
	}
}

type IDPInfo struct {
	Issuer        string   `bson:"issuer"`
	ClientID      string   `bson:"clientId"`
	RequestScopes []string `bson:"requestScopes"`
}

type OIDCCallback func(context.Context, *OIDCArgs) (*OIDCCredential, error)

type OIDCArgs struct {
	Version      int
	IDPInfo      *IDPInfo
	RefreshToken *string
}

type OIDCCredential struct {
	AccessToken  string
	ExpiresAt    time.Time
	RefreshToken *string
}

type oidcOneStep struct {
	accessToken string
}

func (oos *oidcOneStep) Start() (string, []byte, error) {
	return oidcMech, jwtStepRequest(oos.accessToken), nil
}

func newAuthError(msg string, err error) error {
	return fmt.Errorf("authentication error: %s: %w", msg, err)
}

func (oos *oidcOneStep) Next(context.Context, []byte) ([]byte, error) {
	return nil, newAuthError("unexpected step in OIDC machine authentication", nil)
}

func (*oidcOneStep) Completed() bool {
	return true
}

func jwtStepRequest(accessToken string) []byte {
	return bsoncore.NewDocumentBuilder().
		AppendString("jwt", accessToken).
		Build()
}

func principalStepRequest(principal string) []byte {
	doc := bsoncore.NewDocumentBuilder()
	if principal != "" {
		doc.AppendString("n", principal)
	}
	return doc.Build()
}
