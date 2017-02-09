package auth_test

import (
	"testing"

	"gopkg.in/mgo.v2/bson"

	"reflect"

	"strings"

	. "github.com/10gen/mongo-go-driver/auth"
	"github.com/10gen/mongo-go-driver/core/msg"
)

func TestMongoDBCRAuthenticator_Fails(t *testing.T) {
	t.Parallel()

	authenticator := MongoDBCRAuthenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
	}

	getNonceReply := createCommandReply(bson.D{
		{"ok", 1},
		{"nonce", "2375531c32080ae8"},
	})

	authenticateReply := createCommandReply(bson.D{{"ok", 0}})

	conn := &mockConnection{
		responseQ: []*msg.Reply{getNonceReply, authenticateReply},
	}

	err := authenticator.Auth(conn)
	if err == nil {
		t.Fatalf("expected an error but got none")
	}

	errPrefix := "unable to authenticate using mechanism \"MONGODB-CR\""
	if !strings.HasPrefix(err.Error(), errPrefix) {
		t.Fatalf("expected an err starting with \"%s\" but got \"%s\"", errPrefix, err)
	}
}

func TestMongoDBCRAuthenticator_Succeeds(t *testing.T) {
	t.Parallel()

	authenticator := MongoDBCRAuthenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
	}

	getNonceReply := createCommandReply(bson.D{
		{"ok", 1},
		{"nonce", "2375531c32080ae8"},
	})

	authenticateReply := createCommandReply(bson.D{{"ok", 1}})

	conn := &mockConnection{
		responseQ: []*msg.Reply{getNonceReply, authenticateReply},
	}

	err := authenticator.Auth(conn)
	if err != nil {
		t.Fatalf("expected no error but got \"%s\"", err)
	}

	if len(conn.sent) != 2 {
		t.Fatalf("expected 2 messages to be sent but had %d", len(conn.sent))
	}

	getNonceRequest := conn.sent[0].(*msg.Query)
	if !reflect.DeepEqual(getNonceRequest.Query, bson.D{{"getnonce", 1}}) {
		t.Fatalf("getnonce command was incorrect: %v", getNonceRequest.Query)
	}

	authenticateRequest := conn.sent[1].(*msg.Query)
	expectedAuthenticateDoc := bson.D{
		{"authenticate", 1},
		{"user", "user"},
		{"nonce", "2375531c32080ae8"},
		{"key", "21742f26431831d5cfca035a08c5bdf6"},
	}

	if !reflect.DeepEqual(authenticateRequest.Query, expectedAuthenticateDoc) {
		t.Fatalf("authenticate command was incorrect: %v", authenticateRequest.Query)
	}
}
