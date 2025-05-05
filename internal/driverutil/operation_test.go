// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driverutil

import (
	"context"
	"math"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
)

func TestCalculateMaxTimeMS(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		rttMin       time.Duration
		wantZero     bool
		wantOk       bool
		wantPositive bool
		wantExact    int64
	}{
		{
			name:         "no deadline",
			ctx:          context.Background(),
			rttMin:       10 * time.Millisecond,
			wantZero:     true,
			wantOk:       true,
			wantPositive: false,
		},
		{
			name: "deadline expired",
			ctx: func() context.Context {
				ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Second)) //nolint:govet
				return ctx
			}(),
			wantZero:     true,
			wantOk:       false,
			wantPositive: false,
		},
		{
			name: "remaining timeout < rttMin",
			ctx: func() context.Context {
				ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(1*time.Millisecond)) //nolint:govet
				return ctx
			}(),
			rttMin:       10 * time.Millisecond,
			wantZero:     true,
			wantOk:       false,
			wantPositive: false,
		},
		{
			name: "normal positive result",
			ctx: func() context.Context {
				ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(100*time.Millisecond)) //nolint:govet
				return ctx
			}(),
			wantZero:     false,
			wantOk:       true,
			wantPositive: true,
		},
		{
			name: "beyond maxInt32",
			ctx: func() context.Context {
				dur := time.Now().Add(time.Duration(math.MaxInt32+1000) * time.Millisecond)
				ctx, _ := context.WithDeadline(context.Background(), dur) //nolint:govet
				return ctx
			}(),
			wantZero:     true,
			wantOk:       true,
			wantPositive: false,
		},
		{
			name: "round up to 1ms",
			ctx: func() context.Context {
				ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(999*time.Microsecond)) //nolint:govet
				return ctx
			}(),
			wantOk:    true,
			wantExact: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := CalculateMaxTimeMS(tt.ctx, tt.rttMin)

			assert.Equal(t, tt.wantOk, got1)

			if tt.wantExact > 0 && got != tt.wantExact {
				t.Errorf("CalculateMaxTimeMS() got = %v, want %v", got, tt.wantExact)
			}

			if tt.wantZero && got != 0 {
				t.Errorf("CalculateMaxTimeMS() got = %v, want 0", got)
			}

			if !tt.wantZero && got == 0 {
				t.Errorf("CalculateMaxTimeMS() got = %v, want > 0", got)
			}

			if !tt.wantZero && tt.wantPositive && got <= 0 {
				t.Errorf("CalculateMaxTimeMS() got = %v, want > 0", got)
			}
		})
	}

}
