package main

import (
	"log"

	"github.com/10gen/mongo-go-driver/yamgo/private/cluster"
	"github.com/kr/pretty"
)

func main() {
	monitor, err := cluster.StartMonitor()
	if err != nil {
		log.Fatalf("could not start cluster monitor: %v", err)
	}

	updates, _, _ := monitor.Subscribe()

	for desc := range updates {
		log.Printf("%# v", pretty.Formatter(desc))
	}

}
