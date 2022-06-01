// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"bytes"
	"context"
	"io"
	"math"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/drivertest"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
)

func makeHelloReply() []byte {
	doc := bsoncore.NewDocumentBuilder().AppendInt32("ok", 1).Build()
	return drivertest.MakeReply(doc)
}

var _ net.Conn = &mockSlowConn{}

type mockSlowConn struct {
	reader *bytes.Reader
	delay  time.Duration
	closed atomic.Value
}

// newMockSlowConn returns a net.Conn that reads from the specified response after blocking for a
// delay duration. Calls to Write() reset the read buffer, so subsequent Read() calls read from the
// beginning of the provided response.
func newMockSlowConn(response []byte, delay time.Duration) *mockSlowConn {
	var closed atomic.Value
	closed.Store(false)

	return &mockSlowConn{
		reader: bytes.NewReader(response),
		delay:  delay,
		closed: closed,
	}
}

func (msc *mockSlowConn) Read(b []byte) (int, error) {
	time.Sleep(msc.delay)
	if msc.closed.Load().(bool) {
		return 0, io.ErrUnexpectedEOF
	}
	return msc.reader.Read(b)
}

func (msc *mockSlowConn) Write(b []byte) (int, error) {
	if msc.closed.Load().(bool) {
		return 0, io.ErrUnexpectedEOF
	}
	_, err := msc.reader.Seek(0, io.SeekStart)
	return len(b), err
}

// Close closes the mock connection. All subsequent calls to Read or Write return error
// io.ErrUnexpectedEOF. It is not safe to call Close concurrently with Read or Write.
func (msc *mockSlowConn) Close() error {
	msc.closed.Store(true)
	return nil
}

func (*mockSlowConn) LocalAddr() net.Addr                { return nil }
func (*mockSlowConn) RemoteAddr() net.Addr               { return nil }
func (*mockSlowConn) SetDeadline(_ time.Time) error      { return nil }
func (*mockSlowConn) SetReadDeadline(_ time.Time) error  { return nil }
func (*mockSlowConn) SetWriteDeadline(_ time.Time) error { return nil }

func TestRTTMonitor(t *testing.T) {
	t.Run("measures the average, minimum and 90th percentile RTT", func(t *testing.T) {
		t.Parallel()

		dialer := DialerFunc(func(_ context.Context, _, _ string) (net.Conn, error) {
			return newMockSlowConn(makeHelloReply(), 10*time.Millisecond), nil
		})
		rtt := newRTTMonitor(&rttConfig{
			interval: 10 * time.Millisecond,
			createConnectionFn: func() *connection {
				return newConnection("", WithDialer(func(Dialer) Dialer { return dialer }))
			},
			createOperationFn: func(conn driver.Connection) *operation.Hello {
				return operation.NewHello().Deployment(driver.SingleConnectionDeployment{C: conn})
			},
		})
		rtt.connect()
		defer rtt.disconnect()

		assert.Eventuallyf(
			t,
			func() bool { return rtt.getRTT() > 0 && rtt.getMinRTT() > 0 && rtt.getRTT90() > 0 },
			1*time.Second,
			10*time.Millisecond,
			"expected getRTT(), getMinRTT() and getRTT90() to return positive durations within 1 second")
		assert.True(
			t,
			rtt.getRTT() > 0,
			"expected getRTT() to return a positive duration, got %v",
			rtt.getRTT())
		assert.True(
			t,
			rtt.getMinRTT() > 0,
			"expected getMinRTT() to return a positive duration, got %v",
			rtt.getMinRTT())
		assert.True(
			t,
			rtt.getRTT90() > 0,
			"expected getRTT90() to return a positive duration, got %v",
			rtt.getRTT90())
	})

	t.Run("creates the correct size samples slice", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			desc           string
			interval       time.Duration
			wantSamplesLen int
		}{
			{
				desc:           "default",
				interval:       10 * time.Second,
				wantSamplesLen: 30,
			},
			{
				desc:           "min",
				interval:       10 * time.Minute,
				wantSamplesLen: 10,
			},
			{
				desc:           "max",
				interval:       1 * time.Millisecond,
				wantSamplesLen: 500,
			},
		}
		for _, tc := range cases {
			t.Run(tc.desc, func(t *testing.T) {
				rtt := newRTTMonitor(&rttConfig{
					interval:     tc.interval,
					minRTTWindow: 5 * time.Minute,
				})
				assert.Equal(t, tc.wantSamplesLen, len(rtt.samples), "expected samples length to match")
			})
		}
	})

	t.Run("can connect and disconnect repeatedly", func(t *testing.T) {
		t.Parallel()

		dialer := DialerFunc(func(_ context.Context, _, _ string) (net.Conn, error) {
			return newMockSlowConn(makeHelloReply(), 10*time.Millisecond), nil
		})
		rtt := newRTTMonitor(&rttConfig{
			interval: 10 * time.Second,
			createConnectionFn: func() *connection {
				return newConnection("", WithDialer(func(Dialer) Dialer {
					return dialer
				}))
			},
			createOperationFn: func(conn driver.Connection) *operation.Hello {
				return operation.NewHello().Deployment(driver.SingleConnectionDeployment{C: conn})
			},
		})
		for i := 0; i < 100; i++ {
			rtt.connect()
			rtt.disconnect()
		}
	})

	t.Run("works after reset", func(t *testing.T) {
		t.Parallel()

		dialer := DialerFunc(func(_ context.Context, _, _ string) (net.Conn, error) {
			return newMockSlowConn(makeHelloReply(), 10*time.Millisecond), nil
		})
		rtt := newRTTMonitor(&rttConfig{
			interval: 10 * time.Millisecond,
			createConnectionFn: func() *connection {
				return newConnection("", WithDialer(func(Dialer) Dialer { return dialer }))
			},
			createOperationFn: func(conn driver.Connection) *operation.Hello {
				return operation.NewHello().Deployment(driver.SingleConnectionDeployment{C: conn})
			},
		})
		rtt.connect()
		defer rtt.disconnect()

		for i := 0; i < 3; i++ {
			assert.Eventuallyf(
				t,
				func() bool { return rtt.getRTT() > 0 },
				1*time.Second,
				10*time.Millisecond,
				"expected getRTT() to return a positive duration within 1 second")
			assert.Eventuallyf(
				t,
				func() bool { return rtt.getMinRTT() > 0 },
				1*time.Second,
				10*time.Millisecond,
				"expected getMinRTT() to return a positive duration within 1 second")
			assert.Eventuallyf(
				t,
				func() bool { return rtt.getRTT90() > 0 },
				1*time.Second,
				10*time.Millisecond,
				"expected getRTT90() to return a positive duration within 1 second")
			rtt.reset()
		}
	})
}

