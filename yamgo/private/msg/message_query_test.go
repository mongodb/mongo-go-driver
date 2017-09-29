package msg_test

import (
	"testing"

	"github.com/10gen/mongo-go-driver/bson"
	. "github.com/10gen/mongo-go-driver/yamgo/private/msg"
	"github.com/stretchr/testify/require"
)

func TestWrapWithMeta(t *testing.T) {
	req := NewCommand(10, "admin", true, bson.M{"a": 1}).(*Query)

	buf, err := bson.Marshal(req.Query)
	require.NoError(t, err)
	var actual bson.D
	err = bson.Unmarshal(buf, &actual)
	require.NoError(t, err)
	expected := bson.D{
		bson.NewDocElem("a", 1),
	}
	require.Equal(t, expected, actual)

	AddMeta(req, map[string]interface{}{
		"$readPreference": bson.M{
			"mode": "secondary",
		},
	})

	buf, err = bson.Marshal(req.Query)
	require.NoError(t, err)
	err = bson.Unmarshal(buf, &actual)
	require.NoError(t, err)
	expected = bson.D{
		bson.NewDocElem("$query",
			bson.D{bson.NewDocElem("a", 1)},
		),
		bson.NewDocElem("$readPreference",
			bson.D{bson.NewDocElem("mode", "secondary")},
		),
	}
	require.Equal(t, expected, actual)
}
