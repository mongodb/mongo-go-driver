package auth_test

import (
	"bytes"
	"fmt"

	"github.com/10gen/mongo-go-driver/core"
	"github.com/10gen/mongo-go-driver/core/msg"
	"gopkg.in/mgo.v2/bson"
)

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
