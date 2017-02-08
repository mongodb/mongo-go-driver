package main

import (
	"github.com/10gen/mongo-go-driver/core"
	"github.com/10gen/mongo-go-driver/core/msg"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
	"fmt"
)

const databaseName = "test"
const collectionName = "test"

func main() {
	opts := core.ClusterOptions{
		Servers: []core.Endpoint{"localhost"},
	}

	monitor, err := core.StartClusterMonitor(opts)
	if err != nil {
		log.Fatalf("could not start cluster monitor: %v", err)
	}

	time.Sleep(1000000000) // TODO: Have to sleep since server selection doesn't block waiting for a server
	cluster := core.NewCluster(monitor)

	ss := func(clusterDesc *core.ClusterDesc, serverDescs []*core.ServerDesc) []*core.ServerDesc {
		return serverDescs
	}

	server := cluster.SelectServer(ss)

	connection, err := server.Connection()

	findCommand := bson.D{
		{"find", collectionName},
	}
	request := msg.NewCommand(
		msg.NextRequestID(),
		databaseName,
		false,
		findCommand,
	)

	var response core.FindResult

	err = core.ExecuteCommand(connection, request, &response)
	if err != nil {
		log.Fatalf("Failed to execute command: %v", err)
	}

	cursor := core.NewCursor(databaseName, collectionName, response.Cursor.FirstBatch, response.Cursor.ID, 0, connection)
	defer cursor.Close()

	var next bson.D

	for cursor.Next(&next) {
		fmt.Printf("Result: %v\n", next)
	}

	if cursor.Err() != nil {
		fmt.Printf("Error: %v\n", cursor.Err())
	}
}
