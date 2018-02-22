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
	"github.com/mongodb/mongo-go-driver/mongo/private/roots/result"
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
func (*Topology) Description() Description          { return Description{} }
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
	C      <-chan Description
	closed bool
}

func (s *Subscription) Unsubscribe() error { return nil }

type fsm struct{}

// apply should operate on immutable TopologyDescriptions and Descriptions. This way we don't have to
// lock for the entire time we're applying server description.
func (*fsm) apply(Description, ServerDescription) (Description, error) { return Description{}, nil }

type Option func(*config) error

func WithConnString(func(connstring.ConnString) connstring.ConnString) Option { return nil }
func WithMode(func(MonitorMode) MonitorMode) Option                           { return nil }
func WithReplicaSetname(func(string) string) Option                           { return nil }
func WithSeedList(func(...string) []string) Option                            { return nil }
func WithServerOptions(func(...ServerOption) []ServerOption) Option           { return nil }

type config struct{}

type Description struct{}

func (Description) Server(connection.Addr) (ServerDescription, bool) {
	return ServerDescription{}, false
}

type DescriptionDiff struct{}

func DiffDescription(Description, Description) DescriptionDiff { return DescriptionDiff{} }

type Kind uint32

const (
	Single                Kind = 1
	ReplicaSet            Kind = 2
	ReplicaSetNoPrimary   Kind = 4 + ReplicaSet
	ReplicaSetWithPrimary Kind = 8 + ReplicaSet
	Sharded               Kind = 256
)

type ServerDescription struct{}

func NewServerDescription(connection.Addr, result.IsMaster, result.BuildInfo) ServerDescription {
	return ServerDescription{}
}

func (ServerDescription) setAverageRTT(time.Duration) ServerDescription { return ServerDescription{} }

type ServerKind uint32

const (
	Standalone  ServerKind = 1
	RSMember    ServerKind = 2
	RSPrimary   ServerKind = 4 + RSMember
	RSSecondary ServerKind = 8 + RSMember
	RSArbiter   ServerKind = 16 + RSMember
	RSGhost     ServerKind = 32 + RSMember
	Mongos      ServerKind = 256
)

type ServerSelector func(Description, []ServerDescription) ([]ServerDescription, error)

func CompositeSelector([]ServerSelector) ServerSelector { return nil }
func LatencySelector(time.Duration) ServerSelector      { return nil }
func WriteSelector() ServerSelector                     { return nil }

type SelectedServer Server

func (*SelectedServer) TopologyDescription() Description   { return Description{} }
func (*SelectedServer) ReadPreference() *readpref.ReadPref { return nil }

type Server struct{}

func NewServer(connection.Addr, ...ServerOption) (*Server, error)         { return nil, nil }
func (*Server) Close() error                                              { return nil }
func (*Server) Connection(context.Context) (connection.Connection, error) { return nil, nil }
func (*Server) Description() ServerDescription                            { return ServerDescription{} }
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
	C      <-chan ServerDescription
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
