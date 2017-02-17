package server

import (
	"context"

	"github.com/10gen/mongo-go-driver/conn"
)

// New creates a new server. Internally, it
// creates a new Monitor with which to monitor the
// state of the server. When the Server is closed,
// the monitor will be stopped.
func New(endpoint conn.Endpoint, opts ...Option) (*Server, error) {
	monitor, err := StartMonitor(endpoint, opts...)
	if err != nil {
		return nil, err
	}

	return &Server{
		cfg:         monitor.cfg,
		monitor:     monitor,
		ownsMonitor: true,
	}, nil
}

// NewWithMonitor creates a new Server from
// an existing monitor. When the server is closed,
// the monitor will not be stopped. Any unspecified
// options will have their default value pulled from the monitor.
// Any monitor specific options will be ignored.
func NewWithMonitor(monitor *Monitor, opts ...Option) *Server {
	return &Server{
		cfg:     monitor.cfg.reconfig(opts...),
		monitor: monitor,
	}
}

// Server is a logical connection to a server.
type Server struct {
	cfg         *config
	monitor     *Monitor
	ownsMonitor bool
}

// Close closes the server.
func (s *Server) Close() {
	if s.ownsMonitor {
		s.monitor.Stop()
	}
}

// Connection gets a connection to the server.
func (s *Server) Connection(ctx context.Context) (conn.Connection, error) {
	conn, err := s.cfg.dialer(ctx, s.monitor.Endpoint(), s.cfg.connOpts...)
	if err != nil {
		return nil, err
	}

	return &serverConn{
		Connection: conn,
		server:     s,
	}, nil
}

// Desc gets a description of the server as of the last heartbeat.
func (s *Server) Desc() *Desc {
	s.monitor.descLock.Lock()
	desc := s.monitor.desc
	s.monitor.descLock.Unlock()
	return desc
}

func (s *Server) connClosed(conn *serverConn) {
	conn.Connection.Close()
}
