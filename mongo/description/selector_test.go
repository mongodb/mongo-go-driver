// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package description

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/tag"
)

func TestServerSelection(t *testing.T) {
	noerr := func(t *testing.T, err error) {
		if err != nil {
			t.Errorf("Unepexted error: %v", err)
			t.FailNow()
		}
	}

	t.Run("WriteSelector", func(t *testing.T) {
		testCases := []struct {
			name  string
			desc  Topology
			start int
			end   int
		}{
			{
				name: "ReplicaSetWithPrimary",
				desc: Topology{
					Kind: ReplicaSetWithPrimary,
					Servers: []Server{
						{Addr: address.Address("localhost:27017"), Kind: RSPrimary},
						{Addr: address.Address("localhost:27018"), Kind: RSSecondary},
						{Addr: address.Address("localhost:27019"), Kind: RSSecondary},
					},
				},
				start: 0,
				end:   1,
			},
			{
				name: "ReplicaSetNoPrimary",
				desc: Topology{
					Kind: ReplicaSetNoPrimary,
					Servers: []Server{
						{Addr: address.Address("localhost:27018"), Kind: RSSecondary},
						{Addr: address.Address("localhost:27019"), Kind: RSSecondary},
					},
				},
				start: 0,
				end:   0,
			},
			{
				name: "Sharded",
				desc: Topology{
					Kind: Sharded,
					Servers: []Server{
						{Addr: address.Address("localhost:27018"), Kind: Mongos},
						{Addr: address.Address("localhost:27019"), Kind: Mongos},
					},
				},
				start: 0,
				end:   2,
			},
			{
				name: "Single",
				desc: Topology{
					Kind: Single,
					Servers: []Server{
						{Addr: address.Address("localhost:27018"), Kind: Standalone},
					},
				},
				start: 0,
				end:   1,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := WriteSelector().SelectServer(tc.desc, tc.desc.Servers)
				noerr(t, err)
				if len(result) != tc.end-tc.start {
					t.Errorf("Incorrect number of servers selected. got %d; want %d", len(result), tc.end-tc.start)
				}
				if diff := cmp.Diff(result, tc.desc.Servers[tc.start:tc.end]); diff != "" {
					t.Errorf("Incorrect servers selected (-got +want):\n%s", diff)
				}
			})
		}
	})
	t.Run("LatencySelector", func(t *testing.T) {
		testCases := []struct {
			name  string
			desc  Topology
			start int
			end   int
		}{
			{
				name: "NoRTTSet",
				desc: Topology{
					Servers: []Server{
						{Addr: address.Address("localhost:27017")},
						{Addr: address.Address("localhost:27018")},
						{Addr: address.Address("localhost:27019")},
					},
				},
				start: 0,
				end:   3,
			},
			{
				name: "MultipleServers PartialNoRTTSet",
				desc: Topology{
					Servers: []Server{
						{Addr: address.Address("localhost:27017"), AverageRTT: 5 * time.Second, AverageRTTSet: true},
						{Addr: address.Address("localhost:27018"), AverageRTT: 10 * time.Second, AverageRTTSet: true},
						{Addr: address.Address("localhost:27019")},
					},
				},
				start: 0,
				end:   2,
			},
			{
				name: "MultipleServers",
				desc: Topology{
					Servers: []Server{
						{Addr: address.Address("localhost:27017"), AverageRTT: 5 * time.Second, AverageRTTSet: true},
						{Addr: address.Address("localhost:27018"), AverageRTT: 10 * time.Second, AverageRTTSet: true},
						{Addr: address.Address("localhost:27019"), AverageRTT: 26 * time.Second, AverageRTTSet: true},
					},
				},
				start: 0,
				end:   2,
			},
			{
				name:  "No Servers",
				desc:  Topology{Servers: []Server{}},
				start: 0,
				end:   0,
			},
			{
				name: "1 Server",
				desc: Topology{
					Servers: []Server{
						{Addr: address.Address("localhost:27017"), AverageRTT: 26 * time.Second, AverageRTTSet: true},
					},
				},
				start: 0,
				end:   1,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := LatencySelector(20*time.Second).SelectServer(tc.desc, tc.desc.Servers)
				noerr(t, err)
				if len(result) != tc.end-tc.start {
					t.Errorf("Incorrect number of servers selected. got %d; want %d", len(result), tc.end-tc.start)
				}
				if diff := cmp.Diff(result, tc.desc.Servers[tc.start:tc.end]); diff != "" {
					t.Errorf("Incorrect servers selected (-got +want):\n%s", diff)
				}
			})
		}
	})
}

