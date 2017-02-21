package cluster

import (
	"context"
	"errors"
	"math/rand"
	"sync"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal"
	"github.com/10gen/mongo-go-driver/server"
)

// ErrClusterClosed occurs on an attempt to use a closed
// cluster.
var ErrClusterClosed = errors.New("cluster is closed")

// New creates a new cluster. Internally, it
// creates a new Monitor with which to monitor the
// state of the cluster. When the Cluster is closed,
// the monitor will be stopped.
func New(opts ...Option) (*Cluster, error) {
	monitor, err := StartMonitor(opts...)
	if err != nil {
		return nil, err
	}

	cluster := NewWithMonitor(monitor, opts...)
	cluster.ownsMonitor = true
	return cluster, nil
}

// NewWithMonitor creates a new Cluster from
// an existing monitor. When the cluster is closed,
// the monitor will not be stopped. Any unspecified
// options will have their default value pulled from the monitor.
// Any monitor specific options will be ignored.
func NewWithMonitor(monitor *Monitor, opts ...Option) *Cluster {
	cluster := &Cluster{
		cfg:          monitor.cfg.reconfig(opts...),
		stateDesc:    &Desc{},
		stateServers: make(map[conn.Endpoint]*server.Server),
		monitor:      monitor,
	}

	updates, _, _ := monitor.Subscribe()
	go func() {
		for desc := range updates {
			cluster.applyUpdate(desc)
		}
	}()

	return cluster
}

// ServerSelector is a function that selects a server.
type ServerSelector func(*Desc, []*server.Desc) ([]*server.Desc, error)

// Cluster represents a logical connection to a cluster.
type Cluster struct {
	cfg *config

	monitor      *Monitor
	ownsMonitor  bool
	stateDesc    *Desc
	stateLock    sync.Mutex
	stateServers map[conn.Endpoint]*server.Server
}

// Close closes the cluster.
func (c *Cluster) Close() {
	if c.ownsMonitor {
		c.monitor.Stop()
	}

	c.stateLock.Lock()
	defer c.stateLock.Unlock()

	if c.stateServers == nil {
		return
	}

	for _, server := range c.stateServers {
		server.Close()
	}
	c.stateServers = nil
	c.stateDesc = &Desc{}
}

// Desc gets a description of the cluster.
func (c *Cluster) Desc() *Desc {
	c.stateLock.Lock()
	desc := c.stateDesc
	c.stateLock.Unlock()
	return desc
}

// SelectServer selects a server given a selector.
// SelectServer complies with the server selection spec, and will time
// out after serverSelectionTimeout or when the parent context is done.
func (c *Cluster) SelectServer(ctx context.Context, selector ServerSelector) (Server, error) {
	timeout, _ := context.WithTimeout(ctx, c.cfg.serverSelectionTimeout)
	for {
		suitable, err := SelectServers(timeout, c.monitor, selector)
		if err != nil {
			return nil, err
		}

		selected := suitable[rand.Intn(len(suitable))]

		c.stateLock.Lock()
		if c.stateServers == nil {
			c.stateLock.Unlock()
			return nil, ErrClusterClosed
		}
		if server, ok := c.stateServers[selected.Endpoint]; ok {
			c.stateLock.Unlock()
			return server, nil
		}
		c.stateLock.Unlock()

		// this is unfortunate. We have ended up here because we successfully
		// found a server that has since been removed. We need to start this process
		// over.
		continue
	}
}

// SelectServer returns a list of server descriptions matching
// a given selector. SelectServers will only time out when its
// parent context is done.
func SelectServers(ctx context.Context, m *Monitor, selector ServerSelector) ([]*server.Desc, error) {
	return selectServers(ctx, m, selector)
}

func selectServers(ctx context.Context, m monitor, selector ServerSelector) ([]*server.Desc, error) {
	updates, unsubscribe, _ := m.Subscribe()
	defer unsubscribe()

	var clusterDesc *Desc
	for {
		select {
		case <-ctx.Done():
			return nil, internal.WrapError(ctx.Err(), "server selection failed")
		case clusterDesc = <-updates:
			// topology has changed
		}

		var allowedServers []*server.Desc
		for _, s := range clusterDesc.Servers {
			if s.Type != server.Unknown {
				allowedServers = append(allowedServers, s)
			}
		}

		suitable, err := selector(clusterDesc, allowedServers)
		if err != nil {
			return nil, err
		}

		if len(suitable) > 0 {
			return suitable, nil
		}

		m.RequestImmediateCheck()
	}
}

// applyUpdate handles updating the current description as well
// as ensure that the servers are still accurate. This method
// *must* be called under the descLock.
func (c *Cluster) applyUpdate(desc *Desc) {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()

	if c.stateServers == nil {
		return
	}

	diff := Diff(c.stateDesc, desc)
	c.stateDesc = desc

	for _, added := range diff.AddedServers {
		if mon, ok := c.monitor.ServerMonitor(added.Endpoint); ok {
			c.stateServers[added.Endpoint] = server.NewWithMonitor(mon, c.cfg.serverOpts...)
		}
	}

	for _, removed := range diff.RemovedServers {
		if server, ok := c.stateServers[removed.Endpoint]; ok {
			server.Close()
		}

		delete(c.stateServers, removed.Endpoint)
	}
}
