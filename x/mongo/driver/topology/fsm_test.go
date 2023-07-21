// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/description"
)

func TestFSMSessionTimeout(t *testing.T) {
	t.Parallel()

	int64ToPtr := func(i64 int64) *int64 { return &i64 }

	tests := []struct {
		name string
		f    *fsm
		s    description.Server
		want *int64
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
					SessionTimeoutMinutesPtr: int64ToPtr(1),
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
					SessionTimeoutMinutesPtr: int64ToPtr(2),
				},
			},
			s: description.Server{
				Kind:                     description.RSPrimary,
				SessionTimeoutMinutesPtr: int64ToPtr(1),
			},
			want: int64ToPtr(1),
		},
		{
			name: "session support on data-bearing server with no session support on fsm with no servers",
			f:    &fsm{Topology: description.Topology{}},
			s: description.Server{
				Kind:                     description.RSPrimary,
				SessionTimeoutMinutesPtr: int64ToPtr(1),
			},
			want: int64ToPtr(1),
		},
		{
			name: "session support on data-bearing server with no session support on fsm and lower servers",
			f: &fsm{Topology: description.Topology{
				Servers: []description.Server{
					{
						Kind:                     description.RSPrimary,
						SessionTimeoutMinutesPtr: int64ToPtr(1),
					},
				},
			}},
			s: description.Server{
				Kind:                     description.RSPrimary,
				SessionTimeoutMinutesPtr: int64ToPtr(2),
			},
			want: int64ToPtr(1),
		},
		{
			name: "session support on data-bearing server with no session support on fsm and higher servers",
			f: &fsm{Topology: description.Topology{
				Servers: []description.Server{
					{
						Kind:                     description.RSPrimary,
						SessionTimeoutMinutesPtr: int64ToPtr(3),
					},
				},
			}},
			s: description.Server{
				Kind:                     description.RSPrimary,
				SessionTimeoutMinutesPtr: int64ToPtr(2),
			},
			want: int64ToPtr(2),
		},
	}

	for _, test := range tests {
		test := test // capture the range variable

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := selectFSMSessionTimeout(test.f, test.s)
			gotStr := "nil"
			wantStr := "nil"
			if got != nil {
				gotStr = fmt.Sprintf("%v", *got)
			}
			if test.want != nil {
				wantStr = fmt.Sprintf("%v", *test.want)
			}
			assert.Equal(t, test.want, got, "minFSMServersTimeout() = %v (%v), wanted %v (%v).", got, gotStr, test.want, wantStr)
		})
	}

	t.Run("test timeout after server removed", func(t *testing.T) {
		f := fsm{
			Topology: description.Topology{
				Kind:                     description.ReplicaSetWithPrimary,
				SessionTimeoutMinutesPtr: int64ToPtr(1),
				Servers: []description.Server{
					{
						Kind:                     description.RSPrimary,
						SessionTimeoutMinutesPtr: int64ToPtr(2),
						Addr:                     "host1:27017",
						Members: []address.Address{
							"host1:27017",
							"host2:27017",
						},
					},
					{
						Kind:                     description.RSSecondary,
						SessionTimeoutMinutesPtr: int64ToPtr(1),
						Addr:                     "host2:27017",
						Members: []address.Address{
							"host1:27017",
							"host2:27017",
						},
					},
				},
			},
		}

		// Apply a new server description that removes `host2` from the FSM.
		f.apply(description.Server{
			Kind:                     description.RSPrimary,
			SessionTimeoutMinutesPtr: int64ToPtr(2),
			Addr:                     "host1:27017",
			Members: []address.Address{
				"host1:27017",
			},
		})

		expect := []description.Server{
			{
				Kind:                     description.RSPrimary,
				SessionTimeoutMinutesPtr: int64ToPtr(2),
				Addr:                     "host1:27017",
				Members: []address.Address{
					"host1:27017",
				},
			},
		}

		assert.Equal(t, expect, f.Servers)

		// Check that the SessionTimeoutMinutesPtr has been updated to the new minimum (2).
		assert.NotNil(t, f.SessionTimeoutMinutesPtr)
		assert.Equal(t, *f.SessionTimeoutMinutesPtr, int64(2))
	})
}
