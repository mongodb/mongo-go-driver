package auth_test

import (
	"bytes"
	"fmt"
	"testing"

	"gopkg.in/mgo.v2/bson"

	"reflect"

	"strings"

	. "github.com/10gen/mongo-go-driver/auth"
	"github.com/10gen/mongo-go-driver/core"
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

	errPrefix := "unable to authenticate \"user\" on database \"source\" using \"MONGODB-CR\""
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

type mockConnection struct {
	sent      []msg.Request
	responseQ []*msg.Reply
	writeErr  error
}

func (c *mockConnection) Desc() *core.ConnectionDesc {
	return &core.ConnectionDesc{}
}

func (c *mockConnection) Read() (msg.Response, error) {
	if len(c.responseQ) == 0 {
		return nil, fmt.Errorf("no response queued")
	}
	resp := c.responseQ[0]
	c.responseQ = c.responseQ[1:]
	return resp, nil
}

func (c *mockConnection) Write(reqs ...msg.Request) error {
	for i, req := range reqs {
		c.responseQ[i].RespTo = req.RequestID()
		c.sent = append(c.sent, req)
	}
	return c.writeErr
}

func createCommandReply(in interface{}) *msg.Reply {
	doc, _ := bson.Marshal(in)
	reply := &msg.Reply{
		NumberReturned: 1,
		DocumentsBytes: doc,
	}

	// encode it, then decode it to handle the internal workings of msg.Reply
	codec := msg.NewWireProtocolCodec()
	var b bytes.Buffer
	err := codec.Encode(&b, reply)
	if err != nil {
		panic(err)
	}
	resp, err := codec.Decode(&b)
	if err != nil {
		panic(err)
	}

	return resp.(*msg.Reply)
}
