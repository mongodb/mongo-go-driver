package conn

import (
	"context"
	"net"
	"strings"
)

// EndpointDialer is a function that dials an endpoint.
type EndpointDialer func(context.Context, Endpoint) (net.Conn, error)

// DialEndpoint dials an endpoint and returns a net.Conn.
func DialEndpoint(ctx context.Context, endpoint Endpoint) (net.Conn, error) {
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", string(endpoint))
	if err != nil {
		return nil, err
	}

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		// TODO: disable nagle?
	}

	return conn, nil
}

const defaultPort = "27017"

// Endpoint represents the location of a network resource or service.
type Endpoint string

// Canonicalize takes an endpoint and applies some transformations to it.
func (ep Endpoint) Canonicalize() Endpoint {
	// TODO: unicode case folding?
	s := strings.ToLower(string(ep))
	if !strings.Contains(s, "sock") {
		_, _, err := net.SplitHostPort(s)
		if err != nil && strings.Contains(err.Error(), "missing port in address") {
			s += ":" + defaultPort
		}
	}

	return Endpoint(s)
}
