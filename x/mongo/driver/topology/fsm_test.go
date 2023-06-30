// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"testing"

	"go.mongodb.org/mongo-driver/internal"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/mongo/description"
)

func TestCompareUint32Ptr(t *testing.T) {
	t.Parallel()

	uint32ToPtr := func(u uint32) *uint32 { return &u }
	unitPtr := uint32ToPtr(1)

	tests := []struct {
		name       string
		ptr1, ptr2 *uint32
		want       int
	}{
		{
			name: "empty",
			want: 0,
		},
		{
			name: "ptr1 nil",
			ptr2: uint32ToPtr(1),
			want: -2,
		},
		{
			name: "ptr2 nil",
			ptr1: uint32ToPtr(1),
			want: 2,
		},
		{
			name: "ptr1 and ptr2 have same value, differnt address",
			ptr1: uint32ToPtr(1),
			ptr2: uint32ToPtr(1),
			want: 0,
		},
		{
			name: "ptr1 and ptr2 have the same address",
			ptr1: unitPtr,
			ptr2: unitPtr,
			want: 0,
		},
		{
			name: "ptr1 GT ptr2",
			ptr1: uint32ToPtr(1),
			ptr2: uint32ToPtr(0),
			want: 1,
		},
		{
			name: "ptr1 LT ptr2",
			ptr1: uint32ToPtr(0),
			ptr2: uint32ToPtr(1),
			want: -1,
		},
	}

	for _, test := range tests {
		test := test // capture the range variable

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := internal.CompareUint32Ptr(test.ptr1, test.ptr2)
			assert.Equal(t, test.want, got, "compareUint32Ptr() = %v, wanted %v", got, test.want)
		})
	}
}

func TestFSMSessionTimeout(t *testing.T) {
	t.Parallel()

	uint32ToPtr := func(u uint32) *uint32 { return &u }

	tests := []struct {
		name string
		f    *fsm
		s    description.Server
		want *uint32
	}{
		{
			name: "empty",
			f:    &fsm{},
			s:    description.Server{},
			want: nil,
		},
		{
			name: "no session support on data-bearing server with session support on fsm",
			f: &fsm{
				Topology: description.Topology{
					SessionTimeoutMinutesPtr: uint32ToPtr(1),
				},
			},
			s: description.Server{
				Kind: description.RSPrimary,
			},
			want: nil,
		},
		{
			name: "lower timeout on data-bearing server with session support on fsm",
			f: &fsm{
				Topology: description.Topology{
					SessionTimeoutMinutesPtr: uint32ToPtr(2),
				},
			},
			s: description.Server{
				Kind:                     description.RSPrimary,
				SessionTimeoutMinutesPtr: uint32ToPtr(1),
			},
			want: uint32ToPtr(1),
		},
		{
			name: "session support on data-bearing server with no session support on fsm with no servers",
			f:    &fsm{Topology: description.Topology{}},
			s: description.Server{
				Kind:                     description.RSPrimary,
				SessionTimeoutMinutesPtr: uint32ToPtr(1),
			},
			want: uint32ToPtr(1),
		},
		{
			name: "session support on data-bearing server with no session support on fsm and lower servers",
			f: &fsm{Topology: description.Topology{
				Servers: []description.Server{
					{
						Kind:                     description.RSPrimary,
						SessionTimeoutMinutesPtr: uint32ToPtr(1),
					},
				},
			}},
			s: description.Server{
				Kind:                     description.RSPrimary,
				SessionTimeoutMinutesPtr: uint32ToPtr(2),
			},
			want: uint32ToPtr(1),
		},
		{
			name: "session support on data-bearing server with no session support on fsm and higher servers",
			f: &fsm{Topology: description.Topology{
				Servers: []description.Server{
					{
						Kind:                     description.RSPrimary,
						SessionTimeoutMinutesPtr: uint32ToPtr(3),
					},
				},
			}},
			s: description.Server{
				Kind:                     description.RSPrimary,
				SessionTimeoutMinutesPtr: uint32ToPtr(2),
			},
			want: uint32ToPtr(2),
		},
	}

	for _, test := range tests {
		test := test // capture the range variable

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := selectFSMSessionTimeout(test.f, test.s)
			assert.Equal(t, test.want, got, "minFSMServersTimeout() = %v, wanted %v", got, test.want)
		})
	}
}
