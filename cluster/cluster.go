package cluster

import (
	"sync"

	"github.com/10gen/mongo-go-driver/desc"
	"github.com/10gen/mongo-go-driver/server"
)

// New creates a new cluster. Internally, it
// creates a new Monitor with which to monitor the
// state of the cluster. When the Cluster is closed,
// the monitor will be stopped.
func New(opts ...Option) (Cluster, error) {
	monitor, err := StartMonitor(opts...)
	if err != nil {
		return nil, err
	}

	updates, _, _ := monitor.Subscribe()
	return &clusterImpl{
		monitor:     monitor,
		ownsMonitor: true,
		updates:     updates,
	}, nil
}

// NewWithMonitor creates a new Cluster from
// an existing monitor. When the cluster is closed,
// the monitor will not be stopped.
func NewWithMonitor(monitor *Monitor) Cluster {
	updates, _, _ := monitor.Subscribe()
	return &clusterImpl{
		monitor: monitor,
		updates: updates,
	}
}

// Cluster represents a connection to a cluster.
type Cluster interface {
	// Close closes the cluster.
	Close()
	// Desc gets a description of the cluster.
	Desc() *desc.Cluster
	// SelectServer selects a server given a selector.
	SelectServer(ServerSelector) server.Server
}

// ServerSelector is a function that selects a server.
type ServerSelector func(*desc.Cluster, []*desc.Server) []*desc.Server

type clusterImpl struct {
	monitor     *Monitor
	ownsMonitor bool
	updates     <-chan *desc.Cluster
	desc        *desc.Cluster
	descLock    sync.Mutex
}

func (c *clusterImpl) Close() {
	if c.ownsMonitor {
		c.monitor.Stop()
	}
}

func (c *clusterImpl) Desc() *desc.Cluster {
	var desc *desc.Cluster
	c.descLock.Lock()
	select {
	case desc = <-c.updates:
		c.desc = desc
	default:
		// no updates
	}
	c.descLock.Unlock()
	return desc
}

func (c *clusterImpl) SelectServer(selector ServerSelector) server.Server {
	clusterDesc := c.Desc()
	selected := selector(clusterDesc, clusterDesc.Servers)[0]

	// TODO: put this logic into the monitor...
	c.monitor.serversLock.Lock()
	serverMonitor := c.monitor.servers[selected.Endpoint]
	c.monitor.serversLock.Unlock()
	return server.NewWithMonitor(serverMonitor)
}
