package connection

import (
	"net"
	"testing"
)

// bootstrapConnection creates a listener that will listen for a single connection
// on the return address. The user provided run function will be called with the accepted
// connection. The user is responsible for closing the connection.
func bootstrapConnection(t *testing.T, run func(net.Conn)) net.Addr {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Errorf("Could not set up a listener: %v", err)
		t.FailNow()
	}
	go func() {
		c, err := l.Accept()
		if err != nil {
			t.Errorf("Could not accept a connection: %v", err)
		}
		_ = l.Close()
		run(c)
	}()
	return l.Addr()
}
