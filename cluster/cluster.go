package cluster

import (
	"sync"

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
	Desc() *Desc
	// SelectServer selects a server given a selector.
	SelectServer(ServerSelector) (server.Server, error)
}

// ServerSelector is a function that selects a server.
type ServerSelector func(*Desc, []*server.Desc) ([]*server.Desc, error)

type clusterImpl struct {
	monitor     *Monitor
	ownsMonitor bool
	updates     <-chan *Desc
	desc        *Desc
	descLock    sync.Mutex
}

func (c *clusterImpl) Close() {
	if c.ownsMonitor {
		c.monitor.Stop()
	}
}

func (c *clusterImpl) Desc() *Desc {
	var desc *Desc
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

func (c *clusterImpl) SelectServer(selector ServerSelector) (server.Server, error) {
	desc := c.Desc()
	selected, err := selector(desc, desc.Servers)
	if err != nil {
		return nil, err
	}

	// TODO: put this logic into the monitor...
	c.monitor.serversLock.Lock()
	serverMonitor := c.monitor.servers[selected[0].Endpoint]
	c.monitor.serversLock.Unlock()
	return server.NewWithMonitor(serverMonitor), nil
}
