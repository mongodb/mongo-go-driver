package auth_test

import (
	"strings"
	"testing"

	"gopkg.in/mgo.v2/bson"

	"reflect"

	"encoding/base64"

	. "github.com/10gen/mongo-go-driver/auth"
	"github.com/10gen/mongo-go-driver/core/msg"
)

func TestPlainAuthenticator_Fails(t *testing.T) {
	t.Parallel()

	authenticator := PlainAuthenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
	}

	saslStartReply := createCommandReply(bson.D{
		{"ok", 1},
		{"conversationId", 1},
		{"payload", []byte{}},
		{"code", 143},
		{"done", true},
	})

	conn := &mockConnection{
		responseQ: []*msg.Reply{saslStartReply},
	}

	err := authenticator.Auth(conn)
	if err == nil {
		t.Fatalf("expected an error but got none")
	}

	errPrefix := "unable to authenticate using mechanism \"PLAIN\""
	if !strings.HasPrefix(err.Error(), errPrefix) {
		t.Fatalf("expected an err starting with \"%s\" but got \"%s\"", errPrefix, err)
	}
}

func TestPlainAuthenticator_Succeeds(t *testing.T) {
	t.Parallel()

	authenticator := PlainAuthenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
	}

	saslStartReply := createCommandReply(bson.D{
		{"ok", 1},
		{"conversationId", 1},
		{"payload", []byte{}},
		{"done", true},
	})

	conn := &mockConnection{
		responseQ: []*msg.Reply{saslStartReply},
	}

	err := authenticator.Auth(conn)
	if err != nil {
		t.Fatalf("expected no error but got \"%s\"", err)
	}

	if len(conn.sent) != 1 {
		t.Fatalf("expected 1 messages to be sent but had %d", len(conn.sent))
	}

	saslStartRequest := conn.sent[0].(*msg.Query)
	payload, _ := base64.StdEncoding.DecodeString("AHVzZXIAcGVuY2ls")
	expectedCmd := bson.D{
		{"saslStart", 1},
		{"mechanism", "PLAIN"},
		{"payload", payload},
	}
	if !reflect.DeepEqual(saslStartRequest.Query, expectedCmd) {
		t.Fatalf("saslStart command was incorrect: %v", saslStartRequest.Query)
	}
}
