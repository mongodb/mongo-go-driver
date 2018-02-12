// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package cluster

import (
	"time"

	"math"

	"github.com/mongodb/mongo-go-driver/mongo/model"
)

// CompositeSelector combines multiple selectors into a single selector.
func CompositeSelector(selectors []ServerSelector) ServerSelector {
	return func(c *model.Cluster, candidates []*model.Server) ([]*model.Server, error) {
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
	return func(c *model.Cluster, candidates []*model.Server) ([]*model.Server, error) {
		return selectServersByLatency(latency, c, candidates)
	}
}

func selectServersByLatency(latency time.Duration, c *model.Cluster, candidates []*model.Server) ([]*model.Server, error) {
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

		var result []*model.Server
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
	return func(c *model.Cluster, candidates []*model.Server) ([]*model.Server, error) {
		switch c.Kind {
		case model.Single:
			return candidates, nil
		default:
			var result []*model.Server
			for _, candidate := range candidates {
				switch candidate.Kind {
				case model.Mongos, model.RSPrimary, model.Standalone:
					result = append(result, candidate)
				}
			}
			return result, nil
		}
	}
}
