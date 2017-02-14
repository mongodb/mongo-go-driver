package cluster_test

import (
	"testing"
	"time"

	. "github.com/10gen/mongo-go-driver/cluster"
	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/readpref"
	"github.com/10gen/mongo-go-driver/server"
	"github.com/stretchr/testify/require"
)

var readPrefTestPrimary = &server.Desc{
	Endpoint:          conn.Endpoint("localhost:27017"),
	HeartbeatInterval: time.Duration(10) * time.Second,
	LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
	LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
	Type:              server.RSPrimary,
	Tags:              server.NewTagSet("a", "1"),
	Version:           conn.Version{Parts: []uint8{3, 4, 0}},
}
var readPrefTestSecondary1 = &server.Desc{
	Endpoint:          conn.Endpoint("localhost:27018"),
	HeartbeatInterval: time.Duration(10) * time.Second,
	LastWriteTime:     time.Date(2017, 2, 11, 13, 58, 0, 0, time.UTC),
	LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
	Type:              server.RSSecondary,
	Tags:              server.NewTagSet("a", "1"),
	Version:           conn.Version{Parts: []uint8{3, 4, 0}},
}
var readPrefTestSecondary2 = &server.Desc{
	Endpoint:          conn.Endpoint("localhost:27018"),
	HeartbeatInterval: time.Duration(10) * time.Second,
	LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
	LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
	Type:              server.RSSecondary,
	Tags:              server.NewTagSet("a", "2"),
	Version:           conn.Version{Parts: []uint8{3, 4, 0}},
}
var readPrefTestCluster = &Desc{
	Type:    ReplicaSetWithPrimary,
	Servers: []*server.Desc{readPrefTestPrimary, readPrefTestSecondary1, readPrefTestSecondary2},
}

func TestReadPrefSelector_Sharded(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Primary()

	s := &server.Desc{
		Endpoint:          conn.Endpoint("localhost:27017"),
		HeartbeatInterval: time.Duration(10) * time.Second,
		LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
		LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
		Type:              server.Mongos,
		Version:           conn.Version{Parts: []uint8{3, 4, 0}},
	}
	c := &Desc{
		Type:    Sharded,
		Servers: []*server.Desc{s},
	}

	result, err := ReadPrefSelector(subject)(c, c.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{s}, result)
}

func TestReadPrefSelector_Single(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Primary()

	s := &server.Desc{
		Endpoint:          conn.Endpoint("localhost:27017"),
		HeartbeatInterval: time.Duration(10) * time.Second,
		LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
		LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
		Type:              server.Mongos,
		Version:           conn.Version{Parts: []uint8{3, 4, 0}},
	}
	c := &Desc{
		Type:    Single,
		Servers: []*server.Desc{s},
	}

	result, err := ReadPrefSelector(subject)(c, c.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{s}, result)
}

func TestReadPrefSelector_Primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Primary()

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{readPrefTestPrimary}, result)
}

func TestReadPrefSelector_Primary_with_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Primary()

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, []*server.Desc{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Empty(result, 0)
}

func TestReadPrefSelector_PrimaryPreferred(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.PrimaryPreferred()

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{readPrefTestPrimary}, result)
}

func TestReadPrefSelector_PrimaryPreferred_ignores_tags(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.PrimaryPreferred(
		readpref.WithTags("a", "2"),
	)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{readPrefTestPrimary}, result)
}

func TestReadPrefSelector_PrimaryPreferred_with_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.PrimaryPreferred()

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, []*server.Desc{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*server.Desc{readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestReadPrefSelector_PrimaryPreferred_with_no_primary_and_tags(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.PrimaryPreferred(
		readpref.WithTags("a", "2"),
	)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, []*server.Desc{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{readPrefTestSecondary2}, result)
}

func TestReadPrefSelector_PrimaryPreferred_with_maxStaleness(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.PrimaryPreferred(
		readpref.WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{readPrefTestPrimary}, result)
}

func TestReadPrefSelector_PrimaryPreferred_with_maxStaleness_and_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.PrimaryPreferred(
		readpref.WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, []*server.Desc{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{readPrefTestSecondary2}, result)
}

func TestReadPrefSelector_SecondaryPreferred(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.SecondaryPreferred()

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*server.Desc{readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestReadPrefSelector_SecondaryPreferred_with_tags(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.SecondaryPreferred(
		readpref.WithTags("a", "2"),
	)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{readPrefTestSecondary2}, result)
}

func TestReadPrefSelector_SecondaryPreferred_with_tags_that_do_not_match(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.SecondaryPreferred(
		readpref.WithTags("a", "3"),
	)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{readPrefTestPrimary}, result)
}

func TestReadPrefSelector_SecondaryPreferred_with_tags_that_do_not_match_and_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.SecondaryPreferred(
		readpref.WithTags("a", "3"),
	)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, []*server.Desc{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 0)
}

func TestReadPrefSelector_SecondaryPreferred_with_no_secondaries(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.SecondaryPreferred()

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, []*server.Desc{readPrefTestPrimary})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{readPrefTestPrimary}, result)
}