var readPrefTestPrimary = Server{
	Addr:              address.Address("localhost:27017"),
	HeartbeatInterval: time.Duration(10) * time.Second,
	LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
	LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
	Kind:              RSPrimary,
	Tags:              tag.Set{tag.Tag{Name: "a", Value: "1"}},
	WireVersion:       &VersionRange{Min: 0, Max: 5},
}
var readPrefTestSecondary1 = Server{
	Addr:              address.Address("localhost:27018"),
	HeartbeatInterval: time.Duration(10) * time.Second,
	LastWriteTime:     time.Date(2017, 2, 11, 13, 58, 0, 0, time.UTC),
	LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
	Kind:              RSSecondary,
	Tags:              tag.Set{tag.Tag{Name: "a", Value: "1"}},
	WireVersion:       &VersionRange{Min: 0, Max: 5},
}
var readPrefTestSecondary2 = Server{
	Addr:              address.Address("localhost:27018"),
	HeartbeatInterval: time.Duration(10) * time.Second,
	LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
	LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
	Kind:              RSSecondary,
	Tags:              tag.Set{tag.Tag{Name: "a", Value: "2"}},
	WireVersion:       &VersionRange{Min: 0, Max: 5},
}
var readPrefTestTopology = Topology{
	Kind:    ReplicaSetWithPrimary,
	Servers: []Server{readPrefTestPrimary, readPrefTestSecondary1, readPrefTestSecondary2},
}

func TestSelector_Sharded(t *testing.T) {
	t.Parallel()

	subject := readpref.Primary()

	s := Server{
		Addr:              address.Address("localhost:27017"),
		HeartbeatInterval: time.Duration(10) * time.Second,
		LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
		LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
		Kind:              Mongos,
		WireVersion:       &VersionRange{Min: 0, Max: 5},
	}
	c := Topology{
		Kind:    Sharded,
		Servers: []Server{s},
	}

	result, err := ReadPrefSelector(subject).SelectServer(c, c.Servers)

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{s}, result)
}

func BenchmarkLatencySelector(b *testing.B) {
	for _, bcase := range []struct {
		name        string
		serversHook func(servers []Server)
	}{
		{
			name:        "AllFit",
			serversHook: func(servers []Server) {},
		},
		{
			name: "AllButOneFit",
			serversHook: func(servers []Server) {
				servers[0].AverageRTT = 2 * time.Second
			},
		},
		{
			name: "HalfFit",
			serversHook: func(servers []Server) {
				for i := 0; i < len(servers); i += 2 {
					servers[i].AverageRTT = 2 * time.Second
				}
			},
		},
		{
			name: "OneFit",
			serversHook: func(servers []Server) {
				for i := 1; i < len(servers); i++ {
					servers[i].AverageRTT = 2 * time.Second
				}
			},
		},
	} {
		bcase := bcase

		b.Run(bcase.name, func(b *testing.B) {
			s := Server{
				Addr:              address.Address("localhost:27017"),
				HeartbeatInterval: time.Duration(10) * time.Second,
				LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
				LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
				Kind:              Mongos,
				WireVersion:       &VersionRange{Min: 0, Max: 5},
				AverageRTTSet:     true,
				AverageRTT:        time.Second,
			}
			servers := make([]Server, 100)
			for i := 0; i < len(servers); i++ {
				servers[i] = s
			}
			bcase.serversHook(servers)
			//this will make base 1 sec latency < min (0.5) + conf (1)
			//and high latency 2 higher than the threshold
			servers[99].AverageRTT = 500 * time.Millisecond
			c := Topology{
				Kind:    Sharded,
				Servers: servers,
			}

			b.ResetTimer()
			b.RunParallel(func(p *testing.PB) {
				b.ReportAllocs()
				for p.Next() {
					_, _ = LatencySelector(time.Second).SelectServer(c, c.Servers)
				}
			})
		})
	}
}