func TestMin(t *testing.T) {
	cases := []struct {
		desc       string
		samples    []time.Duration
		minSamples int
		want       time.Duration
	}{
		{
			desc:       "Should return the min for minSamples = 0",
			samples:    []time.Duration{1, 0, 0, 0},
			minSamples: 0,
			want:       1,
		},
		{
			desc:       "Should return 0 for fewer than minSamples samples",
			samples:    []time.Duration{1, 0, 0, 0},
			minSamples: 2,
			want:       0,
		},
		{
			desc:       "Should return 0 for empty samples slice",
			samples:    []time.Duration{},
			minSamples: 0,
			want:       0,
		},
		{
			desc:       "Should return 0 for no valid samples",
			samples:    []time.Duration{0, 0, 0},
			minSamples: 0,
			want:       0,
		},
		{
			desc:       "Should return max int64 if all samples are max int64",
			samples:    []time.Duration{math.MaxInt64, math.MaxInt64, math.MaxInt64},
			minSamples: 0,
			want:       math.MaxInt64,
		},
		{
			desc:       "Should return the minimum if there are enough samples",
			samples:    []time.Duration{1 * time.Second, 100 * time.Millisecond, 150 * time.Millisecond, 0, 0, 0},
			minSamples: 3,
			want:       100 * time.Millisecond,
		},
		{
			desc:       "Should return 0 if there are are not enough samples",
			samples:    []time.Duration{1 * time.Second, 100 * time.Millisecond, 0, 0, 0, 0},
			minSamples: 3,
			want:       0,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			got := min(tc.samples, tc.minSamples)
			assert.Equal(t, tc.want, got, "unexpected result from min()")
		})
	}
}

func TestPercentile(t *testing.T) {
	cases := []struct {
		desc       string
		samples    []time.Duration
		minSamples int
		percentile float64
		want       time.Duration
	}{
		{
			desc:       "Should return 0 for fewer than minSamples samples",
			samples:    []time.Duration{1, 0, 0, 0},
			minSamples: 2,
			percentile: 90.0,
			want:       0,
		},
		{
			desc:       "Should return 0 for empty samples slice",
			samples:    []time.Duration{},
			minSamples: 0,
			percentile: 90.0,
			want:       0,
		},
		{
			desc:       "Should return 0 for no valid samples",
			samples:    []time.Duration{0, 0, 0},
			minSamples: 0,
			percentile: 90.0,
			want:       0,
		},
		{
			desc:       "First tertile when minSamples = 0",
			samples:    []time.Duration{1, 2, 3, 0, 0, 0},
			minSamples: 0,
			percentile: 33.34,
			want:       1,
		},
		{
			desc: "90th percentile when there are enough samples",
			samples: []time.Duration{
				100 * time.Millisecond,
				200 * time.Millisecond,
				300 * time.Millisecond,
				400 * time.Millisecond,
				500 * time.Millisecond,
				600 * time.Millisecond,
				700 * time.Millisecond,
				800 * time.Millisecond,
				900 * time.Millisecond,
				1 * time.Second,
				0, 0, 0},
			minSamples: 10,
			percentile: 90.0,
			want:       900 * time.Millisecond,
		},
		{
			desc: "10th percentile when there are enough samples",
			samples: []time.Duration{
				100 * time.Millisecond,
				200 * time.Millisecond,
				300 * time.Millisecond,
				400 * time.Millisecond,
				500 * time.Millisecond,
				600 * time.Millisecond,
				700 * time.Millisecond,
				800 * time.Millisecond,
				900 * time.Millisecond,
				1 * time.Second,
				0, 0, 0},
			minSamples: 10,
			percentile: 10.0,
			want:       100 * time.Millisecond,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			got := percentile(tc.percentile, tc.samples, tc.minSamples)
			assert.Equal(t, tc.want, got, "unexpected result from percentile()")
		})
	}
}
