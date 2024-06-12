package driver

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
)

const oidcMech = "MONGODB-OIDC"
const tokenResourceProp = "TOKEN_RESOURCE"
const environmentProp = "ENVIRONMENT"
const principalProp = "PRINCIPAL"
const allowedHostsProp = "ALLOWED_HOSTS"
const azureEnvironmentValue = "azure"
const gcpEnvironmentValue = "gcp"
const defaultAuthDB = "admin"

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

// OIDC Sasl. This is almost a verbatim copy of auth/sasl introduced to remove the dependency on auth package
// which causes a circular dependency when attempting to do Reauthentication in driver/operation.go.
// This could be removed with a larger refactor.

func newError(err error, mechanism string) error {
	return fmt.Errorf("error during %s SASL conversation: %w", oidcMech, err)
}

// oidcSaslClient is the client piece of a sasl conversation.
type oidcSaslClient interface {
	Start() (string, []byte, error)
	Next(challenge []byte) ([]byte, error)
	Completed() bool
}

// oidcSaslClientCloser is a oidcSaslClient that has resources to clean up.
type oidcSaslClientCloser interface {
	oidcSaslClient
	Close()
}

// extraOptionsOIDCSaslClient is a SaslClient that appends options to the saslStart command.
type extraOptionsOIDCSaslClient interface {
	StartCommandOptions() bsoncore.Document
}

// saslConversation represents a SASL conversation. This type implements the SpeculativeConversation interface so the
// conversation can be executed in multi-step speculative fashion.
type saslConversation struct {
	client      oidcSaslClient
	source      string
	mechanism   string
	speculative bool
}

func newSaslConversation(client oidcSaslClient, source string, speculative bool) *saslConversation {
	authSource := source
	if authSource == "" {
		authSource = defaultAuthDB
	}
	return &saslConversation{
		client:      client,
		source:      authSource,
		speculative: speculative,
	}
}

// FirstMessage returns the first message to be sent to the server. This message contains a "db" field so it can be used
// for speculative authentication.
func (sc *saslConversation) FirstMessage() (bsoncore.Document, error) {
	var payload []byte
	var err error
	sc.mechanism, payload, err = sc.client.Start()
	if err != nil {
		return nil, err
	}

	saslCmdElements := [][]byte{
		bsoncore.AppendInt32Element(nil, "saslStart", 1),
		bsoncore.AppendStringElement(nil, "mechanism", sc.mechanism),
		bsoncore.AppendBinaryElement(nil, "payload", 0x00, payload),
	}
	if sc.speculative {
		// The "db" field is only appended for speculative auth because the hello command is executed against admin
		// so this is needed to tell the server the user's auth source. For a non-speculative attempt, the SASL commands
		// will be executed against the auth source.
		saslCmdElements = append(saslCmdElements, bsoncore.AppendStringElement(nil, "db", sc.source))
	}
	if extraOptionsClient, ok := sc.client.(extraOptionsOIDCSaslClient); ok {
		optionsDoc := extraOptionsClient.StartCommandOptions()
		saslCmdElements = append(saslCmdElements, bsoncore.AppendDocumentElement(nil, "options", optionsDoc))
	}

	return bsoncore.BuildDocumentFromElements(nil, saslCmdElements...), nil
}

type saslResponse struct {
	ConversationID int    `bson:"conversationId"`
	Code           int    `bson:"code"`
	Done           bool   `bson:"done"`
	Payload        []byte `bson:"payload"`
}

// AuthConfig holds the information necessary to perform an authentication attempt.
// this was moved from the auth package to avoid a circular dependency. The auth package
// reexports this under the old name to avoid breaking the public api.
type AuthConfig struct {
	Description   description.Server
	Connection    Connection
	ClusterClock  *session.ClusterClock
	HandshakeInfo HandshakeInformation
	ServerAPI     *ServerAPIOptions
	HTTPClient    *http.Client
}

// finish completes the conversation based on the first server response to authenticate the given connection.
func (sc *saslConversation) finish(ctx context.Context, cfg *AuthConfig, firstResponse bsoncore.Document) error {
	if closer, ok := sc.client.(oidcSaslClientCloser); ok {
		defer closer.Close()
	}

	var saslResp saslResponse
	err := bson.Unmarshal(firstResponse, &saslResp)
	if err != nil {
		fullErr := fmt.Errorf("unmarshal error: %w", err)
		return newError(fullErr, sc.mechanism)
	}

	cid := saslResp.ConversationID
	var payload []byte
	var result bsoncore.Document
	for {
		if saslResp.Code != 0 {
			return newError(err, sc.mechanism)
		}

		if saslResp.Done && sc.client.Completed() {
			return nil
		}

		payload, err = sc.client.Next(saslResp.Payload)
		if err != nil {
			return newError(err, sc.mechanism)
		}

		if saslResp.Done && sc.client.Completed() {
			return nil
		}

		doc := bsoncore.BuildDocumentFromElements(nil,
			bsoncore.AppendInt32Element(nil, "saslContinue", 1),
			bsoncore.AppendInt32Element(nil, "conversationId", int32(cid)),
			bsoncore.AppendBinaryElement(nil, "payload", 0x00, payload),
		)

		saslOp := Operation{
			CommandFn: func(dst []byte, desc description.SelectedServer) ([]byte, error) {
				return append(dst, doc[4:len(doc)-1]...), nil
			},
			ProcessResponseFn: func(info ResponseInfo) error {
				result = info.ServerResponse
				return nil
			},
			Deployment: Deployment(SingleConnectionDeployment{cfg.Connection}),
			Database:   sc.source,
			Clock:      cfg.ClusterClock,
			ServerAPI:  cfg.ServerAPI,
		}

		err = saslOp.Execute(ctx)
		if err != nil {
			return newError(err, sc.mechanism)
		}
		err = bson.Unmarshal(result, &saslResp)
		if err != nil {
			fullErr := fmt.Errorf("unmarshal error: %w", err)
			return newError(fullErr, sc.mechanism)
		}
		return nil
	}
}

// conductOIDCSaslConversation runs a full SASL conversation to authenticate the given connection.
func conductOIDCSaslConversation(ctx context.Context, cfg *AuthConfig, authSource string, client oidcSaslClient) error {
	// Create a non-speculative SASL conversation.
	conversation := newSaslConversation(client, authSource, false)

	doc, err := conversation.FirstMessage()
	if err != nil {
		return newError(err, conversation.mechanism)
	}
	var result bsoncore.Document
	saslOp := Operation{
		CommandFn: func(dst []byte, desc description.SelectedServer) ([]byte, error) {
			return append(dst, doc[4:len(doc)-1]...), nil
		},
		ProcessResponseFn: func(info ResponseInfo) error {
			result = info.ServerResponse
			return nil
		},
		Deployment: Deployment(SingleConnectionDeployment{cfg.Connection}),
		Database:   authSource,
		Clock:      cfg.ClusterClock,
		ServerAPI:  cfg.ServerAPI,
	}
	if err := saslOp.Execute(ctx); err != nil {
		return newError(err, conversation.mechanism)
	}

	return conversation.finish(ctx, cfg, result)
}
