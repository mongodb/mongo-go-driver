package auth

import (
	"context"
	"crypto/md5"
	"fmt"

	"io"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/model"
	"github.com/10gen/mongo-go-driver/msg"
)

// MONGODBCR is the mechanism name for MONGODB-CR.
const MONGODBCR = "MONGODB-CR"

func newMongoDBCRAuthenticator(cred *Cred) (Authenticator, error) {
	return &MongoDBCRAuthenticator{
		DB:       cred.Source,
		Username: cred.Username,
		Password: cred.Password,
	}, nil
}

// MongoDBCRAuthenticator uses the MONGODB-CR algorithm to authenticate a connection.
type MongoDBCRAuthenticator struct {
	DB       string
	Username string
	Password string
}

// Auth authenticates the connection.
func (a *MongoDBCRAuthenticator) Auth(ctx context.Context, c conn.Connection) error {

	// Arbiters cannot be authenticated
	if c.Model().Kind == model.RSArbiter {
		return nil
	}

	db := a.DB
	if db == "" {
		db = defaultAuthDB
	}

	getNonceRequest := msg.NewCommand(
		msg.NextRequestID(),
		db,
		true,
		bson.D{{"getnonce", 1}},
	)
	var getNonceResult struct {
		Nonce string `bson:"nonce"`
	}

	err := conn.ExecuteCommand(ctx, c, getNonceRequest, &getNonceResult)
	if err != nil {
		return newError(err, MONGODBCR)
	}

	authRequest := msg.NewCommand(
		msg.NextRequestID(),
		db,
		true,
		bson.D{
			{"authenticate", 1},
			{"user", a.Username},
			{"nonce", getNonceResult.Nonce},
			{"key", a.createKey(getNonceResult.Nonce)},
		},
	)
	err = conn.ExecuteCommand(ctx, c, authRequest, &bson.D{})
	if err != nil {
		return newError(err, MONGODBCR)
	}

	return nil
}

func (a *MongoDBCRAuthenticator) createKey(nonce string) string {
	h := md5.New()

	io.WriteString(h, nonce)
	io.WriteString(h, a.Username)
	io.WriteString(h, mongoPasswordDigest(a.Username, a.Password))
	return fmt.Sprintf("%x", h.Sum(nil))
}
