// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build go1.17

package topology

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/mongo/address"
)

// fakeTLSConn is a minimal tlsConn stub that returns either an error or success on
// HandshakeContext, controlled by the parent fakeTLSSource.
type fakeTLSConn struct {
	net.Conn
	handshakeErr error
}

func (f *fakeTLSConn) HandshakeContext(context.Context) error { return f.handshakeErr }
func (f *fakeTLSConn) ConnectionState() tls.ConnectionState   { return tls.ConnectionState{} }

// fakeTLSSource records each Client() call and lets a test choose what handshake
// behavior to return on each call. seenCfgs lets tests verify which *tls.Config was
// actually used per attempt.
type fakeTLSSource struct {
	mu       sync.Mutex
	calls    int
	seenCfgs []*tls.Config
	// errSeq is consulted in order: errSeq[0] for call #1, errSeq[1] for call #2, etc.
	// Entries past the end of the slice are treated as nil (success).
	errSeq []error
}

func (f *fakeTLSSource) Client(nc net.Conn, cfg *tls.Config) tlsConn {
	f.mu.Lock()
	defer f.mu.Unlock()
	idx := f.calls
	f.calls++
	f.seenCfgs = append(f.seenCfgs, cfg)
	var err error
	if idx < len(f.errSeq) {
		err = f.errSeq[idx]
	}
	return &fakeTLSConn{Conn: nc, handshakeErr: err}
}

func newTestConnectionWithTLS(t *testing.T, src *fakeTLSSource, baseCfg *tls.Config, reloader func() (*tls.Config, error)) *connection {
	t.Helper()
	opts := []ConnectionOption{
		WithDialer(func(Dialer) Dialer {
			return DialerFunc(func(context.Context, string, string) (net.Conn, error) {
				return &net.TCPConn{}, nil
			})
		}),
		WithTLSConfig(func(*tls.Config) *tls.Config { return baseCfg }),
		withTLSConnectionSource(func(tlsConnectionSource) tlsConnectionSource { return src }),
	}
	if reloader != nil {
		opts = append(opts, WithTLSReloader(reloader))
	}
	return newConnection(address.Address("localhost:27017"), opts...)
}

// All test configs set InsecureSkipVerify so the post-handshake OCSP path is bypassed
// — the fake handshake never produces a real certificate chain to verify.

func TestTLSReload_NoReloaderReturnsOriginalError(t *testing.T) {
	handshakeErr := errors.New("bad cert")
	src := &fakeTLSSource{errSeq: []error{handshakeErr}}
	conn := newTestConnectionWithTLS(t, src, &tls.Config{InsecureSkipVerify: true}, nil)

	err := conn.connect(context.Background())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, handshakeErr), "expected wrapped handshake error, got %v", err)
	assert.Equal(t, 1, src.calls, "expected exactly one handshake attempt without a reloader")
}

func TestTLSReload_SuccessOnRetry(t *testing.T) {
	handshakeErr := errors.New("expired cert")
	src := &fakeTLSSource{errSeq: []error{handshakeErr}} // fail #1, succeed #2
	freshCfg := &tls.Config{ServerName: "fresh", InsecureSkipVerify: true}

	var reloadCalls atomic.Int32
	reloader := func() (*tls.Config, error) {
		reloadCalls.Add(1)
		return freshCfg, nil
	}
	conn := newTestConnectionWithTLS(t, src, &tls.Config{ServerName: "stale", InsecureSkipVerify: true}, reloader)

	err := conn.connect(context.Background())
	assert.NoError(t, err, "expected connect to succeed after reload retry")
	assert.Equal(t, 2, src.calls, "expected exactly two handshake attempts")
	assert.Equal(t, int32(1), reloadCalls.Load(), "reloader should be called exactly once")
	// The second attempt should have used the fresh config (after Clone, ServerName stays).
	assert.Equal(t, "fresh", src.seenCfgs[1].ServerName, "second handshake should use reloaded config")
	// The atomic pointer should now hold the fresh config.
	assert.Equal(t, freshCfg, conn.config.tlsConfigPtr.Load(), "atomic pointer should hold reloaded config")
}

