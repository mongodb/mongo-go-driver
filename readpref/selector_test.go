package readpref_test

import (
	"testing"
	"time"

	"github.com/10gen/mongo-go-driver/desc"
	. "github.com/10gen/mongo-go-driver/readpref"
	"github.com/stretchr/testify/require"
)

var readPrefTestPrimary = &desc.Server{
	Endpoint:          desc.Endpoint("localhost:27017"),
	HeartbeatInterval: time.Duration(10) * time.Second,
	LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
	LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
	Type:              desc.RSPrimary,
	Tags:              desc.NewTagSet("a", "1"),
	Version:           desc.NewVersion(3, 4, 0),
}
var readPrefTestSecondary1 = &desc.Server{
	Endpoint:          desc.Endpoint("localhost:27018"),
	HeartbeatInterval: time.Duration(10) * time.Second,
	LastWriteTime:     time.Date(2017, 2, 11, 13, 58, 0, 0, time.UTC),
	LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
	Type:              desc.RSSecondary,
	Tags:              desc.NewTagSet("a", "1"),
	Version:           desc.NewVersion(3, 4, 0),
}
var readPrefTestSecondary2 = &desc.Server{
	Endpoint:          desc.Endpoint("localhost:27018"),
	HeartbeatInterval: time.Duration(10) * time.Second,
	LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
	LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
	Type:              desc.RSSecondary,
	Tags:              desc.NewTagSet("a", "2"),
	Version:           desc.NewVersion(3, 4, 0),
}
var readPrefTestCluster = &desc.Cluster{
	Type:    desc.ReplicaSetWithPrimary,
	Servers: []*desc.Server{readPrefTestPrimary, readPrefTestSecondary1, readPrefTestSecondary2},
}

func TestSelectServer_Sharded(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(PrimaryMode)

	server := &desc.Server{
		Endpoint:          desc.Endpoint("localhost:27017"),
		HeartbeatInterval: time.Duration(10) * time.Second,
		LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
		LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
		Type:              desc.Mongos,
		Version:           desc.NewVersion(3, 4, 0),
	}
	cluster := &desc.Cluster{
		Type:    desc.Sharded,
		Servers: []*desc.Server{server},
	}

	result, err := SelectServer(subject, cluster, cluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{server}, result)
}

func TestSelectServer_Single(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(PrimaryMode)

	server := &desc.Server{
		Endpoint:          desc.Endpoint("localhost:27017"),
		HeartbeatInterval: time.Duration(10) * time.Second,
		LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
		LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
		Type:              desc.Mongos,
		Version:           desc.NewVersion(3, 4, 0),
	}
	cluster := &desc.Cluster{
		Type:    desc.Single,
		Servers: []*desc.Server{server},
	}

	result, err := SelectServer(subject, cluster, cluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{server}, result)
}

func TestSelectServer_Primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(PrimaryMode)

	result, err := SelectServer(subject, readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{readPrefTestPrimary}, result)
}

func TestSelectServer_Primary_with_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(PrimaryMode)

	result, err := SelectServer(subject, readPrefTestCluster, []*desc.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Empty(result, 0)
}

func TestSelectServer_PrimaryPreferred(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(PrimaryPreferredMode)

	result, err := SelectServer(subject, readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{readPrefTestPrimary}, result)
}

func TestSelectServer_PrimaryPreferred_ignores_tags(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(PrimaryPreferredMode, desc.NewTagSet("a", "2"))

	result, err := SelectServer(subject, readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{readPrefTestPrimary}, result)
}

func TestSelectServer_PrimaryPreferred_with_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(PrimaryPreferredMode)

	result, err := SelectServer(subject, readPrefTestCluster, []*desc.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*desc.Server{readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestSelectServer_PrimaryPreferred_with_no_primary_and_tags(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(PrimaryPreferredMode, desc.NewTagSet("a", "2"))

	result, err := SelectServer(subject, readPrefTestCluster, []*desc.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{readPrefTestSecondary2}, result)
}

func TestSelectServer_PrimaryPreferred_with_maxStaleness(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := NewWithMaxStaleness(PrimaryPreferredMode, time.Duration(90)*time.Second)

	result, err := SelectServer(subject, readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{readPrefTestPrimary}, result)
}

func TestSelectServer_PrimaryPreferred_with_maxStaleness_and_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := NewWithMaxStaleness(PrimaryPreferredMode, time.Duration(90)*time.Second)

	result, err := SelectServer(subject, readPrefTestCluster, []*desc.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{readPrefTestSecondary2}, result)
}

func TestSelectServer_SecondaryPreferred(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(SecondaryPreferredMode)

	result, err := SelectServer(subject, readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*desc.Server{readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestSelectServer_SecondaryPreferred_with_tags(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(SecondaryPreferredMode, desc.NewTagSet("a", "2"))

	result, err := SelectServer(subject, readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{readPrefTestSecondary2}, result)
}

func TestSelectServer_SecondaryPreferred_with_tags_that_do_not_match(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(SecondaryPreferredMode, desc.NewTagSet("a", "3"))

	result, err := SelectServer(subject, readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{readPrefTestPrimary}, result)
}

func TestSelectServer_SecondaryPreferred_with_tags_that_do_not_match_and_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(SecondaryPreferredMode, desc.NewTagSet("a", "3"))

	result, err := SelectServer(subject, readPrefTestCluster, []*desc.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 0)
}

func TestSelectServer_SecondaryPreferred_with_no_secondaries(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(SecondaryPreferredMode)

	result, err := SelectServer(subject, readPrefTestCluster, []*desc.Server{readPrefTestPrimary})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{readPrefTestPrimary}, result)
}

func TestSelectServer_SecondaryPreferred_with_no_secondaries_or_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(SecondaryPreferredMode)

	result, err := SelectServer(subject, readPrefTestCluster, []*desc.Server{})

	require.NoError(err)
	require.Len(result, 0)
}

func TestSelectServer_SecondaryPreferred_with_maxStaleness(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := NewWithMaxStaleness(SecondaryPreferredMode, time.Duration(90)*time.Second)

	result, err := SelectServer(subject, readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{readPrefTestSecondary2}, result)
}

func TestSelectServer_SecondaryPreferred_with_maxStaleness_and_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := NewWithMaxStaleness(SecondaryPreferredMode, time.Duration(90)*time.Second)

	result, err := SelectServer(subject, readPrefTestCluster, []*desc.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{readPrefTestSecondary2}, result)
}

