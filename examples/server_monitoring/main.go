package main

import (
	"log"
	"time"

	"github.com/10gen/mongo-go-driver/core"
	"github.com/kr/pretty"
)

func main() {
	opts := core.ServerOptions{
		ConnectionOptions: core.ConnectionOptions{
			AppName:  "server_monitor test",
			Endpoint: core.Endpoint("localhost:27017"),
		},
		HeartbeatInterval: time.Duration(2) * time.Second,
	}

	monitor, err := core.StartServerMonitor(opts)
	if err != nil {
		log.Fatalf("could not start server monitor: %v", err)
	}

	for desc := range monitor.C {
		log.Printf("%# v", pretty.Formatter(desc))
	}

}
