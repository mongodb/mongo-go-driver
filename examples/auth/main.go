package main

import (
	"fmt"
	"github.com/10gen/mongo-go-driver/auth"
	"github.com/10gen/mongo-go-driver/cluster"
	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/msg"
	"github.com/10gen/mongo-go-driver/server"
	"gopkg.in/mgo.v2/bson"
	"net"
	"os"
	"time"
)

func main() {
	authenticator := auth.ScramSHA1Authenticator{
		DB:       "admin",
		Username: "root",
		Password: "root",
	}

	dialer := func(endpoint conn.Endpoint, options ...conn.Option) (conn.ConnectionCloser, error) {
		return auth.Dial(
			func(endpoint conn.Endpoint, options ...conn.Option) (conn.ConnectionCloser, error) {
				return conn.Dial(endpoint, options...)
			}, &authenticator, endpoint)
	}
	myCluster, err := cluster.New(cluster.WithSeedList("localhost:27017"),
		cluster.WithConnectionMode(cluster.SingleMode),
		cluster.WithServerOptions(
			server.WithConnectionDialer(dialer),
			server.WithConnectionOptions(conn.WithEndpointDialer(func(endpoint conn.Endpoint) (net.Conn, error) {
				return conn.DialEndpoint(endpoint)
			}))))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	time.Sleep(time.Second)

	selectedServer, err := myCluster.SelectServer(func(cluster *cluster.Desc, candidates []*server.Desc) ([]*server.Desc, error) {
		return candidates, nil
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	connection, err := selectedServer.Connection()

	var result bson.D
	conn.ExecuteCommand(connection, msg.NewCommand(msg.NextRequestID(), "test", false, bson.D{{"count", "test"}}), result)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(result)

}