func BenchmarkSelector_Sharded(b *testing.B) {
	for _, bcase := range []struct {
		name        string
		serversHook func(servers []Server)
	}{
		{
			name:        "AllFit",
			serversHook: func(servers []Server) {},
		},
		{
			name: "AllButOneFit",
			serversHook: func(servers []Server) {
				servers[0].Kind = LoadBalancer
			},
		},
		{
			name: "HalfFit",
			serversHook: func(servers []Server) {
				for i := 0; i < len(servers); i += 2 {
					servers[i].Kind = LoadBalancer
				}
			},
		},
		{
			name: "OneFit",
			serversHook: func(servers []Server) {
				for i := 1; i < len(servers); i++ {
					servers[i].Kind = LoadBalancer
				}
			},
		},
	} {
		bcase := bcase

		b.Run(bcase.name, func(b *testing.B) {
			subject := readpref.Primary()

			s := Server{
				Addr:              address.Address("localhost:27017"),
				HeartbeatInterval: time.Duration(10) * time.Second,
				LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
				LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
				Kind:              Mongos,
				WireVersion:       &VersionRange{Min: 0, Max: 5},
			}
			servers := make([]Server, 100)
			for i := 0; i < len(servers); i++ {
				servers[i] = s
			}
			bcase.serversHook(servers)
			c := Topology{
				Kind:    Sharded,
				Servers: servers,
			}

			b.ResetTimer()
			b.RunParallel(func(p *testing.PB) {
				b.ReportAllocs()
				for p.Next() {
					_, _ = ReadPrefSelector(subject).SelectServer(c, c.Servers)
				}
			})
		})
	}
}

func TestSelector_Single(t *testing.T) {
	t.Parallel()

	subject := readpref.Primary()

	s := Server{
		Addr:              address.Address("localhost:27017"),
		HeartbeatInterval: time.Duration(10) * time.Second,
		LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
		LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
		Kind:              Mongos,
		WireVersion:       &VersionRange{Min: 0, Max: 5},
	}
	c := Topology{
		Kind:    Single,
		Servers: []Server{s},
	}

	result, err := ReadPrefSelector(subject).SelectServer(c, c.Servers)

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{s}, result)
}

func TestSelector_Primary(t *testing.T) {
	t.Parallel()

	subject := readpref.Primary()

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, readPrefTestTopology.Servers)

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{readPrefTestPrimary}, result)
}

func TestSelector_Primary_with_no_primary(t *testing.T) {
	t.Parallel()

	subject := readpref.Primary()

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, []Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(t, err)
	require.Len(t, result, 0)
}

func TestSelector_PrimaryPreferred(t *testing.T) {
	t.Parallel()

	subject := readpref.PrimaryPreferred()

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, readPrefTestTopology.Servers)

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{readPrefTestPrimary}, result)
}

func TestSelector_PrimaryPreferred_ignores_tags(t *testing.T) {
	t.Parallel()

	subject := readpref.PrimaryPreferred(
		readpref.WithTags("a", "2"),
	)

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, readPrefTestTopology.Servers)

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{readPrefTestPrimary}, result)
}

func TestSelector_PrimaryPreferred_with_no_primary(t *testing.T) {
	t.Parallel()

	subject := readpref.PrimaryPreferred()

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, []Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, []Server{readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestSelector_PrimaryPreferred_with_no_primary_and_tags(t *testing.T) {
	t.Parallel()

	subject := readpref.PrimaryPreferred(
		readpref.WithTags("a", "2"),
	)

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, []Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{readPrefTestSecondary2}, result)
}

func TestSelector_PrimaryPreferred_with_maxStaleness(t *testing.T) {
	t.Parallel()

	subject := readpref.PrimaryPreferred(
		readpref.WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, readPrefTestTopology.Servers)

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{readPrefTestPrimary}, result)
}

func TestSelector_PrimaryPreferred_with_maxStaleness_and_no_primary(t *testing.T) {
	t.Parallel()

	subject := readpref.PrimaryPreferred(
		readpref.WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, []Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{readPrefTestSecondary2}, result)
}

