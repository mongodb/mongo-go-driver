package main

import (
	"log"

	"github.com/craiggwilson/mongo-go-driver/core"
	"github.com/kr/pretty"
)

func main() {
	opts := core.ClusterOptions{
		Servers: []core.Endpoint{"localhost"},
	}

	monitor, err := core.StartClusterMonitor(opts)
	if err != nil {
		log.Fatalf("could not start cluster monitor: %v", err)
	}

	for desc := range monitor.C {
		log.Printf("%# v", pretty.Formatter(desc))
	}

}
