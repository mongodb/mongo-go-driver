package topology

import (
	"math"
	"time"

	"github.com/mongodb/mongo-go-driver/mongo/private/roots/description"
)

// ServerSelector is an interface implemented by types that can select a server given a
// topology description.
type ServerSelector interface {
	SelectServer(description.Topology, []description.Server) ([]description.Server, error)
}

// SelectedServer represents a specific server that was selected during server selection.
// It contains the kind of the typology it was selected from and the read preference that
// was given during server selection.
type SelectedServer struct {
	*Server

	Kind description.TopologyKind
}

// Description returns a description of the server as of the last heartbeat.
func (ss *SelectedServer) Description() description.SelectedServer {
	sdesc := ss.Server.Description()
	return description.SelectedServer{
		Server: sdesc,
		Kind:   ss.Kind,
	}
}

// ServerSelectorFunc is a function that can be used as a ServerSelector.
type ServerSelectorFunc func(description.Topology, []description.Server) ([]description.Server, error)

// SelectServer implements the ServerSelector interface.
func (ssf ServerSelectorFunc) SelectServer(t description.Topology, s []description.Server) ([]description.Server, error) {
	return ssf(t, s)
}

type compositeSelector struct {
	selectors []ServerSelector
}

// CompositeSelector combines multiple selectors into a single selector.
func CompositeSelector(selectors []ServerSelector) ServerSelector {
	return &compositeSelector{selectors: selectors}
}

func (cs *compositeSelector) SelectServer(t description.Topology, candidates []description.Server) ([]description.Server, error) {
	var err error
	for _, sel := range cs.selectors {
		candidates, err = sel.SelectServer(t, candidates)
		if err != nil {
			return nil, err
		}
	}
	return candidates, nil
}

type latencySelector struct {
	latency time.Duration
}

// LatencySelector creates a ServerSelector which selects servers based on their latency.
func LatencySelector(latency time.Duration) ServerSelector {
	return &latencySelector{latency: latency}
}

func (ls *latencySelector) SelectServer(t description.Topology, candidates []description.Server) ([]description.Server, error) {
	if ls.latency < 0 {
		return candidates, nil
	}

	switch len(candidates) {
	case 0, 1:
		return candidates, nil
	default:
		min := time.Duration(math.MaxInt64)
		for _, candidate := range candidates {
			if candidate.AverageRTTSet {
				if candidate.AverageRTT < min {
					min = candidate.AverageRTT
				}
			}
		}

		if min == math.MaxInt64 {
			return candidates, nil
		}

		max := min + ls.latency

		var result []description.Server
		for _, candidate := range candidates {
			if candidate.AverageRTTSet {
				if candidate.AverageRTT <= max {
					result = append(result, candidate)
				}
			}
		}

		return result, nil
	}
}

// WriteSelector selects all the writable servers.
func WriteSelector() ServerSelector {
	return ServerSelectorFunc(func(t description.Topology, candidates []description.Server) ([]description.Server, error) {
		switch t.Kind {
		case description.Single:
			return candidates, nil
		default:
			result := []description.Server{}
			for _, candidate := range candidates {
				switch candidate.Kind {
				case description.Mongos, description.RSPrimary, description.Standalone:
					result = append(result, candidate)
				}
			}
			return result, nil
		}
	})
}
