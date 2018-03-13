package mongo

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/mongo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestClientOptions_simple(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	cs := testutil.ConnString(t)
	client, err := NewClientWithOptions(cs.String(), ClientOpt.AppName("foo"))
	require.NoError(t, err)

	require.Equal(t, "foo", client.connString.AppName)
}

func TestClientOptions_deferToConnString(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	cs := testutil.ConnString(t)
	uri := testutil.AddOptionsToURI(cs.String(), "appname=bar")

	client, err := NewClientWithOptions(uri, ClientOpt.AppName("foo"))
	require.NoError(t, err)

	require.Equal(t, "bar", client.connString.AppName)
}
