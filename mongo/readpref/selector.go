// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package readpref

import (
	"fmt"
	"time"

	"github.com/mongodb/mongo-go-driver/mongo/internal/feature"
	"github.com/mongodb/mongo-go-driver/mongo/model"
)

// Selector creates a ServerSelector that selects servers based
// on a read preference.
func Selector(rp *ReadPref) func(c *model.Cluster, candidates []*model.Server) ([]*model.Server, error) {
	return func(c *model.Cluster, candidates []*model.Server) ([]*model.Server, error) {
		return selectServer(rp, c, candidates)
	}
}

func selectServer(rp *ReadPref, c *model.Cluster, candidates []*model.Server) ([]*model.Server, error) {
	if _, set := rp.MaxStaleness(); set {
		for _, s := range candidates {
			if s.Kind != model.Unknown {
				if err := feature.MaxStaleness(s.Version, s.WireVersion); err != nil {
					return nil, err
				}
			}
		}
	}

	switch c.Kind {
	case model.Single:
		return candidates, nil
	case model.ReplicaSetNoPrimary, model.ReplicaSetWithPrimary:
		return selectForReplicaSet(rp, c, candidates)
	case model.Sharded:
		return selectByKind(candidates, model.Mongos), nil
	}

	return nil, nil
}

func selectForReplicaSet(rp *ReadPref, c *model.Cluster, candidates []*model.Server) ([]*model.Server, error) {
	if err := verifyMaxStaleness(rp, c); err != nil {
		return nil, err
	}

	switch rp.Mode() {
	case PrimaryMode:
		return selectByKind(candidates, model.RSPrimary), nil
	case PrimaryPreferredMode:
		selected := selectByKind(candidates, model.RSPrimary)

		if len(selected) == 0 {
			selected = selectSecondaries(rp, candidates)
			return selectByTagSet(selected, rp.TagSets()), nil
		}

		return selected, nil
	case SecondaryPreferredMode:
		selected := selectSecondaries(rp, candidates)
		selected = selectByTagSet(selected, rp.TagSets())
		if len(selected) > 0 {
			return selected, nil
		}
		return selectByKind(candidates, model.RSPrimary), nil
	case SecondaryMode:
		selected := selectSecondaries(rp, candidates)
		return selectByTagSet(selected, rp.TagSets()), nil
	case NearestMode:
		selected := selectByKind(candidates, model.RSPrimary)
		selected = append(selected, selectSecondaries(rp, candidates)...)
		return selectByTagSet(selected, rp.TagSets()), nil
	}

	return nil, fmt.Errorf("unsupported mode: %d", rp.Mode())
}

func selectSecondaries(rp *ReadPref, candidates []*model.Server) []*model.Server {
	secondaries := selectByKind(candidates, model.RSSecondary)
	if len(secondaries) == 0 {
		return secondaries
	}
	if maxStaleness, set := rp.MaxStaleness(); set {

		primaries := selectByKind(candidates, model.RSPrimary)
		if len(primaries) == 0 {
			baseTime := secondaries[0].LastWriteTime
			for i := 1; i < len(secondaries); i++ {
				if secondaries[i].LastWriteTime.After(baseTime) {
					baseTime = secondaries[i].LastWriteTime
				}
			}

			var selected []*model.Server
			for _, secondary := range secondaries {
				estimatedStaleness := baseTime.Sub(secondary.LastWriteTime) + secondary.HeartbeatInterval
				if estimatedStaleness <= maxStaleness {
					selected = append(selected, secondary)
				}
			}
			return selected
		}

		primary := primaries[0]

		var selected []*model.Server
		for _, secondary := range secondaries {
			estimatedStaleness := secondary.LastUpdateTime.Sub(secondary.LastWriteTime) - primary.LastUpdateTime.Sub(primary.LastWriteTime) + secondary.HeartbeatInterval
			if estimatedStaleness <= maxStaleness {
				selected = append(selected, secondary)
			}
		}
		return selected
	}

	return secondaries
}

func selectByTagSet(candidates []*model.Server, tagSets []model.TagSet) []*model.Server {
	if len(tagSets) == 0 {
		return candidates
	}

	for _, ts := range tagSets {
		var results []*model.Server
		for _, s := range candidates {
			if len(s.Tags) > 0 && s.Tags.ContainsAll(ts) {
				results = append(results, s)
			}
		}

		if len(results) > 0 {
			return results
		}
	}

	return []*model.Server{}
}

func selectByKind(candidates []*model.Server, kind model.ServerKind) []*model.Server {
	var result []*model.Server
	for _, s := range candidates {
		if s.Kind == kind {
			result = append(result, s)
		}
	}

	return result
}

func verifyMaxStaleness(rp *ReadPref, c *model.Cluster) error {
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
