package conn

import (
	"context"
	"net"
)

// NetDialer creates a net.Conn.
type NetDialer func(context.Context, string, string) (net.Conn, error)

func dialNet(ctx context.Context, network, address string) (net.Conn, error) {
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		// TODO: disable nagle?
	}

	return conn, nil
}
