package main

import (
	"fmt"
	"github.com/10gen/mongo-go-driver/auth"
	"github.com/10gen/mongo-go-driver/cluster"
	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/msg"
	"github.com/10gen/mongo-go-driver/server"
	"gopkg.in/mgo.v2/bson"
	"os"
	"time"
)

func main() {
	dialer := auth.NewDialer(conn.Dial, &auth.ScramSHA1Authenticator{
		DB:       "admin",
		Username: "root",
		Password: "root",
	})

	myCluster, err := cluster.New(cluster.WithSeedList("localhost:27017"),
		cluster.WithConnectionMode(cluster.SingleMode),
		cluster.WithServerOptions(server.WithConnectionDialer(dialer)))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	time.Sleep(time.Second)  // TODO: remove once server selection is fully implemented

	selectedServer, err := myCluster.SelectServer(cluster.WriteSelector())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	connection, err := selectedServer.Connection()

	var result bson.D
	conn.ExecuteCommand(connection, msg.NewCommand(msg.NextRequestID(), "test", false, bson.D{{"count", "test"}}), &result)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(result)

}
