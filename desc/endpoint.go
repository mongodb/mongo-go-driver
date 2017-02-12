package desc

import (
	"io"
	"net"
	"strings"
)

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

// EndpointDialer dials an endpoint.
type EndpointDialer func(endpoint Endpoint) (io.ReadWriteCloser, error)
