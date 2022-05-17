// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/montanaflynn/stats"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
)

const (
	rttAlphaValue = 0.2
	minSamples    = 10
	maxSamples    = 500
)

type rttConfig struct {
	interval           time.Duration
	minRTTWindow       time.Duration // Window size to calculate minimum RTT over.
	createConnectionFn func() *connection
	createOperationFn  func(driver.Connection) *operation.Hello
}

type rttMonitor struct {
	mu            sync.RWMutex // mu guards samples, offset, minRTT, averageRTT, and averageRTTSet
	samples       []time.Duration
	offset        int
	minRTT        time.Duration
	RTT90         time.Duration
	averageRTT    time.Duration
	averageRTTSet bool

	closeWg  sync.WaitGroup
	cfg      *rttConfig
	ctx      context.Context
	cancelFn context.CancelFunc
}

func newRTTMonitor(cfg *rttConfig) *rttMonitor {
	if cfg.interval <= 0 {
		panic("RTT monitor interval must be greater than 0")
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Determine the number of samples we need to keep to store the minWindow of RTT durations. The
	// number of samples must be between [10, 500].
	numSamples := int(math.Max(minSamples, math.Min(maxSamples, float64((cfg.minRTTWindow)/cfg.interval))))

	return &rttMonitor{
		samples:  make([]time.Duration, numSamples),
		cfg:      cfg,
		ctx:      ctx,
		cancelFn: cancel,
	}
}

func (r *rttMonitor) connect() {
	r.closeWg.Add(1)
	go r.start()
}

func (r *rttMonitor) disconnect() {
	// Signal for the routine to stop.
	r.cancelFn()
	r.closeWg.Wait()
}

func (r *rttMonitor) start() {
	defer r.closeWg.Done()
	ticker := time.NewTicker(r.cfg.interval)
	defer ticker.Stop()

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

	for {
		conn = r.runHello(conn)

		select {
		case <-ticker.C:
		case <-r.ctx.Done():
			return
		}
	}
}

// runHello runs a "hello" operation using the provided connection, measures the duration, and adds
// the duration as an RTT sample and returns the connection used. If the provided connection is nil
// or closed, runHello tries to establish a new connection. If the "hello" operation returns an
// error, runHello closes the connection.
func (r *rttMonitor) runHello(conn *connection) *connection {
	if conn == nil || conn.closed() {
		conn := r.cfg.createConnectionFn()

		err := conn.connect(r.ctx)
		if err != nil {
			return nil
		}

		// If we just created a new connection, record the "hello" RTT from the new connection and
		// return the new connection. Don't run another "hello" command this interval because it's
		// now unnecessary.
		r.addSample(conn.helloRTT)
		return conn
	}

	start := time.Now()
	err := r.cfg.createOperationFn(initConnection{conn}).Execute(r.ctx)
	if err != nil {
		// Errors from the RTT monitor do not reset the RTTs or update the topology, so we close the
		// existing connection and recreate it on the next check.
		_ = conn.close()
		return nil
	}
	r.addSample(time.Since(start))

	return conn
}

// reset sets the average and min RTT to 0. This should only be called from the server monitor when an error
// occurs during a server check. Errors in the RTT monitor should not reset the RTTs.
func (r *rttMonitor) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.samples {
		r.samples[i] = 0
	}
	r.offset = 0
	r.minRTT = 0
	r.RTT90 = 0
	r.averageRTT = 0
	r.averageRTTSet = false
}

func (r *rttMonitor) addSample(rtt time.Duration) {
	// Lock for the duration of this method. We're doing compuationally inexpensive work very infrequently, so lock
	// contention isn't expected.
	r.mu.Lock()
	defer r.mu.Unlock()

	r.samples[r.offset] = rtt
	r.offset = (r.offset + 1) % len(r.samples)
	// Set the minRTT and 90th percentile RTT of all collected samples. Require at least 10 samples before
	// setting these to prevent noisy samples on startup from artificially increasing RTT and to allow the
	// calculation of a 90th percentile.
	r.minRTT = min(r.samples, minSamples)
	r.RTT90 = percentile(90.0, r.samples, minSamples)

	if !r.averageRTTSet {
		r.averageRTT = rtt
		r.averageRTTSet = true
		return
	}

	r.averageRTT = time.Duration(rttAlphaValue*float64(rtt) + (1-rttAlphaValue)*float64(r.averageRTT))
}

// min returns the minimum value of the slice of duration samples. Zero values are not considered
// samples and are ignored. If no samples or fewer than minSamples are found in the slice, min
// returns 0.
func min(samples []time.Duration, minSamples int) time.Duration {
	count := 0
	min := time.Duration(math.MaxInt64)
	for _, d := range samples {
		if d > 0 {
			count++
		}
		if d > 0 && d < min {
			min = d
		}
	}
	if count == 0 || count < minSamples {
		return 0
	}
	return min
}

// percentile returns the specified percentile value of the slice of duration samples. Zero values
// are not considered samples and are ignored. If no samples or fewer than minSamples are found
// in the slice, percentile returns 0.
func percentile(perc float64, samples []time.Duration, minSamples int) time.Duration {
	// Convert Durations to float64s.
	floatSamples := make([]float64, 0, len(samples))
	for _, sample := range samples {
		if sample > 0 {
			floatSamples = append(floatSamples, float64(sample))
		}
	}
	if len(floatSamples) == 0 || len(floatSamples) < minSamples {
		return 0
	}

	p, err := stats.Percentile(floatSamples, perc)
	if err != nil {
		panic(fmt.Errorf("x/mongo/driver/topology: error calculating %f percentile RTT: %v for samples:\n%v", perc, err, floatSamples))
	}
	return time.Duration(p)
}

// getRTT returns the exponentially weighted moving average observed round-trip time.
func (r *rttMonitor) getRTT() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.averageRTT
}

// getMinRTT returns the minimum observed round-trip time over the window period.
func (r *rttMonitor) getMinRTT() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.minRTT
}

// getRTT90 returns the 90th percentile observed round-trip time over the window period.
func (r *rttMonitor) getRTT90() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.RTT90
}
