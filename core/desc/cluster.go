package desc

// Cluster is a description of a cluster.
type Cluster struct {
	ClusterType ClusterType
	Servers     []*Server
}

// Server returns the ServerDesc with the specified endpoint.
func (d *Cluster) Server(endpoint Endpoint) (*Server, bool) {
	for _, server := range d.Servers {
		if server.Endpoint == endpoint {
			return server, true
		}
	}
	return nil, false
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
