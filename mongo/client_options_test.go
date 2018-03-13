package mongo

import (
	"testing"

	"time"

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

func TestClientOptions_chainAll(t *testing.T) {
	t.Parallel()

	opts := ClientOpt.
		AppName("foo").
		AuthMechanism("MONGODB-X509").
		AuthMechanismProperties(map[string]string{"foo": "bar"}).
		AuthSource("$external").
		ConnectTimeout(500 * time.Millisecond).
		HeartbeatInterval(15 * time.Second).
		Hosts([]string{
			"mongodb://localhost:27018",
			"mongodb://localhost:27019"}).
		Journal(true).
		LocalThreshold(time.Second).
		MaxConnIdleTime(30 * time.Second).
		MaxIdleConnsPerHost(150).
		MaxIdleConnsPerHost(20).
		Password("supersecurepassword").
		ReadConcernLevel("majority").
		ReadPreference("secondary").
		ReadPreferenceTagSets([]map[string]string{
			{"nyc": "1"}}).
		ReplicaSet("foo").
		ServerSelectionTimeout(time.Second).
		Single(false).
		SocketTimeout(2 * time.Second).
		SSL(true).
		SSLClientCertificateKeyFile("client.pem").
		SSLInsecure(false).
		SSLCaFile("ca.pem").
		WString("majority").
		WNumber(3).
		Username("admin").
		WTimeout(2 * time.Second)

	for opts.opt != nil || opts.err != nil {
		require.NoError(t, opts.err)
		opts = opts.next
	}
}