func TestSelectServer_Secondary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(SecondaryMode)

	result, err := SelectServer(subject, readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*desc.Server{readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestSelectServer_Secondary_with_tags(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(SecondaryMode, desc.NewTagSet("a", "2"))

	result, err := SelectServer(subject, readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{readPrefTestSecondary2}, result)
}

func TestSelectServer_Secondary_with_tags_that_do_not_match(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(SecondaryMode, desc.NewTagSet("a", "3"))

	result, err := SelectServer(subject, readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 0)
}

func TestSelectServer_Secondary_with_no_secondaries(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(SecondaryMode)

	result, err := SelectServer(subject, readPrefTestCluster, []*desc.Server{readPrefTestPrimary})

	require.NoError(err)
	require.Len(result, 0)
}

func TestSelectServer_Secondary_with_maxStaleness(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := NewWithMaxStaleness(SecondaryMode, time.Duration(90)*time.Second)

	result, err := SelectServer(subject, readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{readPrefTestSecondary2}, result)
}

func TestSelectServer_Secondary_with_maxStaleness_and_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := NewWithMaxStaleness(SecondaryMode, time.Duration(90)*time.Second)

	result, err := SelectServer(subject, readPrefTestCluster, []*desc.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{readPrefTestSecondary2}, result)
}

func TestSelectServer_Nearest(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(NearestMode)

	result, err := SelectServer(subject, readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 3)
	require.Equal([]*desc.Server{readPrefTestPrimary, readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestSelectServer_Nearest_with_tags(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(NearestMode, desc.NewTagSet("a", "1"))

	result, err := SelectServer(subject, readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*desc.Server{readPrefTestPrimary, readPrefTestSecondary1}, result)
}

func TestSelectServer_Nearest_with_tags_that_do_not_match(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(NearestMode, desc.NewTagSet("a", "3"))

	result, err := SelectServer(subject, readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 0)
}

func TestSelectServer_Nearest_with_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(NearestMode)

	result, err := SelectServer(subject, readPrefTestCluster, []*desc.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*desc.Server{readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestSelectServer_Nearest_with_no_secondaries(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := New(NearestMode)

	result, err := SelectServer(subject, readPrefTestCluster, []*desc.Server{readPrefTestPrimary})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{readPrefTestPrimary}, result)
}

func TestSelectServer_Nearest_with_maxStaleness(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := NewWithMaxStaleness(NearestMode, time.Duration(90)*time.Second)

	result, err := SelectServer(subject, readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*desc.Server{readPrefTestPrimary, readPrefTestSecondary2}, result)
}

func TestSelectServer_Nearest_with_maxStaleness_and_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := NewWithMaxStaleness(NearestMode, time.Duration(90)*time.Second)

	result, err := SelectServer(subject, readPrefTestCluster, []*desc.Server{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*desc.Server{readPrefTestSecondary2}, result)
}

func TestSelectServer_Max_staleness_is_less_than_90_seconds(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := NewWithMaxStaleness(NearestMode, time.Duration(50)*time.Second)

	server := &desc.Server{
		Endpoint:          desc.Endpoint("localhost:27017"),
		HeartbeatInterval: time.Duration(10) * time.Second,
		LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
		LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
		Type:              desc.RSPrimary,
		Version:           desc.NewVersion(3, 4, 0),
	}
	cluster := &desc.Cluster{
		Type:    desc.ReplicaSetWithPrimary,
		Servers: []*desc.Server{server},
	}

	_, err := SelectServer(subject, cluster, cluster.Servers)

	require.Error(err)
}

func TestSelectServer_Max_staleness_is_too_low(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := NewWithMaxStaleness(NearestMode, time.Duration(100)*time.Second)

	server := &desc.Server{
		Endpoint:          desc.Endpoint("localhost:27017"),
		HeartbeatInterval: time.Duration(100) * time.Second,
		LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
		LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
		Type:              desc.RSPrimary,
		Version:           desc.NewVersion(3, 4, 0),
	}
	cluster := &desc.Cluster{
		Type:    desc.ReplicaSetWithPrimary,
		Servers: []*desc.Server{server},
	}

	_, err := SelectServer(subject, cluster, cluster.Servers)

	require.Error(err)
}
