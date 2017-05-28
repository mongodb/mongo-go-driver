package conn

import (
	"context"
	"net"
)

// Dialer dials a server according to the network and address.
type Dialer func(ctx context.Context, dialer *net.Dialer, network, address string) (net.Conn, error)

// Dial is the default Dialer.
func Dial(ctx context.Context, dialer *net.Dialer, network, address string) (net.Conn, error) {
	conn, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
