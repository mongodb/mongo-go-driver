package conn

import (
	"context"
	"net"
)

// Dialer dials a server according to the network and address.
type Dialer func(ctx context.Context, network, address string) (net.Conn, error)

// Dial is the default Dialer.
func Dial(ctx context.Context, network, address string) (net.Conn, error) {
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
