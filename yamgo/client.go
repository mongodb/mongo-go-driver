package yamgo

import (
	"context"

	"github.com/10gen/mongo-go-driver/yamgo/private/cluster"
	"github.com/10gen/mongo-go-driver/yamgo/connstring"
	"github.com/10gen/mongo-go-driver/yamgo/private/ops"
	"github.com/10gen/mongo-go-driver/yamgo/readpref"
)

type Client struct {
	cluster    *cluster.Cluster
	connString connstring.ConnString
}

func NewClient(uri string) (*Client, error) {
	cs, err := connstring.Parse(uri)
	if err != nil {
		return nil, err
	}

	return NewClientFromConnString(cs)
}

func NewClientFromConnString(cs connstring.ConnString) (*Client, error) {
	clst, err := cluster.New(cluster.WithConnString(cs))
	if err != nil {
		return nil, err
	}

	return &Client{cluster: clst, connString: cs}, nil
}

func (client *Client) Database(name string) *Database {
	return &Database{client: client, name: name}
}

func (client *Client) ConnectionString() connstring.ConnString {
	return client.connString
}

func (client *Client) RunCommand(ctx context.Context, db string, command interface{}, result interface{}) error {
	s, err := client.cluster.SelectServer(context.Background(), cluster.WriteSelector())
	if err != nil {
		return err
	}

	return ops.Run(
		ctx,
		&ops.SelectedServer{
			Server:   s,
			ReadPref: readpref.Primary(),
		},
		db,
		command,
		&result,
	)
}
