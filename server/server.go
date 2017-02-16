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
		monitor:     monitor,
		ownsMonitor: true,
	}, nil
}

// NewWithMonitor creates a new Server from
// an existing monitor. When the server is closed,
// the monitor will not be stopped.
func NewWithMonitor(monitor *Monitor) *Server {
	return &Server{
		monitor: monitor,
	}
}

// Server is a logical connection to a server.
type Server struct {
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
	conn, err := s.monitor.connDialer(ctx, s.monitor.endpoint, s.monitor.connOpts...)
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

// RequestImmediateCheck will cause the Monitor to send
// a heartbeat to the server right away, instead of waiting for
// the heartbeat timeout.
func (s *Server) RequestImmediateCheck() {
	s.monitor.RequestImmediateCheck()
}

// Subscribe returns a channel on which all updated server descriptions
// will be sent. The channel will have a buffer size of one, and
// will be pre-populated with the current description.
// Subscribe also returns a function that, when called, will close
// the subscription channel and remove it from the list of subscriptions.
func (s *Server) Subscribe() (<-chan *Desc, func(), error) {
	return s.monitor.Subscribe()
}

func (s *Server) connClosed(conn *serverConn) {
	conn.Connection.Close()
}
