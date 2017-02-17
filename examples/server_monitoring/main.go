package main

import (
	"log"
	"time"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/server"
	"github.com/kr/pretty"
)

func main() {
	monitor, err := server.StartMonitor(
		conn.Endpoint("localhost:27017"),
		server.WithHeartbeatInterval(2*time.Second),
		server.WithConnectionOptions(
			conn.WithAppName("server_monitor test"),
		),
	)
	if err != nil {
		log.Fatalf("could not start server monitor: %v", err)
	}

	updates, _, _ := monitor.Subscribe()

	for desc := range updates {
		log.Printf("%# v", pretty.Formatter(desc))
	}
}
