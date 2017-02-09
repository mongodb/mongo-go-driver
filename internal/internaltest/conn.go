package internaltest

import (
	"fmt"

	"github.com/10gen/mongo-go-driver/core"
	"github.com/10gen/mongo-go-driver/core/msg"
)

type MockConnection struct {
	Sent      []msg.Request
	ResponseQ []*msg.Reply
	WriteErr  error

	SkipResponseToFixup bool
}

func (c *MockConnection) Desc() *core.ConnectionDesc {
	return &core.ConnectionDesc{}
}

func (c *MockConnection) Read() (msg.Response, error) {
	if len(c.ResponseQ) == 0 {
		return nil, fmt.Errorf("no response queued")
	}
	resp := c.ResponseQ[0]
	c.ResponseQ = c.ResponseQ[1:]
	return resp, nil
}

func (c *MockConnection) Write(reqs ...msg.Request) error {
	if c.WriteErr != nil {
		err := c.WriteErr
		c.WriteErr = nil
		return err
	}

	for i, req := range reqs {
		c.Sent = append(c.Sent, req)
		if !c.SkipResponseToFixup && i < len(c.ResponseQ) {
			c.ResponseQ[i].RespTo = req.RequestID()
		}
	}
	return nil
}
