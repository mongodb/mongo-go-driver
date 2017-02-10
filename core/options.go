package core

import (
	"fmt"
	"time"

	"github.com/10gen/mongo-go-driver/core/desc"
	"github.com/10gen/mongo-go-driver/core/msg"
)

// ConnectionOptions are options for a connection.
type ConnectionOptions struct {
	AppName        string
	Codec          msg.Codec
	Endpoint       desc.Endpoint
	EndpointDialer EndpointDialer
}

func (o *ConnectionOptions) fillDefaults() {
	if o.Codec == nil {
		o.Codec = defaultCodec()
	}
	if o.EndpointDialer == nil {
		o.EndpointDialer = defaultEndpointDialer
	}
}

func (o *ConnectionOptions) validate() error {
	if o.Endpoint == "" {
		return fmt.Errorf("address cannot be empty")
	}

	return nil
}

// ServerOptions are options for connecting to a server.
type ServerOptions struct {
	ConnectionOptions

	ConnectionDialer  ConnectionDialer
	HeartbeatInterval time.Duration
}

func (o *ServerOptions) fillDefaults() {
	if o.ConnectionDialer == nil {
		o.ConnectionDialer = defaultConnectionDialer
	}
	if o.HeartbeatInterval == 0 {
		o.HeartbeatInterval = defaultHeartbeatInterval
	}
}

func (o *ServerOptions) validate() error {
	// TODO: validate heartbeat interval?

	return o.ConnectionOptions.validate()
}

// ConnectionDialer dials a connection.
type ConnectionDialer func(ConnectionOptions) (ConnectionCloser, error)

// ClusterOptions are options for connecting to a cluster.
type ClusterOptions struct {
	ConnectionMode       ClusterConnectionMode
	ReplicaSetName       string
	Servers              []desc.Endpoint
	ServerOptionsFactory ServerOptionsFactory
}

func (o *ClusterOptions) fillDefaults() {
	if o.ServerOptionsFactory == nil {
		o.ServerOptionsFactory = defaultServerOptionsFactory
	}
}

func (o *ClusterOptions) validate() error {
	if len(o.Servers) == 0 {
		return fmt.Errorf("no servers configured")
	}

	return nil
}

// ClusterConnectionMode determines the initial cluster type.
type ClusterConnectionMode uint8

// ClusterConnectionMode constants.
const (
	AutomaticMode ClusterConnectionMode = iota
	SingleMode
)

// ServerOptionsFactory returns ServerOptions given an Endpoint.
type ServerOptionsFactory func(desc.Endpoint) ServerOptions
