package testutil

import (
	"context"
	"strings"
	"testing"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/yamgo/internal/testutil/helpers"
	"github.com/10gen/mongo-go-driver/yamgo/private/cluster"
	"github.com/10gen/mongo-go-driver/yamgo/private/conn"
	"github.com/10gen/mongo-go-driver/yamgo/private/msg"
	"github.com/10gen/mongo-go-driver/yamgo/readpref"
	"github.com/stretchr/testify/require"
)

// AutoCreateIndex creates an index in the test cluster.
func AutoCreateIndex(t *testing.T, keys []string) {
	indexes := bson.M{}
	for _, k := range keys {
		indexes[k] = 1
	}
	name := strings.Join(keys, "_")
	indexes = bson.M{"key": indexes, "name": name}

	createIndexCommand := bson.D{
		bson.NewDocElem("createIndexes", ColName(t)),
		bson.NewDocElem("indexes", []bson.M{indexes}),
	}

	request := msg.NewCommand(
		msg.NextRequestID(),
		DBName(t),
		false,
		createIndexCommand,
	)

	s, err := Cluster(t).SelectServer(context.Background(), cluster.WriteSelector(), readpref.Primary())
	require.NoError(t, err)
	c, err := s.Connection(context.Background())
	require.NoError(t, err)
	defer testhelpers.RequireNoErrorOnClose(t, c)

	err = conn.ExecuteCommand(context.Background(), c, request, &bson.D{})
	require.NoError(t, err)
}

// AutoDropCollection drops the collection in the test cluster.
func AutoDropCollection(t *testing.T) {
	DropCollection(t, DBName(t), ColName(t))
}

// DropCollection drops the collection in the test cluster.
func DropCollection(t *testing.T, dbname, colname string) {
	s, err := Cluster(t).SelectServer(context.Background(), cluster.WriteSelector(), readpref.Primary())
	require.NoError(t, err)
	c, err := s.Connection(context.Background())
	require.NoError(t, err)
	defer testhelpers.RequireNoErrorOnClose(t, c)

	err = conn.ExecuteCommand(
		context.Background(),
		c,
		msg.NewCommand(
			msg.NextRequestID(),
			dbname,
			false,
			bson.D{bson.NewDocElem("drop", colname)},
		),
		&bson.D{},
	)
	if err != nil && !strings.HasSuffix(err.Error(), "ns not found") {
		t.Fatal(err)
	}
}

func autoDropDB(t *testing.T, clstr *cluster.Cluster) {
	s, err := clstr.SelectServer(context.Background(), cluster.WriteSelector(), readpref.Primary())
	require.NoError(t, err)

	c, err := s.Connection(context.Background())
	require.NoError(t, err)
	defer testhelpers.RequireNoErrorOnClose(t, c)

	err = conn.ExecuteCommand(
		context.Background(),
		c,
		msg.NewCommand(
			msg.NextRequestID(),
			DBName(t),
			false,
			bson.D{bson.NewDocElem("dropDatabase", 1)},
		),
		&bson.D{},
	)
	require.NoError(t, err)
}

// AutoInsertDocs inserts the docs into the test cluster.
func AutoInsertDocs(t *testing.T, docs ...bson.D) {
	InsertDocs(t, DBName(t), ColName(t), docs...)
}

// InsertDocs inserts the docs into the test cluster.
func InsertDocs(t *testing.T, dbname, colname string, docs ...bson.D) {
	insertCommand := bson.D{
		bson.NewDocElem("insert", colname),
		bson.NewDocElem("documents", docs),
	}

	request := msg.NewCommand(
		msg.NextRequestID(),
		dbname,
		false,
		insertCommand,
	)

	s, err := Cluster(t).SelectServer(context.Background(), cluster.WriteSelector(), readpref.Primary())
	require.NoError(t, err)

	c, err := s.Connection(context.Background())
	require.NoError(t, err)
	defer testhelpers.RequireNoErrorOnClose(t, c)

	err = conn.ExecuteCommand(context.Background(), c, request, &bson.D{})
	require.NoError(t, err)
}

// EnableMaxTimeFailPoint turns on the max time fail point in the test cluster.
func EnableMaxTimeFailPoint(t *testing.T, s cluster.Server) error {
	c, err := s.Connection(context.Background())
	require.NoError(t, err)
	defer testhelpers.RequireNoErrorOnClose(t, c)

	return conn.ExecuteCommand(
		context.Background(),
		c,
		msg.NewCommand(
			msg.NextRequestID(),
			"admin",
			false,
			bson.D{
				bson.NewDocElem("configureFailPoint", "maxTimeAlwaysTimeOut"),
				bson.NewDocElem("mode", "alwaysOn"),
			},
		),
		&bson.D{},
	)
}

// DisableMaxTimeFailPoint turns off the max time fail point in the test cluster.
func DisableMaxTimeFailPoint(t *testing.T, s cluster.Server) {
	c, err := s.Connection(context.Background())
	require.NoError(t, err)
	defer testhelpers.RequireNoErrorOnClose(t, c)

	err = conn.ExecuteCommand(
		context.Background(),
		c,
		msg.NewCommand(msg.NextRequestID(),
			"admin",
			false,
			bson.D{
				bson.NewDocElem("configureFailPoint", "maxTimeAlwaysTimeOut"),
				bson.NewDocElem("mode", "off"),
			},
		),
		&bson.D{},
	)
	require.NoError(t, err)
}
