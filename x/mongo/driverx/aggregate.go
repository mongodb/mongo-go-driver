package driverx

import (
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
	"go.mongodb.org/mongo-driver/x/network/description"
)

//go:generate drivergen AggregateOperation aggregate.generated.go

type AggregateOperation struct {
	// ServerSelector sets the selector used to retrieve a server from a Deployment.
	serverSelector description.ServerSelector
	readPref       *readpref.ReadPref
	client         *session.Client
	clock          *session.ClusterClock
}
