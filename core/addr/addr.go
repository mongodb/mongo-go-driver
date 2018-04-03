package addr

import (
	"net"
	"strings"
)

const defaultPort = "27017"

// Addr is a network address. It can either be an IP address or a DNS name.
type Addr string

// Network is the network protocol for this address. In most cases this will be
// "tcp" or "unix".
func (a Addr) Network() string {
	if strings.HasSuffix(string(a), "sock") {
		return "unix"
	}
	return "tcp"
}

// String is the canonical version of this address, e.g. localhost:27017,
// 1.2.3.4:27017, example.com:27017.
func (a Addr) String() string {
	// TODO: unicode case folding?
	s := strings.ToLower(string(a))
	if len(s) == 0 {
		return ""
	}
	if a.Network() != "unix" {
		_, _, err := net.SplitHostPort(s)
		if err != nil && strings.Contains(err.Error(), "missing port in address") {
			s += ":" + defaultPort
		}
	}

	return s
}

// Canonicalize creates a canonicalized address.
func (a Addr) Canonicalize() Addr {
	return Addr(a.String())
}
