package ops_test

import (
	"context"
	"testing"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/yamgo/internal/testconfig"
	. "github.com/10gen/mongo-go-driver/yamgo/private/ops"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Parallel()
	testconfig.Integration(t)

	server := getServer(t)

	ctx := context.Background()
	result := bson.M{}

	err := Run(
		ctx,
		server,
		"admin",
		bson.D{bson.NewDocElem("getnonce", 1)},
		result,
	)
	require.NoError(t, err)
	require.Equal(t, float64(1), result["ok"])
	require.NotEqual(t, "", result["nonce"], "MongoDB returned empty nonce")

	result = bson.M{}
	err = Run(
		ctx,
		server,
		"admin",
		bson.D{bson.NewDocElem("ping", 1)},
		result,
	)

	require.NoError(t, err)
	require.Equal(t, float64(1), result["ok"], "Unable to ping MongoDB")

}
