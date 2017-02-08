package main

import (
	"github.com/10gen/mongo-go-driver/core"
	"github.com/10gen/mongo-go-driver/core/msg"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
)

func main() {
	opts := core.ClusterOptions{
		Servers: []core.Endpoint{"localhost"},
	}

	monitor, err := core.StartClusterMonitor(opts)
	if err != nil {
		log.Fatalf("could not start cluster monitor: %v", err)
	}

	time.Sleep(1000000000) // Note: Have to sleep since server selection doesn't block waiting for a server
	cluster := core.NewCluster(monitor)

	ss := func(clusterDesc *core.ClusterDesc, serverDescs []*core.ServerDesc) []*core.ServerDesc {
		return serverDescs
	}

	server := cluster.SelectServer(ss)

	connection, err := server.Connection()

	aggregateCommand := bson.D{
		{"aggregate", "test"},
		{"pipeline", []bson.D{}},
		{"cursor", bson.D{}},
	}
	request := msg.NewCommand(
		msg.NextRequestID(),
		"test",
		false,
		aggregateCommand,
	)

	response := bson.D{}

	err = core.ExecuteCommand(connection, request, &response)
	if err != nil {
		log.Fatalf("Failed to execute command: %v", err)
	}

	cursor := response.Map()["cursor"].(bson.D).Map()

	printBatch(cursor["firstBatch"].([]interface{}))

	cursorId := cursor["id"].(int64)

	for cursorId != 0 {
		getMoreCommand := bson.D{
			{"getMore", cursorId},
			{"collection", "test"},
		}
		getMoreRequest := msg.NewCommand(
			msg.NextRequestID(),
			"test",
			false,
			getMoreCommand,
		)

		response = bson.D{}

		err = core.ExecuteCommand(connection, getMoreRequest, &response)
		if err != nil {
			log.Fatalf("Failed to execute command: %v", err)
			return
		}

		cursor := response.Map()["cursor"].(bson.D).Map()
		printBatch(cursor["nextBatch"].([]interface{}))
		cursorId = cursor["id"].(int64)
	}
}

func printBatch(batch []interface{}) {
	for _, cur := range batch {
		log.Println(cur)
	}
}
