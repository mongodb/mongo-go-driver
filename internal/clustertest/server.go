package clustertest

import (
	"context"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/server"
)

func NewFakeServer(endpoint conn.Endpoint) *FakeServer {
	return &FakeServer{endpoint}
}

type FakeServer struct {
	endpoint conn.Endpoint
}

func (f *FakeServer) Connection(context.Context) (conn.Connection, error) {
	panic("not implemented")
}

func (f *FakeServer) Desc() *server.Desc {
	return &server.Desc{
		Endpoint: f.endpoint,
	}
}

func (f *FakeServer) Close() {
	// no-op
}
