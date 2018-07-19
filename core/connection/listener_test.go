package connection

import (
	"net"
	"testing"
	"time"
)

func TestListener(t *testing.T) {
	readTimeout, writeTimeout := 25*time.Second, 25*time.Second
	network, addr := "tcp", "localhost:0"
	l, err := newListener(
		network, addr,
		WithReadTimeout(func(time.Duration) time.Duration { return readTimeout }),
		WithWriteTimeout(func(time.Duration) time.Duration { return writeTimeout }),
	)
	noerr(t, err)
	defer func() { _ = l.Close() }()

	if l.addr.String() == addr {
		t.Errorf("Address should be updated to match net.Listener address. got %s; want %s", l.addr, addr)
	}
	if l.readTimeout != readTimeout {
		t.Errorf("ReadTimeout should be set. got %v; want %v", l.readTimeout, readTimeout)
	}
	if l.writeTimeout != writeTimeout {
		t.Errorf("WriteTimeout should be set. got %v; want %v", l.writeTimeout, writeTimeout)
	}

	go func(addr string) {
		c, err := net.Dial(network, addr)
		noerr(t, err)
		_ = c.Close()
	}(l.addr.String())

	done := make(chan struct{}, 0)
	go func() {
		c, err := l.Accept()
		noerr(t, err)
		_ = c.Close()
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Errorf("Timed out waiting to accept connection from listener")
	}
}
