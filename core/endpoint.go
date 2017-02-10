package core

import (
	"io"
	"net"

	"github.com/10gen/mongo-go-driver/core/desc"
)

// DialEndpoint opens a connection with the endpoint. The is the default
// EndpointDialer.
func DialEndpoint(endpoint desc.Endpoint) (io.ReadWriteCloser, error) {
	return net.Dial("tcp", string(endpoint))
}

// EndpointDialer dials an endpoint.
type EndpointDialer func(endpoint desc.Endpoint) (io.ReadWriteCloser, error)
