package auth_test

import (
	"strings"
	"testing"

	"gopkg.in/mgo.v2/bson"

	"reflect"

	"encoding/base64"

	. "github.com/10gen/mongo-go-driver/auth"
	"github.com/10gen/mongo-go-driver/core/msg"
	"github.com/10gen/mongo-go-driver/internal/internaltest"
)

func TestScramSHA1Authenticator_Fails(t *testing.T) {
	t.Parallel()

	authenticator := ScramSHA1Authenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
	}

	saslStartReply := internaltest.CreateCommandReply(bson.D{
		{"ok", 1},
		{"conversationId", 1},
		{"payload", []byte{}},
		{"code", 143},
		{"done", true},
	})

	conn := &internaltest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply},
	}

	err := authenticator.Auth(conn)
	if err == nil {
		t.Fatalf("expected an error but got none")
	}

	errPrefix := "unable to authenticate using mechanism \"SCRAM-SHA-1\""
	if !strings.HasPrefix(err.Error(), errPrefix) {
		t.Fatalf("expected an err starting with \"%s\" but got \"%s\"", errPrefix, err)
	}
}

func TestScramSHA1Authenticator_Invalid_server_nonce(t *testing.T) {
	t.Parallel()

	authenticator := ScramSHA1Authenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
		NonceGenerator: func(dst []byte) error {
			copy(dst, []byte("fyko+d2lbbFgONRv9qkxdawL"))
			return nil
		},
	}

	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvLWQybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPTEwMDAw")
	saslStartReply := internaltest.CreateCommandReply(bson.D{
		{"ok", 1},
		{"conversationId", 1},
		{"payload", payload},
		{"done", false},
	})

	conn := &internaltest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply},
	}

	err := authenticator.Auth(conn)
	if err == nil {
		t.Fatalf("expected an error but got none")
	}

	errPrefix := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid server nonce"
	if !strings.HasPrefix(err.Error(), errPrefix) {
		t.Fatalf("expected an err starting with \"%s\" but got \"%s\"", errPrefix, err)
	}
}

func TestScramSHA1Authenticator_Invalid_server_signature(t *testing.T) {
	t.Parallel()

	authenticator := ScramSHA1Authenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
		NonceGenerator: func(dst []byte) error {
			copy(dst, []byte("fyko+d2lbbFgONRv9qkxdawL"))
			return nil
		},
	}

	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPTEwMDAw")
	saslStartReply := internaltest.CreateCommandReply(bson.D{
		{"ok", 1},
		{"conversationId", 1},
		{"payload", payload},
		{"done", false},
	})
	payload, _ = base64.StdEncoding.DecodeString("dj1VTVdlSTI1SkQxeU5ZWlJNcFo0Vkh2aFo5ZTBh")
	saslContinueReply := internaltest.CreateCommandReply(bson.D{
		{"ok", 1},
		{"conversationId", 1},
		{"payload", payload},
		{"done", false},
	})

	conn := &internaltest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply, saslContinueReply},
	}

	err := authenticator.Auth(conn)
	if err == nil {
		t.Fatalf("expected an error but got none")
	}

	errPrefix := "unable to authenticate using mechanism \"SCRAM-SHA-1\": invalid server signature"
	if !strings.HasPrefix(err.Error(), errPrefix) {
		t.Fatalf("expected an err starting with \"%s\" but got \"%s\"", errPrefix, err)
	}
}

func TestScramSHA1Authenticator_Succeeds(t *testing.T) {
	t.Parallel()

	authenticator := ScramSHA1Authenticator{
		DB:       "source",
		Username: "user",
		Password: "pencil",
		NonceGenerator: func(dst []byte) error {
			copy(dst, []byte("fyko+d2lbbFgONRv9qkxdawL"))
			return nil
		},
	}

	payload, _ := base64.StdEncoding.DecodeString("cj1meWtvK2QybGJiRmdPTlJ2OXFreGRhd0xIbytWZ2s3cXZVT0tVd3VXTElXZzRsLzlTcmFHTUhFRSxzPXJROVpZM01udEJldVAzRTFURFZDNHc9PSxpPTEwMDAw")
	saslStartReply := internaltest.CreateCommandReply(bson.D{
		{"ok", 1},
		{"conversationId", 1},
		{"payload", payload},
		{"done", false},
	})
	payload, _ = base64.StdEncoding.DecodeString("dj1VTVdlSTI1SkQxeU5ZWlJNcFo0Vkh2aFo5ZTA9")
	saslContinueReply := internaltest.CreateCommandReply(bson.D{
		{"ok", 1},
		{"conversationId", 1},
		{"payload", payload},
		{"done", true},
	})

	conn := &internaltest.MockConnection{
		ResponseQ: []*msg.Reply{saslStartReply, saslContinueReply},
	}

	err := authenticator.Auth(conn)
	if err != nil {
		t.Fatalf("expected no error but got \"%s\"", err)
	}

	if len(conn.Sent) != 2 {
		t.Fatalf("expected 2 messages to be sent but had %d", len(conn.Sent))
	}

	saslStartRequest := conn.Sent[0].(*msg.Query)
	payload, _ = base64.RawStdEncoding.DecodeString("biwsbj11c2VyLHI9ZnlrbytkMmxiYkZnT05Sdjlxa3hkYXdM")
	expectedCmd := bson.D{
		{"saslStart", 1},
		{"mechanism", "SCRAM-SHA-1"},
		{"payload", payload},
	}
	if !reflect.DeepEqual(saslStartRequest.Query, expectedCmd) {
		t.Fatalf("saslStart command was incorrect:\n  expected: %v\n    actual: %v", expectedCmd, saslStartRequest.Query)
	}

	saslContinueRequest := conn.Sent[1].(*msg.Query)
	payload, _ = base64.RawStdEncoding.DecodeString("Yz1iaXdzLHI9ZnlrbytkMmxiYkZnT05Sdjlxa3hkYXdMSG8rVmdrN3F2VU9LVXd1V0xJV2c0bC85U3JhR01IRUUscD1NQzJUOEJ2Ym1XUmNrRHc4b1dsNUlWZ2h3Q1k9")
	expectedCmd = bson.D{
		{"saslContinue", 1},
		{"conversationId", 1},
		{"payload", payload},
	}
	if !reflect.DeepEqual(saslContinueRequest.Query, expectedCmd) {
		t.Fatalf("saslContinue command was incorrect:\n  expected: %v\n    actual: %v", expectedCmd, saslContinueRequest.Query)
	}
}
