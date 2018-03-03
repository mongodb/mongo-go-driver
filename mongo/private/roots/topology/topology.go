// Package topology contains types that handles the discovery, monitoring, and selection
// of servers. This package is designed to expose enough inner workings of service discovery
// and monitoring to allow low level applications to have fine grained control, while hiding
// most of the detailed implementation of the algorithms.
package topology

import (
	"context"
	"time"

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

type ServerSelector func(description.Topology, []description.Server) ([]description.Server, error)

func CompositeSelector([]ServerSelector) ServerSelector { return nil }
func LatencySelector(time.Duration) ServerSelector      { return nil }
func WriteSelector() ServerSelector                     { return nil }

type SelectedServer struct {
	Server

	Kind     description.TopologyKind
	ReadPref *readpref.ReadPref
}

type monitor struct{}
