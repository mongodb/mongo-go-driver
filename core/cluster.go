package core

import (
	"sort"
	"strings"
	"sync"
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
	Desc() *ClusterDesc
	// SelectServer selects a server given a selector.
	SelectServer(ServerSelector) Server
}

// ClusterDesc is a description of a cluster.
type ClusterDesc struct {
	clusterType ClusterType
	servers     []*ServerDesc
}

// Server returns the ServerDesc with the specified endpoint.
func (d *ClusterDesc) Server(endpoint Endpoint) (*ServerDesc, bool) {
	for _, server := range d.servers {
		if server.endpoint == endpoint {
			return server, true
		}
	}
	return nil, false
}

// Servers are the known servers that are part of the cluster.
func (d *ClusterDesc) Servers() []*ServerDesc {
	new := make([]*ServerDesc, len(d.servers))
	copy(new, d.servers)
	return new
}

// Type is the type of the cluster.
func (d *ClusterDesc) Type() ClusterType {
	return d.clusterType
}

// ClusterType represents a type of the cluster.
type ClusterType uint32

// ServerType constants.
const (
	UnknownClusterType    ClusterType = 0
	Single                ClusterType = 1
	ReplicaSet            ClusterType = 2
	ReplicaSetNoPrimary   ClusterType = 4 + ReplicaSet
	ReplicaSetWithPrimary ClusterType = 8 + ReplicaSet
	Sharded               ClusterType = 256
)

// ServerSelector is a function that selects a server.
type ServerSelector func(*ClusterDesc, []*ServerDesc) []*ServerDesc

func diffClusterDesc(old, new *ClusterDesc) clusterDescDiff {
	var diff clusterDescDiff
	oldServers := serverDescSorter(old.Servers())
	newServers := serverDescSorter(new.Servers())

	sort.Sort(oldServers)
	sort.Sort(newServers)

	i := 0
	j := 0
	for {
		if i < len(oldServers) && j < len(newServers) {
			comp := strings.Compare(string(oldServers[i].endpoint), string(newServers[j].endpoint))
			switch comp {
			case 1:
				//left is bigger than
				diff.AddedServers = append(diff.AddedServers, newServers[j])
				j++
			case -1:
				// right is bigger
				diff.RemovedServers = append(diff.RemovedServers, oldServers[i])
				i++
			case 0:
				i++
				j++
			}
		} else if i < len(oldServers) {
			diff.RemovedServers = append(diff.RemovedServers, oldServers[i])
			i++
		} else if j < len(newServers) {
			diff.AddedServers = append(diff.AddedServers, newServers[j])
			j++
		} else {
			break
		}
	}

	return diff
}

type clusterDescDiff struct {
	AddedServers   []*ServerDesc
	RemovedServers []*ServerDesc
}

type serverDescSorter []*ServerDesc

func (x serverDescSorter) Len() int      { return len(x) }
func (x serverDescSorter) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x serverDescSorter) Less(i, j int) bool {
	return strings.Compare(string(x[i].endpoint), string(x[j].endpoint)) < 0
}

type clusterImpl struct {
	monitor  *ClusterMonitor
	updates  <-chan *ClusterDesc
	desc     *ClusterDesc
	descLock sync.Mutex
}

func (c *clusterImpl) Desc() *ClusterDesc {
	var desc *ClusterDesc
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
	selected := selector(clusterDesc, clusterDesc.Servers())
	serverOpts := c.monitor.serverOptionsFactory(selected[0].endpoint)
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
