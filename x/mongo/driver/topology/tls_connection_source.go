package topology

import (
	"crypto/tls"
	"net"
)

type tlsConnectionSource interface {
	Client(net.Conn, *tls.Config) *tls.Conn
}

type tlsConnectionSourceFn func(net.Conn, *tls.Config) *tls.Conn

func (t tlsConnectionSourceFn) Client(nc net.Conn, cfg *tls.Config) *tls.Conn {
	return t(nc, cfg)
}

var defaultTLSConnectionSource tlsConnectionSourceFn = func(nc net.Conn, cfg *tls.Config) *tls.Conn {
	return tls.Client(nc, cfg)
}
