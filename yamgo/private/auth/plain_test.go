package auth_test

import (
	"context"
	"strings"
	"testing"

	"github.com/10gen/mongo-go-driver/bson"

	"reflect"

	"encoding/base64"

	"github.com/10gen/mongo-go-driver/yamgo/internal/conntest"
	"github.com/10gen/mongo-go-driver/yamgo/internal/msgtest"
	. "github.com/10gen/mongo-go-driver/yamgo/private/auth"
	"github.com/10gen/mongo-go-driver/yamgo/private/msg"
)

func TestPlainAuthenticator_Fails(t *testing.T) {
	t.Parallel()

	authenticator := PlainAuthenticator{
		Username: "user",
		Password: "pencil",
	}

	saslStartReply := msgtest.CreateCommandReply(bson.D{
		bson.NewDocElem("ok", 1),
		bson.NewDocElem("conversationId", 1),
		bson.NewDocElem("payload", []byte{}),
		bson.NewDocElem("code", 143),
		bson.NewDocElem("done", true),
	})

	conn := &conntest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply},
	}

	err := authenticator.Auth(context.Background(), conn)
	if err == nil {
		t.Fatalf("expected an error but got none")
	}

	errPrefix := "unable to authenticate using mechanism \"PLAIN\""
	if !strings.HasPrefix(err.Error(), errPrefix) {
		t.Fatalf("expected an err starting with \"%s\" but got \"%s\"", errPrefix, err)
	}
}

func TestPlainAuthenticator_Extra_server_message(t *testing.T) {
	t.Parallel()

	authenticator := PlainAuthenticator{
		Username: "user",
		Password: "pencil",
	}

	saslStartReply := msgtest.CreateCommandReply(bson.D{
		bson.NewDocElem("ok", 1),
		bson.NewDocElem("conversationId", 1),
		bson.NewDocElem("payload", []byte{}),
		bson.NewDocElem("done", false),
	})
	saslContinueReply := msgtest.CreateCommandReply(bson.D{
		bson.NewDocElem("ok", 1),
		bson.NewDocElem("conversationId", 1),
		bson.NewDocElem("payload", []byte{}),
		bson.NewDocElem("done", true),
	})

	conn := &conntest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply, saslContinueReply},
	}

	err := authenticator.Auth(context.Background(), conn)
	if err == nil {
		t.Fatalf("expected an error but got none")
	}

	errPrefix := "unable to authenticate using mechanism \"PLAIN\": unexpected server challenge"
	if !strings.HasPrefix(err.Error(), errPrefix) {
		t.Fatalf("expected an err starting with \"%s\" but got \"%s\"", errPrefix, err)
	}
}

func TestPlainAuthenticator_Succeeds(t *testing.T) {
	t.Parallel()

	authenticator := PlainAuthenticator{
		Username: "user",
		Password: "pencil",
	}

	saslStartReply := msgtest.CreateCommandReply(bson.D{
		bson.NewDocElem("ok", 1),
		bson.NewDocElem("conversationId", 1),
		bson.NewDocElem("payload", []byte{}),
		bson.NewDocElem("done", true),
	})

	conn := &conntest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply},
	}

	err := authenticator.Auth(context.Background(), conn)
	if err != nil {
		t.Fatalf("expected no error but got \"%s\"", err)
	}

	if len(conn.Sent) != 1 {
		t.Fatalf("expected 1 messages to be sent but had %d", len(conn.Sent))
	}

	saslStartRequest := conn.Sent[0].(*msg.Query)
	payload, _ := base64.StdEncoding.DecodeString("AHVzZXIAcGVuY2ls")
	expectedCmd := bson.D{
		bson.NewDocElem("saslStart", 1),
		bson.NewDocElem("mechanism", "PLAIN"),
		bson.NewDocElem("payload", payload),
	}
	if !reflect.DeepEqual(saslStartRequest.Query, expectedCmd) {
		t.Fatalf("saslStart command was incorrect: %v", saslStartRequest.Query)
	}
}