func TestSelector_SecondaryPreferred(t *testing.T) {
	t.Parallel()

	subject := readpref.SecondaryPreferred()

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, readPrefTestTopology.Servers)

	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, []Server{readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestSelector_SecondaryPreferred_with_tags(t *testing.T) {
	t.Parallel()

	subject := readpref.SecondaryPreferred(
		readpref.WithTags("a", "2"),
	)

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, readPrefTestTopology.Servers)

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{readPrefTestSecondary2}, result)
}

func TestSelector_SecondaryPreferred_with_tags_that_do_not_match(t *testing.T) {
	t.Parallel()

	subject := readpref.SecondaryPreferred(
		readpref.WithTags("a", "3"),
	)

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, readPrefTestTopology.Servers)

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{readPrefTestPrimary}, result)
}

func TestSelector_SecondaryPreferred_with_tags_that_do_not_match_and_no_primary(t *testing.T) {
	t.Parallel()

	subject := readpref.SecondaryPreferred(
		readpref.WithTags("a", "3"),
	)

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, []Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(t, err)
	require.Len(t, result, 0)
}

func TestSelector_SecondaryPreferred_with_no_secondaries(t *testing.T) {
	t.Parallel()

	subject := readpref.SecondaryPreferred()

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, []Server{readPrefTestPrimary})

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{readPrefTestPrimary}, result)
}

func TestSelector_SecondaryPreferred_with_no_secondaries_or_primary(t *testing.T) {
	t.Parallel()

	subject := readpref.SecondaryPreferred()

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, []Server{})

	require.NoError(t, err)
	require.Len(t, result, 0)
}

func TestSelector_SecondaryPreferred_with_maxStaleness(t *testing.T) {
	t.Parallel()

	subject := readpref.SecondaryPreferred(
		readpref.WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, readPrefTestTopology.Servers)

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{readPrefTestSecondary2}, result)
}

func TestSelector_SecondaryPreferred_with_maxStaleness_and_no_primary(t *testing.T) {
	t.Parallel()

	subject := readpref.SecondaryPreferred(
		readpref.WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, []Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{readPrefTestSecondary2}, result)
}

func TestSelector_Secondary(t *testing.T) {
	t.Parallel()

	subject := readpref.Secondary()

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, readPrefTestTopology.Servers)

	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, []Server{readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestSelector_Secondary_with_tags(t *testing.T) {
	t.Parallel()

	subject := readpref.Secondary(
		readpref.WithTags("a", "2"),
	)

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, readPrefTestTopology.Servers)

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{readPrefTestSecondary2}, result)
}

func TestSelector_Secondary_with_empty_tag_set(t *testing.T) {
	t.Parallel()

	primaryNoTags := Server{
		Addr:        address.Address("localhost:27017"),
		Kind:        RSPrimary,
		WireVersion: &VersionRange{Min: 0, Max: 5},
	}
	firstSecondaryNoTags := Server{
		Addr:        address.Address("localhost:27018"),
		Kind:        RSSecondary,
		WireVersion: &VersionRange{Min: 0, Max: 5},
	}
	secondSecondaryNoTags := Server{
		Addr:        address.Address("localhost:27019"),
		Kind:        RSSecondary,
		WireVersion: &VersionRange{Min: 0, Max: 5},
	}
	topologyNoTags := Topology{
		Kind:    ReplicaSetWithPrimary,
		Servers: []Server{primaryNoTags, firstSecondaryNoTags, secondSecondaryNoTags},
	}

	nonMatchingSet := tag.Set{
		{Name: "foo", Value: "bar"},
	}
	emptyTagSet := tag.Set{}
	rp := readpref.Secondary(
		readpref.WithTagSets(nonMatchingSet, emptyTagSet),
	)

	result, err := ReadPrefSelector(rp).SelectServer(topologyNoTags, topologyNoTags.Servers)
	assert.Nil(t, err, "SelectServer error: %v", err)
	expectedResult := []Server{firstSecondaryNoTags, secondSecondaryNoTags}
	assert.Equal(t, expectedResult, result, "expected result %v, got %v", expectedResult, result)
}

