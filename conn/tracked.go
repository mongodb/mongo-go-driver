package conn

import (
	"context"

	"github.com/10gen/mongo-go-driver/msg"
)

// Tracked creates a tracked connection.
func Tracked(c Connection) *TrackedConnection {
	return &TrackedConnection{
		c:     c,
		usage: 1,
	}
}

// TrackedConnection is a connection that only closes
// once it's usage count is 0.
type TrackedConnection struct {
	c     Connection
	usage int
}

// Alive indicates if the connection is still alive.
func (tc *TrackedConnection) Alive() bool {
	return tc.c.Alive()
}

// Close closes the connection.
func (tc *TrackedConnection) Close() error {
	tc.usage--
	if tc.usage <= 0 {
		return tc.c.Close()
	}

	return nil
}

// Desc gets a description of the connection.
func (tc *TrackedConnection) Desc() *Desc {
	return tc.c.Desc()
}

// Expired indicates if the connection has expired.
func (tc *TrackedConnection) Expired() bool {
	return tc.c.Expired()
}

// Inc increments the usage count.
func (tc *TrackedConnection) Inc() {
	tc.usage++
}

// Read reads a message from the connection.
func (tc *TrackedConnection) Read(ctx context.Context, responseTo int32) (msg.Response, error) {
	return tc.c.Read(ctx, responseTo)
}

// Write writes a number of messages to the connection.
func (tc *TrackedConnection) Write(ctx context.Context, reqs ...msg.Request) error {
	return tc.c.Write(ctx, reqs...)
}
