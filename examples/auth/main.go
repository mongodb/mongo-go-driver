package main

import (
	"context"
	"log"
	"time"

	"github.com/10gen/mongo-go-driver/auth"
	"github.com/10gen/mongo-go-driver/cluster"
	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/msg"
	"github.com/10gen/mongo-go-driver/server"
	"gopkg.in/mgo.v2/bson"
)

func main() {
	dialer := auth.Dialer(conn.Dial, &auth.ScramSHA1Authenticator{
		DB:       "admin",
		Username: "root",
		Password: "root",
	})

	ctx := context.Background()
	myCluster, err := cluster.New(
		cluster.WithMode(cluster.SingleMode),
		cluster.WithServerOptions(
			server.WithConnectionDialer(dialer),
		),
	)
	if err != nil {
		log.Fatalf("could not start a cluster: %v", err)
	}

	time.Sleep(time.Second) // TODO: remove once server selection is fully implemented

	selectedServer, err := myCluster.SelectServer(ctx, cluster.WriteSelector())
	if err != nil {
		log.Fatalf("could not select a server: %v", err)
	}

	connection, err := selectedServer.Connection(ctx)
	if err != nil {
		log.Fatalf("could not get a connection: %s", err)
	}

	var result bson.D
	conn.ExecuteCommand(ctx, connection, msg.NewCommand(msg.NextRequestID(), "test", false, bson.D{{"count", "test"}}), &result)
	if err != nil {
		log.Fatalf("failed executing count command: %v", err)
	}

	log.Println(result)
}