func TestTLSReload_ReloadErrorPropagated(t *testing.T) {
	handshakeErr := errors.New("bad cert")
	reloadErr := errors.New("file vanished")
	src := &fakeTLSSource{errSeq: []error{handshakeErr}}
	reloader := func() (*tls.Config, error) { return nil, reloadErr }
	conn := newTestConnectionWithTLS(t, src, &tls.Config{InsecureSkipVerify: true}, reloader)

	err := conn.connect(context.Background())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, handshakeErr), "expected original error to be wrapped, got %v", err)
	// reloadErr is wrapped via %v (not %w) so we match on substring.
	assert.Contains(t, err.Error(), reloadErr.Error(), "expected reload error in message: %v", err)
	assert.Equal(t, 1, src.calls, "should not retry handshake when reload fails")
}

func TestTLSReload_RetryHandshakeAlsoFails(t *testing.T) {
	handshakeErr := errors.New("bad cert")
	retryErr := errors.New("still bad")
	src := &fakeTLSSource{errSeq: []error{handshakeErr, retryErr}}

	var reloadCalls atomic.Int32
	reloader := func() (*tls.Config, error) {
		reloadCalls.Add(1)
		return &tls.Config{ServerName: "fresh", InsecureSkipVerify: true}, nil
	}
	conn := newTestConnectionWithTLS(t, src, &tls.Config{InsecureSkipVerify: true}, reloader)

	err := conn.connect(context.Background())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, handshakeErr), "expected original error to be wrapped, got %v", err)
	assert.Contains(t, err.Error(), retryErr.Error(), "expected retry error in message: %v", err)
	assert.Equal(t, 2, src.calls, "expected exactly one retry, no infinite loop")
	assert.Equal(t, int32(1), reloadCalls.Load(), "reloader should be called exactly once")
}

func TestTLSReload_ConcurrentSingleFlight(t *testing.T) {
	// Every handshake fails so each goroutine takes the reload path. The reloader
	// should be invoked at most once; subsequent callers should reuse the published
	// pointer.
	const N = 16
	src := &fakeTLSSource{} // empty errSeq → first len(errSeq) calls fail (none here)
	src.errSeq = make([]error, 2*N)
	failure := errors.New("bad cert")
	for i := range src.errSeq {
		src.errSeq[i] = failure
	}

	var reloadCalls atomic.Int32
	freshCfg := &tls.Config{ServerName: "fresh", InsecureSkipVerify: true}
	reloader := func() (*tls.Config, error) {
		reloadCalls.Add(1)
		return freshCfg, nil
	}

	// WithTLSReloader allocates the atomic pointer + mutex once and captures them in
	// its returned closure, so every connection built from this same option slice
	// naturally shares the single-flight state. No manual fixup needed — that
	// sharing is precisely what this test verifies.
	baseOpts := []ConnectionOption{
		WithDialer(func(Dialer) Dialer {
			return DialerFunc(func(context.Context, string, string) (net.Conn, error) {
				return &net.TCPConn{}, nil
			})
		}),
		WithTLSConfig(func(*tls.Config) *tls.Config { return &tls.Config{ServerName: "stale", InsecureSkipVerify: true} }),
		withTLSConnectionSource(func(tlsConnectionSource) tlsConnectionSource { return src }),
		WithTLSReloader(reloader),
	}
	conns := make([]*connection, N)
	for i := 0; i < N; i++ {
		conns[i] = newConnection(address.Address("localhost:27017"), baseOpts...)
	}
	// Sanity check: pointer and mutex must be the same instance across all
	// connections, otherwise the single-flight property below is meaningless.
	for i := 1; i < N; i++ {
		assert.True(t, conns[i].config.tlsConfigPtr == conns[0].config.tlsConfigPtr,
			"tlsConfigPtr must be shared across connections built from the same option")
		assert.True(t, conns[i].config.tlsReloadMu == conns[0].config.tlsReloadMu,
			"tlsReloadMu must be shared across connections built from the same option")
	}

	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()
			_ = conns[i].connect(context.Background())
		}()
	}
	wg.Wait()

	// Reloader should fire exactly once because once one goroutine published the
	// fresh config, every later caller picks it up via the atomic pointer.
	assert.Equal(t, int32(1), reloadCalls.Load(),
		"expected single-flight reload, got %d invocations", reloadCalls.Load())
	assert.Equal(t, freshCfg, conns[0].config.tlsConfigPtr.Load(),
		"expected fresh config published to atomic pointer")
}
