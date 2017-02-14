package cluster

import (
	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/server"
)

// Desc is a description of a cluster.
type Desc struct {
	Servers []*server.Desc
	Type    Type
}

// Server returns the server description with the specified endpoint.
func (d *Desc) Server(endpoint conn.Endpoint) (*server.Desc, bool) {
	for _, server := range d.Servers {
		if server.Endpoint == endpoint {
			return server, true
		}
	}
	return nil, false
}
