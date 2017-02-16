package conn_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gopkg.in/mgo.v2/bson"

	. "github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/msg"
	"github.com/stretchr/testify/require"
)

func createIntegrationTestConn() (Connection, error) {
	c, err := Dial(context.Background(), Endpoint(*host),
		WithAppName("mongo-go-driver-test"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed dialing mongodb server - ensure that one is running at %s: %v", *host, err)
	}
	return c, nil
}

func TestConn_ReadWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConn()
	if err != nil {
		t.Error(err)
	}

	isMasterRequest := msg.NewCommand(
		msg.NextRequestID(),
		"admin",
		true,
		bson.D{{"ismaster", 1}},
	)

	ctx, cancel := context.WithCancel(context.Background())
	err = subject.Write(ctx, isMasterRequest)
	require.NoError(t, err)
	_, err = subject.Read(ctx)
	require.NoError(t, err)
	cancel()

	require.NotEmpty(t, subject.Desc())
}

func TestConnection_Write_cancel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConn()
	if err != nil {
		t.Error(err)
	}

	isMasterRequest := msg.NewCommand(
		msg.NextRequestID(),
		"admin",
		true,
		bson.D{{"ismaster", 1}},
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = subject.Write(ctx, isMasterRequest)
	require.Error(t, err)
}

func TestConnection_Write_timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConn()
	if err != nil {
		t.Error(err)
	}

	isMasterRequest := msg.NewCommand(
		msg.NextRequestID(),
		"admin",
		true,
		bson.D{{"ismaster", 1}},
	)

	err = subject.Write(&timeoutContext{}, isMasterRequest)
	require.Error(t, err)
}

func TestConnection_Write_after_connection_is_dead(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConn()
	if err != nil {
		t.Error(err)
	}

	isMasterRequest := msg.NewCommand(
		msg.NextRequestID(),
		"admin",
		true,
		bson.D{{"ismaster", 1}},
	)

	err = subject.Write(&timeoutContext{}, isMasterRequest)
	require.Error(t, err)

	err = subject.Write(context.Background(), isMasterRequest)
	require.Error(t, err)
}

func TestConnection_Read_cancel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConn()
	if err != nil {
		t.Error(err)
	}

	isMasterRequest := msg.NewCommand(
		msg.NextRequestID(),
		"admin",
		true,
		bson.D{{"ismaster", 1}},
	)

	ctx, cancel := context.WithCancel(context.Background())
	err = subject.Write(ctx, isMasterRequest)
	require.NoError(t, err)

	cancel()
	_, err = subject.Read(ctx)
	require.Error(t, err)
}

func TestConnection_Read_timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConn()
	if err != nil {
		t.Error(err)
	}

	isMasterRequest := msg.NewCommand(
		msg.NextRequestID(),
		"admin",
		true,
		bson.D{{"ismaster", 1}},
	)

	err = subject.Write(context.Background(), isMasterRequest)
	require.NoError(t, err)

	_, err = subject.Read(&timeoutContext{})
	require.Error(t, err)
}

func TestConnection_Read_after_connection_is_dead(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	subject, err := createIntegrationTestConn()
	if err != nil {
		t.Error(err)
	}

	isMasterRequest := msg.NewCommand(
		msg.NextRequestID(),
		"admin",
		true,
		bson.D{{"ismaster", 1}},
	)

	err = subject.Write(context.Background(), isMasterRequest)
	require.NoError(t, err)

	_, err = subject.Read(&timeoutContext{})
	require.Error(t, err)

	_, err = subject.Read(context.Background())
	require.Error(t, err)
}

type timeoutContext struct {
	context.Context
}

func (c *timeoutContext) Deadline() (time.Time, bool) {
	return time.Now(), true
}

func (c *timeoutContext) Done() <-chan struct{} {
	return make(chan struct{})
}

func (c *timeoutContext) Err() error {
	return nil
}

func (c *timeoutContext) Value(key interface{}) interface{} {
	return nil
}
