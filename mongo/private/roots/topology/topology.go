// Package topology contains types that handles the discovery, monitoring, and selection
// of servers. This package is designed to expose enough inner workings of service discovery
// and monitoring to allow low level applications to have fine grained control, while hiding
// most of the detailed implementation of the algorithms.
package topology

import (
	"context"
	"time"

	"github.com/mongodb/mongo-go-driver/mongo/connstring"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/connection"
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
)

type MonitorMode uint8

const (
	AutomaticMode MonitorMode = iota
	SingleMode
)

// Topology respresents a MongoDB deployment.
type Topology struct{}

func New(...Option) (*Topology, error)              { return nil, nil }
func (*Topology) Close() error                      { return nil }
func (*Topology) Description() description.Topology { return description.Topology{} }
func (*Topology) Subscribe() (*Subscription, error) { return nil, nil }
func (*Topology) RequestImmediateCheck()            { return }
func (*Topology) SelectServer(context.Context, ServerSelector, readpref.ReadPref) (*SelectedServer, error) {
	// It doesn't seem like we also need a separate SelectServer method in this package like the cluster
	// package has. This is especially true since we are trying not to expose monitors.
	return nil, nil
}

// Subscription is a subscription to updates to the description of the Topology that created this
// Subscription.
type Subscription struct {
	C      <-chan description.Topology
	closed bool
}

func (s *Subscription) Unsubscribe() error { return nil }

type Option func(*config) error

func WithConnString(func(connstring.ConnString) connstring.ConnString) Option { return nil }
func WithMode(func(MonitorMode) MonitorMode) Option                           { return nil }
func WithReplicaSetname(func(string) string) Option                           { return nil }
func WithSeedList(func(...string) []string) Option                            { return nil }
func WithServerOptions(func(...ServerOption) []ServerOption) Option           { return nil }

type config struct{}

type ServerSelector func(description.Topology, []description.Server) ([]description.Server, error)

func CompositeSelector([]ServerSelector) ServerSelector { return nil }
func LatencySelector(time.Duration) ServerSelector      { return nil }
func WriteSelector() ServerSelector                     { return nil }

type SelectedServer struct {
	Server

	Kind     description.TopologyKind
	ReadPref *readpref.ReadPref
}

type Server struct{}

func NewServer(connection.Addr, ...ServerOption) (*Server, error)         { return nil, nil }
func (*Server) Close() error                                              { return nil }
func (*Server) Connection(context.Context) (connection.Connection, error) { return nil, nil }
func (*Server) Description() description.Server                           { return description.Server{} }
func (*Server) Subscribe() (*ServerSubscription, error)                   { return nil, nil }
func (*Server) RequestImmediateCheck()                                    { return }

// Drain will drain the connection pool of this server. This is mainly here so the
// pool for the server doesn't need to be directly exposed and so that when an error
// is returned from reading or writing, a client can drain the pool for this server.
// This is exposed here so we don't have to wrap the Connection type and sniff responses
// for errors that would cause the pool to be drained, which can in turn centralize the
// logic for handling errors in the Client type.
func (*Server) Drain() error { return nil }

type ServerSubscription struct {
	C      <-chan description.Server
	closed bool
}

func (ss *ServerSubscription) Unsubscribe() error { return nil }

type monitor struct{}

type serverConfig struct{}

type ServerOption func(*serverConfig) error

func WithConfigurer(func(connection.Configurer) connection.Configurer) ServerOption     { return nil }
func WithConnectionOptions(func(...connection.Option) []connection.Option) ServerOption { return nil }
func WithHeartbeatInterval(func(time.Duration) time.Duration) ServerOption              { return nil }
func WithHeartbeatTimeout(func(time.Duration) time.Duration) ServerOption               { return nil }
func WithMaxConnections(func(uint16) uint16) ServerOption                               { return nil }
func WithMaxIdleConnections(func(uint16) uint16) ServerOption                           { return nil }