func TestSelector_Secondary_with_tags_that_do_not_match(t *testing.T) {
	t.Parallel()

	subject := readpref.Secondary(
		readpref.WithTags("a", "3"),
	)

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, readPrefTestTopology.Servers)

	require.NoError(t, err)
	require.Len(t, result, 0)
}

func TestSelector_Secondary_with_no_secondaries(t *testing.T) {
	t.Parallel()

	subject := readpref.Secondary()

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, []Server{readPrefTestPrimary})

	require.NoError(t, err)
	require.Len(t, result, 0)
}

func TestSelector_Secondary_with_maxStaleness(t *testing.T) {
	t.Parallel()

	subject := readpref.Secondary(
		readpref.WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, readPrefTestTopology.Servers)

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{readPrefTestSecondary2}, result)
}

func TestSelector_Secondary_with_maxStaleness_and_no_primary(t *testing.T) {
	t.Parallel()

	subject := readpref.Secondary(
		readpref.WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, []Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{readPrefTestSecondary2}, result)
}

func TestSelector_Nearest(t *testing.T) {
	t.Parallel()

	subject := readpref.Nearest()

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, readPrefTestTopology.Servers)

	require.NoError(t, err)
	require.Len(t, result, 3)
	require.Equal(t, []Server{readPrefTestPrimary, readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestSelector_Nearest_with_tags(t *testing.T) {
	t.Parallel()

	subject := readpref.Nearest(
		readpref.WithTags("a", "1"),
	)

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, readPrefTestTopology.Servers)

	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, []Server{readPrefTestPrimary, readPrefTestSecondary1}, result)
}

func TestSelector_Nearest_with_tags_that_do_not_match(t *testing.T) {
	t.Parallel()

	subject := readpref.Nearest(
		readpref.WithTags("a", "3"),
	)

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, readPrefTestTopology.Servers)

	require.NoError(t, err)
	require.Len(t, result, 0)
}

func TestSelector_Nearest_with_no_primary(t *testing.T) {
	t.Parallel()

	subject := readpref.Nearest()

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, []Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, []Server{readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestSelector_Nearest_with_no_secondaries(t *testing.T) {
	t.Parallel()

	subject := readpref.Nearest()

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, []Server{readPrefTestPrimary})

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{readPrefTestPrimary}, result)
}

func TestSelector_Nearest_with_maxStaleness(t *testing.T) {
	t.Parallel()

	subject := readpref.Nearest(
		readpref.WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, readPrefTestTopology.Servers)

	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, []Server{readPrefTestPrimary, readPrefTestSecondary2}, result)
}

func TestSelector_Nearest_with_maxStaleness_and_no_primary(t *testing.T) {
	t.Parallel()

	subject := readpref.Nearest(
		readpref.WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := ReadPrefSelector(subject).SelectServer(readPrefTestTopology, []Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, []Server{readPrefTestSecondary2}, result)
}

func TestSelector_Max_staleness_is_less_than_90_seconds(t *testing.T) {
	t.Parallel()

	subject := readpref.Nearest(
		readpref.WithMaxStaleness(time.Duration(50) * time.Second),
	)

	s := Server{
		Addr:              address.Address("localhost:27017"),
		HeartbeatInterval: time.Duration(10) * time.Second,
		LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
		LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
		Kind:              RSPrimary,
		WireVersion:       &VersionRange{Min: 0, Max: 5},
	}
	c := Topology{
		Kind:    ReplicaSetWithPrimary,
		Servers: []Server{s},
	}

	_, err := ReadPrefSelector(subject).SelectServer(c, c.Servers)

	require.Error(t, err)
}

func TestSelector_Max_staleness_is_too_low(t *testing.T) {
	t.Parallel()

	subject := readpref.Nearest(
		readpref.WithMaxStaleness(time.Duration(100) * time.Second),
	)

	s := Server{
		Addr:              address.Address("localhost:27017"),
		HeartbeatInterval: time.Duration(100) * time.Second,
		LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
		LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
		Kind:              RSPrimary,
		WireVersion:       &VersionRange{Min: 0, Max: 5},
	}
	c := Topology{
		Kind:    ReplicaSetWithPrimary,
		Servers: []Server{s},
	}

	_, err := ReadPrefSelector(subject).SelectServer(c, c.Servers)

	require.Error(t, err)
}
