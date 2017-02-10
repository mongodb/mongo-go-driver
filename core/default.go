package core

import (
	"time"

	"github.com/10gen/mongo-go-driver/core/desc"
	"github.com/10gen/mongo-go-driver/core/msg"
)

var defaultCodec = msg.NewWireProtocolCodec

var defaultConnectionDialer = DialConnection

var defaultEndpoint = desc.Endpoint("localhost:27017")

var defaultEndpointDialer = DialEndpoint

var defaultHeartbeatInterval = time.Duration(10) * time.Second

var defaultServerOptionsFactory = func(endpoint desc.Endpoint) ServerOptions {
	return ServerOptions{
		ConnectionOptions: ConnectionOptions{
			Endpoint: endpoint,
		},
		HeartbeatInterval: defaultHeartbeatInterval,
	}
}
