// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// TODO: Remove this package.

package experiment

import (
	"context"
	"time"
)

// Metrics is a collector for timing trace values for the "maxTimeMS" feedback
// mechanism experiment.
type Metrics struct {
	Time              time.Time
	Server            string
	OriginalTimeout   time.Duration
	RemainingTimeout  time.Duration
	MinRTT            time.Duration
	AdjustmentPct     float64
	Adjustment        time.Duration
	MaxTimeMS         int64
	OpDuration        time.Duration
	Retries           int
	ConnectionsClosed int
	Err               error
}

type metricsValue struct{}

// WithMetrics adds a Metrics to the given context.
func WithMetrics(ctx context.Context, metrics *Metrics) context.Context {
	return context.WithValue(ctx, metricsValue{}, metrics)
}

// GetMetrics returns the Metrics from the given context, or nil if there is no
// Metrics.
func GetMetrics(ctx context.Context) *Metrics {
	val := ctx.Value(metricsValue{})
	if val == nil {
		return nil
	}
	return val.(*Metrics)
}
