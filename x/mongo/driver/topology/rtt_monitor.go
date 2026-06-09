// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/mnet"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/operation"
)

const (
	rttAlphaValue             = 0.2
	minRTTSamplesForMovingMin = 2
	maxRTTSamplesForMovingMin = 10
)

type rttConfig struct {
	// The minimum interval between RTT measurements. The actual interval may be greater if running
	// the operation takes longer than the interval.
	interval time.Duration

	minRTTWindow       time.Duration
	createConnectionFn func() *connection
	connectTimeout     time.Duration
	createOperationFn  func(*mnet.Connection) *operation.Hello
}

type rttMonitor struct {
	mu sync.RWMutex // mu guards samples, offset, minRTT, averageRTT, and averageRTTSet

	// connMu guards connecting and disconnecting. This is necessary since
	// disconnecting will await the cancellation of a started connection. The
	// use case for rttMonitor.connect needs to be goroutine safe.
	connMu                 sync.Mutex
	averageRTT             time.Duration
	averageRTTSet          bool
	movingMin              *list.List
	minRTT                 time.Duration
	stddevRTT              time.Duration
	stddevSum              float64
	callsToAppendMovingMin int

	closeWg  sync.WaitGroup
	cfg      *rttConfig
	ctx      context.Context
	cancelFn context.CancelFunc
}

var _ driver.RTTMonitor = &rttMonitor{}

func newRTTMonitor(cfg *rttConfig) *rttMonitor {
	if cfg.interval <= 0 {
		panic("RTT monitor interval must be greater than 0")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &rttMonitor{
		cfg:       cfg,
		ctx:       ctx,
		cancelFn:  cancel,
		movingMin: list.New(),
	}
}

func (r *rttMonitor) connect() {
	r.connMu.Lock()
	defer r.connMu.Unlock()

	r.closeWg.Add(1)

	go func() {
		defer r.closeWg.Done()

		r.start()
	}()
}

func (r *rttMonitor) disconnect() {
	r.connMu.Lock()
	defer r.connMu.Unlock()

	r.cancelFn()

	// Wait for the existing connection to complete.
	r.closeWg.Wait()
}

func (r *rttMonitor) start() {
	var conn *connection
	defer func() {
		if conn != nil {
			// If the connection exists, we need to wait for it to be connected because
			// conn.connect() and conn.close() cannot be called concurrently. If the connection
			// wasn't successfully opened, its state was set back to disconnected, so calling
			// conn.close() will be a no-op.
			conn.closeConnectContext()
			conn.wait()
			_ = conn.close()
		}
	}()

	ticker := time.NewTicker(r.cfg.interval)
	defer ticker.Stop()

	for {
		conn = r.cfg.createConnectionFn()

		ctx, cancel := context.WithTimeout(r.ctx, r.cfg.connectTimeout)

		err := conn.connect(ctx)
		cancel()

		// Add an RTT sample from the new connection handshake and start a runHellos() loop if we
		// successfully established the new connection. Otherwise, close the connection and try to
		// create another new connection.
		if err == nil {
			r.runHellos(conn)
			r.addSample(conn.helloRTT)
		}

		// Close any connection here because we're either about to try to create another new
		// connection or we're about to exit the loop.
		_ = conn.close()

		// If a connection error happens quickly, always wait for the monitoring interval to try
		// to create a new connection to prevent creating connections too quickly.
		select {
		case <-ticker.C:
		case <-r.ctx.Done():
			return
		}
	}
}

// runHellos runs "hello" operations in a loop using the provided connection, measuring and
// recording the operation durations as RTT samples. If it encounters any errors, it returns.
func (r *rttMonitor) runHellos(conn *connection) {
	ticker := time.NewTicker(r.cfg.interval)
	defer ticker.Stop()

	for {
		// Assume that the connection establishment recorded the first RTT sample, so wait for the
		// first tick before trying to record another RTT sample.
		select {
		case <-ticker.C:
		case <-r.ctx.Done():
			return
		}

		// Create a Context with the operation timeout specified in the RTT monitor config. If a
		// timeout is not set in the RTT monitor config, default to the connection's
		// "connectTimeoutMS". The purpose of the timeout is to allow the RTT monitor to continue
		// monitoring server RTTs after an operation gets stuck. An operation can get stuck if the
		// server or a proxy stops responding to requests on the RTT connection but does not close
		// the TCP socket, effectively creating an operation that will never complete. We expect
		// that "connectTimeoutMS" provides at least enough time for a single round trip.
		ctx, cancel := context.WithTimeout(r.ctx, r.cfg.connectTimeout)

		start := time.Now()
		iconn := mnet.NewConnection(initConnection{conn})

		err := r.cfg.createOperationFn(iconn).Execute(ctx)
		cancel()
		if err != nil {
			return
		}
		// Only record a sample if the "hello" operation was successful. If it was not successful,
		// the operation may not have actually performed a complete round trip, so the duration may
		// be artificially short.
		r.addSample(time.Since(start))
	}
}

// reset sets the average, min, and stddev RTT to 0. This should only be called from the server monitor
// when an error occurs during a server check. Errors in the RTT monitor should not reset the RTTs.
func (r *rttMonitor) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.movingMin = list.New()
	r.averageRTT = 0
	r.averageRTTSet = false
	r.stddevSum = 0
	r.callsToAppendMovingMin = 0
}

