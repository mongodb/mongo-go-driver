package core

import (
	"sync"

	"github.com/10gen/mongo-go-driver/core/desc"
)

// NewCluster creates a new Cluster.
func NewCluster(monitor *ClusterMonitor) Cluster {
	updates, _, _ := monitor.Subscribe()
	return &clusterImpl{
		monitor: monitor,
		updates: updates,
	}
}

// Cluster represents a connection to a cluster.
type Cluster interface {
	// Desc gets a description of the cluster.
	Desc() *desc.Cluster
	// SelectServer selects a server given a selector.
	SelectServer(ServerSelector) Server
}

// ServerSelector is a function that selects a server.
type ServerSelector func(*desc.Cluster, []*desc.Server) []*desc.Server

type clusterImpl struct {
	monitor  *ClusterMonitor
	updates  <-chan *desc.Cluster
	desc     *desc.Cluster
	descLock sync.Mutex
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

func (c *clusterImpl) SelectServer(selector ServerSelector) Server {
	clusterDesc := c.Desc()
	selected := selector(clusterDesc, clusterDesc.Servers)
	serverOpts := c.monitor.serverOptionsFactory(selected[0].Endpoint)
	serverOpts.fillDefaults()
	return &serverImpl{
		cluster:    c,
		serverOpts: serverOpts,
	}
}

type serverImpl struct {
	cluster    *clusterImpl
	serverOpts ServerOptions
}

func (s *serverImpl) Connection() (Connection, error) {
	return s.serverOpts.ConnectionDialer(s.serverOpts.ConnectionOptions)
}
