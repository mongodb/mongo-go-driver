package mongo

import (
	"context"
	"net"
	"sync/atomic"
	"testing"

	"time"

	"github.com/mongodb/mongo-go-driver/core/connstring"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
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

func TestClientOptions_doesNotAlterConnectionString(t *testing.T) {
	t.Parallel()

	cs := connstring.ConnString{}
	client, err := newClient(cs, ClientOpt.AppName("foobar"))
	require.NoError(t, err)
	if cs.AppName != "" {
		t.Errorf("Creating a new Client should not alter the original connection string, but it did. got %s; want <empty>", cs.AppName)
	}
	if client.connString.AppName != "foobar" {
		t.Errorf("Creating a new Client should alter the internal copy of the connection string, but it didn't. got %s; want %s", client.connString.AppName, "foobar")
	}
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
		MaxConnsPerHost(150).
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
		SSLClientCertificateKeyPassword(func() string { return "password" }).
		SSLInsecure(false).
		SSLCaFile("ca.pem").
		WString("majority").
		WNumber(3).
		Username("admin").
		WTimeout(2 * time.Second)

	client := new(Client)
	for opts.opt != nil {
		err := opts.opt(client)
		require.NoError(t, err)
		opts = opts.next
	}
}

func TestClientOptions_CustomDialer(t *testing.T) {
	td := &testDialer{d: &net.Dialer{}}
	opts := ClientOpt.Dialer(td)
	client, err := newClient(testutil.ConnString(t), opts)
	require.NoError(t, err)
	_, err = client.ListDatabases(context.Background(), nil)
	require.NoError(t, err)
	got := atomic.LoadInt32(&td.called)
	if got < 1 {
		t.Errorf("Custom dialer was not used when dialing new connections")
	}
}

type testDialer struct {
	called int32
	d      Dialer
}

func (td *testDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	atomic.AddInt32(&td.called, 1)
	return td.d.DialContext(ctx, network, address)
}