// appendMovingMin will append the RTT to the movingMin list which tracks a
// minimum RTT within the last "minRTTSamplesForMovingMin" RTT samples.
func (r *rttMonitor) appendMovingMin(rtt time.Duration) {
	r.callsToAppendMovingMin++

	if r.movingMin == nil || rtt < 0 {
		return
	}

	if r.movingMin.Len() == maxRTTSamplesForMovingMin {
		r.movingMin.Remove(r.movingMin.Front())
	}

	r.movingMin.PushBack(rtt)

	// Collect a sum of stddevs over maxRTTSamplesForMovingMin calls, ignore if calls are less than max
	if r.callsToAppendMovingMin >= maxRTTSamplesForMovingMin {
		stddev := standardDeviationList(r.movingMin)
		r.stddevSum += stddev
	}
}

// min will return the minimum value in the movingMin list.
func (r *rttMonitor) min() time.Duration {
	if r.movingMin == nil || r.movingMin.Len() < minRTTSamplesForMovingMin {
		return 0
	}

	var min time.Duration
	for e := r.movingMin.Front(); e != nil; e = e.Next() {
		val := e.Value.(time.Duration)

		if min == 0 || val < min {
			min = val
		}
	}

	return min
}

// stddev will return the current moving stddev.
func (r *rttMonitor) stddev() time.Duration {
	var stddev time.Duration

	if r.callsToAppendMovingMin < maxRTTSamplesForMovingMin {
		return 0
	}

	// Get the number of times stddev was updated and calculate the average stddev
	frequency := (r.callsToAppendMovingMin + 1) - maxRTTSamplesForMovingMin
	stddev = time.Duration(r.stddevSum / float64(frequency))

	return stddev
}

func (r *rttMonitor) addSample(rtt time.Duration) {
	// Lock for the duration of this method. We're doing compuationally inexpensive work very infrequently, so lock
	// contention isn't expected.
	r.mu.Lock()
	defer r.mu.Unlock()

	r.appendMovingMin(rtt)
	r.minRTT = r.min()
	r.stddevRTT = r.stddev()

	if !r.averageRTTSet {
		r.averageRTT = rtt
		r.averageRTTSet = true
		return
	}

	r.averageRTT = time.Duration(rttAlphaValue*float64(rtt) + (1-rttAlphaValue)*float64(r.averageRTT))
}

// EWMA returns the exponentially weighted moving average observed round-trip time.
func (r *rttMonitor) EWMA() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.averageRTT
}

// Min returns the minimum observed round-trip time over the window period.
func (r *rttMonitor) Min() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.minRTT
}

// Stats returns stringified stats of the current state of the monitor.
func (r *rttMonitor) Stats() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return fmt.Sprintf(
		"network round-trip time stats: moving avg: %v, min: %v, moving stddev: %v",
		r.averageRTT,
		r.minRTT,
		r.stddevRTT)
}