func TestReadPrefSelector_SecondaryPreferred_with_no_secondaries_or_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.SecondaryPreferred()

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, []*server.Desc{})

	require.NoError(err)
	require.Len(result, 0)
}

func TestReadPrefSelector_SecondaryPreferred_with_maxStaleness(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.SecondaryPreferred(
		readpref.WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{readPrefTestSecondary2}, result)
}

func TestReadPrefSelector_SecondaryPreferred_with_maxStaleness_and_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.SecondaryPreferred(
		readpref.WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, []*server.Desc{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{readPrefTestSecondary2}, result)
}

func TestReadPrefSelector_Secondary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Secondary()

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*server.Desc{readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestReadPrefSelector_Secondary_with_tags(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Secondary(
		readpref.WithTags("a", "2"),
	)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{readPrefTestSecondary2}, result)
}

func TestReadPrefSelector_Secondary_with_tags_that_do_not_match(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Secondary(
		readpref.WithTags("a", "3"),
	)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 0)
}

func TestReadPrefSelector_Secondary_with_no_secondaries(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Secondary()

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, []*server.Desc{readPrefTestPrimary})

	require.NoError(err)
	require.Len(result, 0)
}

func TestReadPrefSelector_Secondary_with_maxStaleness(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Secondary(
		readpref.WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{readPrefTestSecondary2}, result)
}

func TestReadPrefSelector_Secondary_with_maxStaleness_and_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Secondary(
		readpref.WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, []*server.Desc{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{readPrefTestSecondary2}, result)
}

func TestReadPrefSelector_Nearest(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.New(readpref.NearestMode)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 3)
	require.Equal([]*server.Desc{readPrefTestPrimary, readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestReadPrefSelector_Nearest_with_tags(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Nearest(
		readpref.WithTags("a", "1"),
	)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*server.Desc{readPrefTestPrimary, readPrefTestSecondary1}, result)
}

func TestReadPrefSelector_Nearest_with_tags_that_do_not_match(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Nearest(
		readpref.WithTags("a", "3"),
	)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 0)
}

func TestReadPrefSelector_Nearest_with_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Nearest()

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, []*server.Desc{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*server.Desc{readPrefTestSecondary1, readPrefTestSecondary2}, result)
}

func TestReadPrefSelector_Nearest_with_no_secondaries(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Nearest()

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, []*server.Desc{readPrefTestPrimary})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{readPrefTestPrimary}, result)
}

func TestReadPrefSelector_Nearest_with_maxStaleness(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Nearest(
		readpref.WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, readPrefTestCluster.Servers)

	require.NoError(err)
	require.Len(result, 2)
	require.Equal([]*server.Desc{readPrefTestPrimary, readPrefTestSecondary2}, result)
}

func TestReadPrefSelector_Nearest_with_maxStaleness_and_no_primary(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Nearest(
		readpref.WithMaxStaleness(time.Duration(90) * time.Second),
	)

	result, err := ReadPrefSelector(subject)(readPrefTestCluster, []*server.Desc{readPrefTestSecondary1, readPrefTestSecondary2})

	require.NoError(err)
	require.Len(result, 1)
	require.Equal([]*server.Desc{readPrefTestSecondary2}, result)
}

func TestReadPrefSelector_Max_staleness_is_less_than_90_seconds(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Nearest(
		readpref.WithMaxStaleness(time.Duration(50) * time.Second),
	)

	s := &server.Desc{
		Endpoint:          conn.Endpoint("localhost:27017"),
		HeartbeatInterval: time.Duration(10) * time.Second,
		LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
		LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
		Type:              server.RSPrimary,
		Version:           conn.Version{Parts: []uint8{3, 4, 0}},
	}
	c := &Desc{
		Type:    ReplicaSetWithPrimary,
		Servers: []*server.Desc{s},
	}

	_, err := ReadPrefSelector(subject)(c, c.Servers)

	require.Error(err)
}

func TestReadPrefSelector_Max_staleness_is_too_low(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	subject := readpref.Nearest(
		readpref.WithMaxStaleness(time.Duration(100) * time.Second),
	)

	s := &server.Desc{
		Endpoint:          conn.Endpoint("localhost:27017"),
		HeartbeatInterval: time.Duration(100) * time.Second,
		LastWriteTime:     time.Date(2017, 2, 11, 14, 0, 0, 0, time.UTC),
		LastUpdateTime:    time.Date(2017, 2, 11, 14, 0, 2, 0, time.UTC),
		Type:              server.RSPrimary,
		Version:           conn.Version{Parts: []uint8{3, 4, 0}},
	}
	c := &Desc{
		Type:    ReplicaSetWithPrimary,
		Servers: []*server.Desc{s},
	}

	_, err := ReadPrefSelector(subject)(c, c.Servers)

	require.Error(err)
}
