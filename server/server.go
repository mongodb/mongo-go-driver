package server

import (
	"context"

	"github.com/10gen/mongo-go-driver/conn"
)

// New creates a new server. Internally, it
// creates a new Monitor with which to monitor the
// state of the server. When the Server is closed,
// the monitor will be stopped.
func New(endpoint conn.Endpoint, opts ...Option) (Server, error) {
	monitor, err := StartMonitor(endpoint, opts...)
	if err != nil {
		return nil, err
	}

	return &serverImpl{
		monitor:     monitor,
		ownsMonitor: true,
	}, nil
}

// NewWithMonitor creates a new Server from
// an existing monitor. When the server is closed,
// the monitor will not be stopped.
func NewWithMonitor(monitor *Monitor) Server {
	return &serverImpl{
		monitor: monitor,
	}
}

// Server represents a server.
type Server interface {
	// Closes closes the server.
	Close()
	// Connection gets a connection to the server.
	Connection(context.Context) (conn.Connection, error)
}

type serverImpl struct {
	monitor     *Monitor
	ownsMonitor bool
}

func (s *serverImpl) Close() {
	if s.ownsMonitor {
		s.monitor.Stop()
	}
}

func (s *serverImpl) Connection(ctx context.Context) (conn.Connection, error) {
	return s.monitor.connDialer(ctx, s.monitor.endpoint, s.monitor.connOpts...)
}
