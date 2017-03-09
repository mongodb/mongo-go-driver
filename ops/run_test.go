package ops_test

import (
	"context"
	"testing"

	"github.com/10gen/mongo-go-driver/bson"
	. "github.com/10gen/mongo-go-driver/ops"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Parallel()

	server := getServer()

	ctx := context.Background()
	result := bson.M{}

	err := Run(
		ctx,
		server,
		"admin",
		bson.D{{"getnonce", 1}},
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
		bson.D{{"ping", 1}},
		result,
	)

	require.NoError(t, err)
	require.Equal(t, float64(1), result["ok"], "Unable to ping MongoDB")

}
