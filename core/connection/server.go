// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package connection

import (
	"context"
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/mongodb/mongo-go-driver/core/address"
)

// ErrServerClosed is returned when a server has been closed or shutdown.
var ErrServerClosed = errors.New("server is closed")

// Handler handles an individual Connection. Returning signals that the Connection
// is no longer needed and can be closed.
type Handler interface {
	HandleConnection(Connection)
}

// HandlerFunc is an adapter to allow ordinary functions to be used as a connection handler.
type HandlerFunc func(Connection)

var _ Handler = HandlerFunc(nil)

// HandleConnection implements the Handler interface.
func (hf HandlerFunc) HandleConnection(c Connection) { hf(c) }

// Server is used to handle incoming Connections. It handles the boilerplate of accepting a
// Connection and cleaning it up after running a Handler. This also makes it easier to build
// higher level processors, like proxies, by handling the life cycle of the underlying
// connection.
//
// TODO(GODRIVER-269): Implement this.
type Server struct {
	Addr    address.Address
	Handler Handler

	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	mu        sync.Mutex
	listeners map[Listener]struct{}
	done      chan struct{}
}

// ListenAndServe listens on the network address srv.Addr and calls Serve to
// handle requests on incoming connections. If srv.Addr is blank, "localhost:27017"
// is used.
func (s *Server) ListenAndServe() error {
	addr := s.Addr.String()
	if addr == "" {
		addr = "27017"
	}
	l, err := newListener(
		s.Addr.Network(), addr,
		WithReadTimeout(func(time.Duration) time.Duration { return s.ReadTimeout }),
		WithWriteTimeout(func(time.Duration) time.Duration { return s.WriteTimeout }),
	)
	if err != nil {
		return err
	}

	return s.Serve(l)
}

// Serve accepts incoming connections on the Listener l, creating a new service
// goroutine for each. The service goroutines call srv.Handler and do not processing
// beforehand. When srv.Handler returns, the connection is closed.
func (s *Server) Serve(l Listener) error {
	defer func() { _ = l.Close() }()

	s.trackListener(l, true)
	defer s.trackListener(l, false)

	var tempDelay time.Duration

	for {
		log.Println("running and waiting!")
		c, err := l.Accept()
		if err != nil {
			select {
			case <-s.getDoneChan():
				return ErrServerClosed
			default:
			}
			if neterr, ok := err.(net.Error); ok && neterr.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}

				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				// TODO: We should log something here
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		tempDelay = 0
		go s.handle(c)
	}
}

func (s *Server) handle(c Connection) {
	s.Handler.HandleConnection(c)
	// TODO: We should do something with this error
	_ = c.Close()
}

// Shutdown shuts down the server by closing the active listeners. Shutdown
// does not handle or wait for all open connections to close and return before returning.
func (s *Server) Shutdown(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeDoneChanLocked()
	return s.closeListenersLocked()
}

func (s *Server) closeListenersLocked() error {
	var err error
	for l := range s.listeners {
		if e := l.Close(); e != nil && err == nil {
			err = e
		}
		delete(s.listeners, l)
	}
	return err
}

func (s *Server) trackListener(l Listener, add bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listeners == nil {
		s.listeners = make(map[Listener]struct{})
	}
	if add {
		// Reusing the server, remake done
		if len(s.listeners) == 0 {
			s.done = nil
		}
		s.listeners[l] = struct{}{}
	} else {
		delete(s.listeners, l)
	}
}

func (s *Server) getDoneChan() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getDoneChanLocked()
}

func (s *Server) getDoneChanLocked() chan struct{} {
	if s.done == nil {
		s.done = make(chan struct{})
	}
	return s.done
}

func (s *Server) closeDoneChanLocked() {
	ch := s.getDoneChanLocked()
	select {
	case <-ch: // already closed
	default:
		close(ch)
	}
}
