package cluster

//go:generate go run selector_spec_test_generator.go

import (
	"fmt"
	"time"

	"math"

	"github.com/10gen/mongo-go-driver/internal/feature"
	"github.com/10gen/mongo-go-driver/readpref"
	"github.com/10gen/mongo-go-driver/server"
)

// CompositeSelector combines mulitple selectors into a single selector.
func CompositeSelector(selectors []ServerSelector) ServerSelector {
	return func(c *Desc, candidates []*server.Desc) ([]*server.Desc, error) {
		var err error
		for _, sel := range selectors {
			candidates, err = sel(c, candidates)
			if err != nil {
				return nil, err
			}
		}
		return candidates, nil
	}
}

// LatencySelector creates a ServerSelector which selects servers based on their latency.
func LatencySelector(latency time.Duration) ServerSelector {
	return func(c *Desc, candidates []*server.Desc) ([]*server.Desc, error) {
		return selectServersByLatency(latency, c, candidates)
	}
}

func selectServersByLatency(latency time.Duration, c *Desc, candidates []*server.Desc) ([]*server.Desc, error) {
	if latency < 0 {
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

		max := min + latency

		var result []*server.Desc
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

// ReadPrefSelector creates a ServerSelector that selects servers based
// on a read preference.
func ReadPrefSelector(rp *readpref.ReadPref) ServerSelector {
	return func(c *Desc, candidates []*server.Desc) ([]*server.Desc, error) {
		return selectServer(rp, c, candidates)
	}
}

func selectServer(rp *readpref.ReadPref, c *Desc, candidates []*server.Desc) ([]*server.Desc, error) {
	if _, set := rp.MaxStaleness(); set {
		for _, s := range candidates {
			if s.Type != server.Unknown {
				if err := feature.MaxStaleness(s.Version); err != nil {
					return nil, err
				}
			}
		}
	}

	switch c.Type {
	case Single:
		return candidates, nil
	case ReplicaSetNoPrimary, ReplicaSetWithPrimary:
		return selectForReplicaSet(rp, c, candidates)
	case Sharded:
		return selectByType(candidates, server.Mongos), nil
	}

	return nil, nil
}

func selectForReplicaSet(rp *readpref.ReadPref, c *Desc, candidates []*server.Desc) ([]*server.Desc, error) {
	if err := verifyMaxStaleness(rp, c); err != nil {
		return nil, err
	}

	switch rp.Mode() {
	case readpref.PrimaryMode:
		return selectByType(candidates, server.RSPrimary), nil
	case readpref.PrimaryPreferredMode:
		selected := selectByType(candidates, server.RSPrimary)

		if len(selected) == 0 {
			selected = selectSecondaries(rp, candidates)
			return selectByTagSet(selected, rp.TagSets()), nil
		}

		return selected, nil
	case readpref.SecondaryPreferredMode:
		selected := selectSecondaries(rp, candidates)
		selected = selectByTagSet(selected, rp.TagSets())
		if len(selected) > 0 {
			return selected, nil
		}
		return selectByType(candidates, server.RSPrimary), nil
	case readpref.SecondaryMode:
		selected := selectSecondaries(rp, candidates)
		return selectByTagSet(selected, rp.TagSets()), nil
	case readpref.NearestMode:
		selected := selectByType(candidates, server.RSPrimary)
		selected = append(selected, selectSecondaries(rp, candidates)...)
		return selectByTagSet(selected, rp.TagSets()), nil
	}

	return nil, fmt.Errorf("unsupported mode: %d", rp.Mode())
}

func selectSecondaries(rp *readpref.ReadPref, candidates []*server.Desc) []*server.Desc {
	secondaries := selectByType(candidates, server.RSSecondary)
	if len(secondaries) == 0 {
		return secondaries
	}
	if maxStaleness, set := rp.MaxStaleness(); set {

		primaries := selectByType(candidates, server.RSPrimary)
		if len(primaries) == 0 {
			baseTime := secondaries[0].LastWriteTime
			for i := 1; i < len(secondaries); i++ {
				if secondaries[i].LastWriteTime.After(baseTime) {
					baseTime = secondaries[i].LastWriteTime
				}
			}

			var selected []*server.Desc
			for _, secondary := range secondaries {
				estimatedStaleness := baseTime.Sub(secondary.LastWriteTime) + secondary.HeartbeatInterval
				if estimatedStaleness < maxStaleness {
					selected = append(selected, secondary)
				}
			}
			return selected
		}

		primary := primaries[0]

		var selected []*server.Desc
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

func selectByTagSet(candidates []*server.Desc, tagSets []server.TagSet) []*server.Desc {
	if len(tagSets) == 0 {
		return candidates
	}

	for _, ts := range tagSets {
		var results []*server.Desc
		for _, s := range candidates {
			if len(s.Tags) > 0 && s.Tags.ContainsAll(ts) {
				results = append(results, s)
			}
		}

		if len(results) > 0 {
			return results
		}
	}

	return []*server.Desc{}
}

func selectByType(candidates []*server.Desc, t server.Type) []*server.Desc {
	var result []*server.Desc
	for _, s := range candidates {
		if s.Type == t {
			result = append(result, s)
		}
	}

	return result
}

func verifyMaxStaleness(rp *readpref.ReadPref, c *Desc) error {
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

	// we'll assume all candidates have the same heartbeat interval...
	s := c.Servers[0]
	idleWritePeriod := time.Duration(10) * time.Second

	if maxStaleness < s.HeartbeatInterval+idleWritePeriod {
		return fmt.Errorf(
			"max staleness (%s) must be greater than or equal to the heartbeat interval (%s) plus idle write period (%s)",
			maxStaleness,
			s.HeartbeatInterval,
			idleWritePeriod)
	}

	return nil
}
