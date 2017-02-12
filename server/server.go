package server

import (
	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/desc"
)

// New creates a new server. Internally, it
// creates a new Monitor with which to monitor the
// state of the server. When the Server is closed,
// the monitor will be stopped.
func New(endpoint desc.Endpoint, opts ...Option) (Server, error) {
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
	Connection() (conn.ConnectionCloser, error)
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

func (s *serverImpl) Connection() (conn.ConnectionCloser, error) {
	return s.monitor.dialer(s.monitor.endpoint, s.monitor.connOpts...)
}
