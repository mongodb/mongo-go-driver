// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package stats

import (
	"container/list"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
)

func TestStandardDeviationList_Float64(t *testing.T) {
	tests := []struct {
		name string
		data []float64
		want float64
	}{
		{
			name: "empty",
			data: []float64{},
			want: 0,
		},
		{
			name: "multiple",
			data: []float64{-1, 1, 6, 8, 10},
			want: 4.1665333,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			l := list.New()
			for _, d := range test.data {
				l.PushBack(d)
			}

			got := StandardDeviationList[float64](l)

			assert.InDelta(t, test.want, got, 1e-6)
		})
	}
}

func TestStandardDeviationList_Duration(t *testing.T) {
	tests := []struct {
		name string
		data []time.Duration
		want float64
	}{
		{
			name: "empty",
			data: []time.Duration{},
			want: 0,
		},
		{
			name: "multiple",
			data: []time.Duration{
				time.Millisecond,
				2 * time.Millisecond,
				time.Microsecond,
			},
			want: 816088.36667497,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			l := list.New()
			for _, d := range test.data {
				l.PushBack(d)
			}

			got := StandardDeviationList[time.Duration](l)

			assert.InDelta(t, test.want, got, 1e-6)
		})
	}
}
