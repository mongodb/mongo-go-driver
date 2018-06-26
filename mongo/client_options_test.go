package mongo

import (
	"context"
	"net"
	"sync/atomic"
	"testing"

	"time"

	"github.com/mongodb/mongo-go-driver/core/connstring"
	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/tag"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil"
	"github.com/mongodb/mongo-go-driver/mongo/clientopt"
	"github.com/stretchr/testify/require"
)

func TestClientOptions_simple(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip()
	}

	cs := testutil.ConnString(t)
	client, err := NewClientWithOptions(cs.String(), clientopt.AppName("foo"))
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

	client, err := NewClientWithOptions(uri, clientopt.AppName("foo"))
	require.NoError(t, err)

	require.Equal(t, "bar", client.connString.AppName)
}

func TestClientOptions_doesNotAlterConnectionString(t *testing.T) {
	t.Parallel()

	cs := connstring.ConnString{}
	client, err := newClient(cs, clientopt.AppName("foobar"))
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
	readPrefMode, err := readpref.ModeFromString("secondary")
	require.NoError(t, err)
	rp, err := readpref.New(
		readPrefMode,
		readpref.WithTagSets(tag.NewTagSetsFromMaps([]map[string]string{{"nyc": "1"}})...),
		readpref.WithMaxStaleness(2*time.Second),
	)
	rc := readconcern.New(readconcern.Level("majority"))
	wc := writeconcern.New(
		writeconcern.J(true),
		writeconcern.WTagSet("majority"),
		writeconcern.W(3),
		writeconcern.WTimeout(2*time.Second),
	)
	require.NoError(t, err)
	opts := clientopt.BundleClient().
		AppName("foo").
		Auth(clientopt.Credential{
			AuthMechanism:           "MONGODB-X509",
			AuthMechanismProperties: map[string]string{"foo": "bar"},
			AuthSource:              "$external",
			Password:                "supersecurepassword",
			Username:                "admin",
		}).
		ConnectTimeout(500 * time.Millisecond).
		HeartbeatInterval(15 * time.Second).
		Hosts([]string{
			"mongodb://localhost:27018",
			"mongodb://localhost:27019"}).
		LocalThreshold(time.Second).
		MaxConnIdleTime(30 * time.Second).
		MaxConnsPerHost(150).
		MaxIdleConnsPerHost(20).
		ReadConcern(rc).
		ReadPreference(rp).
		ReplicaSet("foo").
		ServerSelectionTimeout(time.Second).
		Single(false).
		SocketTimeout(2 * time.Second).
		SSL(&clientopt.SSLOpt{
			Enabled:                      true,
			ClientCertificateKeyFile:     "client.pem",
			ClientCertificateKeyPassword: nil,
			Insecure:                     false,
			CaFile:                       "ca.pem",
		}).
		WriteConcern(wc)

	expectedClient := &clientopt.Client{
		TopologyOptions: nil,
		ConnString: connstring.ConnString{
			AppName:                 "foo",
			AuthMechanism:           "MONGODB-X509",
			AuthMechanismProperties: map[string]string{"foo": "bar"},
			AuthSource:              "$external",
			Username:                "admin",
			Password:                "supersecurepassword",
			ConnectTimeout:          500 * time.Millisecond,
			ConnectTimeoutSet:       true,
			HeartbeatInterval:       15 * time.Second,
			HeartbeatIntervalSet:    true,
			Hosts: []string{
				"mongodb://localhost:27018",
				"mongodb://localhost:27019",
			},
			LocalThresholdSet:         true,
			LocalThreshold:            time.Second,
			MaxConnIdleTime:           30 * time.Second,
			MaxConnIdleTimeSet:        true,
			MaxConnsPerHost:           150,
			MaxConnsPerHostSet:        true,
			MaxIdleConnsPerHost:       20,
			MaxIdleConnsPerHostSet:    true,
			ReplicaSet:                "foo",
			ServerSelectionTimeoutSet: true,
			ServerSelectionTimeout:    time.Second,
			Connect:                   connstring.AutoConnect,
			ConnectSet:                true,
			SocketTimeout:             2 * time.Second,
			SocketTimeoutSet:          true,
			SSL:                       true,
			SSLSet:                    true,
			SSLClientCertificateKeyFile:        "client.pem",
			SSLClientCertificateKeyFileSet:     true,
			SSLClientCertificateKeyPassword:    nil,
			SSLClientCertificateKeyPasswordSet: true,
			SSLInsecure:                        false,
			SSLInsecureSet:                     true,
			SSLCaFile:                          "ca.pem",
			SSLCaFileSet:                       true,
		},
		ReadConcern:    rc,
		ReadPreference: rp,
		WriteConcern:   wc,
	}

	client, err := opts.Unbundle(connstring.ConnString{})
	require.NoError(t, err)
	require.NotNil(t, client)
	require.Equal(t, expectedClient, client)
}

func TestClientOptions_CustomDialer(t *testing.T) {
	td := &testDialer{d: &net.Dialer{}}
	opts := clientopt.Dialer(td)
	client, err := newClient(testutil.ConnString(t), opts)
	require.NoError(t, err)
	err = client.Connect(context.Background())
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
