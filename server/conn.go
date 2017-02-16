package server

import (
	"context"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/msg"
)

type serverConn struct {
	server *Server
	conn.Connection
}

// Alive indicates if the connection is still alive.
func (c *serverConn) Alive() bool {
	return c.Connection.Alive()
}

// Close closes the connection.
func (c *serverConn) Close() error {
	c.server.connClosed(c)
	return nil
}

// Desc gets a description of the connection.
func (c *serverConn) Desc() *conn.Desc {
	return c.Connection.Desc()
}

// Expired indicates if the connection has expired.
func (c *serverConn) Expired() bool {
	return c.Connection.Expired()
}

// Read reads a message from the connection.
func (c *serverConn) Read(ctx context.Context) (msg.Response, error) {
	resp, err := c.Connection.Read(ctx)
	if err != nil {
		c.server.monitor.RequestImmediateCheck()
	}
	return resp, err
}

// Write writes a number of messages to the connection.
func (c *serverConn) Write(ctx context.Context, reqs ...msg.Request) error {
	err := c.Connection.Write(ctx, reqs...)
	if err != nil {
		c.server.monitor.RequestImmediateCheck()
	}

	return err
}
