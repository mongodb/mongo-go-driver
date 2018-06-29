package dispatch

import (
	"github.com/mongodb/mongo-go-driver/core/description"
	"github.com/mongodb/mongo-go-driver/core/topology"
)

func hasDataBearing(servers []description.Server) bool {
	for _, server := range servers {
		if server.DataBearing() {
			return true
		}
	}
	return false
}

func addDataBearingSelector(selector description.ServerSelector, topo *topology.Topology) description.ServerSelector {
	// If not a direct connection and no servers are data bearing, select on data bearing servers.
	if topo.Description().Kind != description.Single && hasDataBearing(topo.Description().Servers) {
		selector = description.CompositeSelector([]description.ServerSelector{
			selector,
			description.DataBearingSelector(),
		})
	}

	return selector
}
