package connection

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/mongodb/mongo-go-driver/core/address"
)

func TestServer(t *testing.T) {
	done := make(chan struct{})
	var handler HandlerFunc = func(_ Connection) {
		done <- struct{}{}
	}

	s := &Server{
		Addr:    address.Address("localhost:0"),
		Handler: handler,

		ReadTimeout:  25 * time.Second,
		WriteTimeout: 25 * time.Second,
	}

	numListeners := 1
	go func() {
		err := s.ListenAndServe()
		if err != ErrServerClosed {
			t.Errorf("Unexpected error from Shutdown. got %v; want %v", err, ErrServerClosed)
		}
		done <- struct{}{}
	}()

	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(10 * time.Microsecond)
	for {
		s.mu.Lock()
		if len(s.listeners) == numListeners {
			s.mu.Unlock()
			break
		}
		s.mu.Unlock()
		select {
		case <-ticker.C:
		case <-timeout:
			t.Errorf("Timed out waiting for listeners to be added")
		}
	}
	var iters int
	s.mu.Lock()
	for l := range s.listeners {
		go func(addr string) {
			c, err := net.Dial(s.Addr.Network(), addr)
			noerr(t, err)
			_ = c.Close()
		}(l.Addr().String())
		iters++
	}
	s.mu.Unlock()

	for range make([]struct{}, iters) {
		select {
		case <-done:
		case <-timeout:
			t.Errorf("Timed out waiting for connections")
		}
	}

	err := s.Shutdown(context.Background())
	noerr(t, err)
	select {
	case <-done:
	case <-timeout:
		t.Errorf("Timed out waiting for ListenAndServe to exit")
	}
}
