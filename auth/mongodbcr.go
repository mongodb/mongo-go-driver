package auth

import (
	"context"
	"crypto/md5"
	"fmt"

	"io"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/msg"
	"gopkg.in/mgo.v2/bson"
)

const mongodbCR = "MONGODB-CR"

func newMongoDBCRAuthenticator(db, username, password string, props map[string]string) (Authenticator, error) {
	return &MongoDBCRAuthenticator{
		DB:       db,
		Username: username,
		Password: password,
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
		return newError(err, mongodbCR)
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
		return newError(err, mongodbCR)
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
