package model

// Cluster is a description of a cluster.
type Cluster struct {
	Servers []*Server
	Kind    ClusterKind
}

// Server returns the model.Server with the specified address.
func (i *Cluster) Server(addr Addr) (*Server, bool) {
	for _, server := range i.Servers {
		if server.Addr.String() == addr.String() {
			return server, true
		}
	}
	return nil, false
}
