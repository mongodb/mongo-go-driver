package readpref

//go:generate go run selector_spec_test_generator.go

import (
	"fmt"
	"time"

	"github.com/10gen/mongo-go-driver/desc"
	"github.com/10gen/mongo-go-driver/feature"
)

// NewServerSelector creates a server selector that applies the read preference.
func NewServerSelector(rp *ReadPref) func(*desc.Cluster, []*desc.Server) ([]*desc.Server, error) {
	return func(cluster *desc.Cluster, servers []*desc.Server) ([]*desc.Server, error) {
		return SelectServer(rp, cluster, servers)
	}
}

// SelectServer returns the eligible servers from the provider serverDescs.
func SelectServer(rp *ReadPref, cluster *desc.Cluster, servers []*desc.Server) ([]*desc.Server, error) {
	if rp.maxStalenessSet {
		for _, server := range servers {
			if server.Type != desc.UnknownServerType {
				if err := feature.MaxStaleness(server.Version); err != nil {
					return nil, err
				}
			}
		}
	}

	switch cluster.Type {
	case desc.Single:
		return servers, nil
	case desc.ReplicaSetNoPrimary, desc.ReplicaSetWithPrimary:
		return selectForReplicaSet(rp, cluster, servers)
	case desc.Sharded:
		return selectByType(servers, desc.Mongos), nil
	}

	return nil, nil
}

func selectForReplicaSet(rp *ReadPref, cluster *desc.Cluster, servers []*desc.Server) ([]*desc.Server, error) {
	if err := verifyMaxStaleness(rp, cluster); err != nil {
		return nil, err
	}

	switch rp.Mode() {
	case PrimaryMode:
		return selectByType(servers, desc.RSPrimary), nil
	case PrimaryPreferredMode:
		selected := selectByType(servers, desc.RSPrimary)

		if len(selected) == 0 {
			selected = selectSecondaries(rp, servers)
			return selectByTagSet(selected, rp.tagSets), nil
		}

		return selected, nil
	case SecondaryPreferredMode:
		selected := selectSecondaries(rp, servers)
		selected = selectByTagSet(selected, rp.tagSets)
		if len(selected) > 0 {
			return selected, nil
		}
		return selectByType(servers, desc.RSPrimary), nil
	case SecondaryMode:
		selected := selectSecondaries(rp, servers)
		return selectByTagSet(selected, rp.tagSets), nil
	case NearestMode:
		selected := selectByType(servers, desc.RSPrimary)
		selected = append(selected, selectSecondaries(rp, servers)...)
		return selectByTagSet(selected, rp.tagSets), nil
	}

	return nil, fmt.Errorf("unsupported mode: %d", rp.mode)
}

func selectSecondaries(rp *ReadPref, servers []*desc.Server) []*desc.Server {
	secondaries := selectByType(servers, desc.RSSecondary)
	if len(secondaries) == 0 {
		return secondaries
	}
	if maxStaleness, set := rp.MaxStaleness(); set {

		primaries := selectByType(servers, desc.RSPrimary)
		if len(primaries) == 0 {
			baseTime := secondaries[0].LastWriteTime
			for i := 1; i < len(secondaries); i++ {
				if secondaries[i].LastWriteTime.After(baseTime) {
					baseTime = secondaries[i].LastWriteTime
				}
			}

			var selected []*desc.Server
			for _, secondary := range secondaries {
				estimatedStaleness := baseTime.Sub(secondary.LastWriteTime) + secondary.HeartbeatInterval
				if estimatedStaleness < maxStaleness {
					selected = append(selected, secondary)
				}
			}
			return selected
		}

		primary := primaries[0]

		var selected []*desc.Server
		for _, secondary := range secondaries {
			estimatedStaleness := secondary.LastUpdateTime.Sub(secondary.LastWriteTime) - primary.LastUpdateTime.Sub(primary.LastWriteTime) + secondary.HeartbeatInterval
			if estimatedStaleness < maxStaleness {
				selected = append(selected, secondary)
			}
		}
		return selected
	}

	return secondaries
}

func selectByTagSet(servers []*desc.Server, tagSets []desc.TagSet) []*desc.Server {
	if len(tagSets) == 0 {
		return servers
	}

	for _, ts := range tagSets {
		var results []*desc.Server
		for _, server := range servers {
			if len(server.Tags) > 0 && server.Tags.ContainsAll(ts) {
				results = append(results, server)
			}
		}

		if len(results) > 0 {
			return results
		}
	}

	return []*desc.Server{}
}

func selectByType(servers []*desc.Server, t desc.ServerType) []*desc.Server {
	var result []*desc.Server
	for _, server := range servers {
		if server.Type == t {
			result = append(result, server)
		}
	}

	return result
}

func verifyMaxStaleness(rp *ReadPref, cluster *desc.Cluster) error {
	maxStaleness, set := rp.MaxStaleness()
	if !set {
		return nil
	}

	if maxStaleness < time.Duration(90)*time.Second {
		return fmt.Errorf(
			"max staleness (%s) must be greater than or equal to 90s",
			maxStaleness,
		)
	}

	// we'll assume all servers have the same heartbeat interval...
	server := cluster.Servers[0]
	idleWritePeriod := time.Duration(10) * time.Second

	if maxStaleness < server.HeartbeatInterval+idleWritePeriod {
		return fmt.Errorf(
			"max staleness (%s) must be greater than or equal to the heartbeat interval (%s) plus idle write period (%s)",
			maxStaleness,
			server.HeartbeatInterval,
			idleWritePeriod)
	}

	return nil
}
