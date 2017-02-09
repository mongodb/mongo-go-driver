package auth

import (
	"crypto/md5"
	"fmt"

	"io"

	"github.com/10gen/mongo-go-driver/core"
	"github.com/10gen/mongo-go-driver/core/msg"
	"gopkg.in/mgo.v2/bson"
)

// MongoDBCRAuthenticator uses the MONGODB-CR algorithm to authenticate a connection.
type MongoDBCRAuthenticator struct {
	DB       string
	Username string
	Password string
}

// Name returns MONGODB-CR.
func (a *MongoDBCRAuthenticator) Name() string {
	return "MONGODB-CR"
}

// Auth authenticates the connection.
func (a *MongoDBCRAuthenticator) Auth(c core.Connection) error {
	db := a.DB
	if db == "" {
		db = "admin"
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

	err := core.ExecuteCommand(c, getNonceRequest, &getNonceResult)
	if err != nil {
		return newError(err, a.Name())
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
	err = core.ExecuteCommand(c, authRequest, &bson.D{})
	if err != nil {
		return newError(err, a.Name())
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
