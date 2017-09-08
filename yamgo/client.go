package yamgo

import (
	"context"

	"github.com/10gen/mongo-go-driver/yamgo/connstring"
	"github.com/10gen/mongo-go-driver/yamgo/private/cluster"
	"github.com/10gen/mongo-go-driver/yamgo/private/ops"
	"github.com/10gen/mongo-go-driver/yamgo/readpref"
)

// Client performs operations on a given cluster.
type Client struct {
	cluster    *cluster.Cluster
	connString connstring.ConnString
}

// NewClient creates a new client to connect to a cluster specified by the uri.
func NewClient(uri string) (*Client, error) {
	cs, err := connstring.Parse(uri)
	if err != nil {
		return nil, err
	}

	return NewClientFromConnString(cs)
}

// NewClientFromConnString creates a new client to connect to a cluster specified by the connection string.
func NewClientFromConnString(cs connstring.ConnString) (*Client, error) {
	clst, err := cluster.New(cluster.WithConnString(cs))
	if err != nil {
		return nil, err
	}

	return &Client{cluster: clst, connString: cs}, nil
}

// Database returns a handle for a given database.
func (client *Client) Database(name string) *Database {
	return &Database{client: client, name: name}
}

// ConnectionString returns the connection string of the cluster the client is connected to.
func (client *Client) ConnectionString() connstring.ConnString {
	return client.connString
}

func (client *Client) selectServer(ctx context.Context, selector cluster.ServerSelector,
	pref *readpref.ReadPref) (*ops.SelectedServer, error) {

	s, err := client.cluster.SelectServer(ctx, selector)
	if err != nil {
		return nil, err
	}

	selected := ops.SelectedServer{
		Server:   s,
		ReadPref: pref,
	}

	return &selected, nil
}
