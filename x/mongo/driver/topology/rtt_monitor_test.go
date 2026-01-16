// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"bytes"
	"container/list"
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/mongo/address"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/drivertest"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/mnet"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/operation"
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
	t.Run("measures the average and minimum RTT", func(t *testing.T) {
		t.Parallel()

		dialer := DialerFunc(func(_ context.Context, _, _ string) (net.Conn, error) {
			return newMockSlowConn(makeHelloReply(), 10*time.Millisecond), nil
		})
		rtt := newRTTMonitor(&rttConfig{
			interval:       10 * time.Millisecond,
			connectTimeout: defaultConnectionTimeout,
			createConnectionFn: func() *connection {
				return newConnection("", WithDialer(func(Dialer) Dialer { return dialer }))
			},
			createOperationFn: func(conn *mnet.Connection) *operation.Hello {
				return operation.NewHello().Deployment(driver.SingleConnectionDeployment{C: conn})
			},
		})
		rtt.connect()
		defer rtt.disconnect()

		assert.Eventuallyf(
			t,
			func() bool { return rtt.EWMA() > 0 && rtt.Min() > 0 },
			1*time.Second,
			10*time.Millisecond,
			"expected EWMA() and Min() to return positive durations within 1 second")
		assert.True(
			t,
			rtt.EWMA() > 0,
			"expected EWMA() to return a positive duration, got %v",
			rtt.EWMA())
		assert.True(
			t,
			rtt.Min() > 0,
			"expected Min() to return a positive duration, got %v",
			rtt.Min())
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
			createOperationFn: func(conn *mnet.Connection) *operation.Hello {
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
			connectTimeout: defaultConnectionTimeout,
			interval:       10 * time.Millisecond,
			createConnectionFn: func() *connection {
				return newConnection("", WithDialer(func(Dialer) Dialer { return dialer }))
			},
			createOperationFn: func(conn *mnet.Connection) *operation.Hello {
				return operation.NewHello().Deployment(driver.SingleConnectionDeployment{C: conn})
			},
		})
		rtt.connect()
		defer rtt.disconnect()

		for i := 0; i < 3; i++ {
			assert.Eventuallyf(
				t,
				func() bool { return rtt.EWMA() > 0 },
				1*time.Second,
				10*time.Millisecond,
				"expected EWMA() to return a positive duration within 1 second")
			assert.Eventuallyf(
				t,
				func() bool { return rtt.Min() > 0 },
				1*time.Second,
				10*time.Millisecond,
				"expected Min() to return a positive duration within 1 second")

			rtt.reset()
		}
	})

	// GODRIVER-2464
	// Test that the RTT monitor can continue monitoring server RTTs after an operation gets stuck.
	// An operation can get stuck if the server or a proxy stops responding to requests on the RTT
	// connection but does not close the TCP socket, effectively creating an operation that will
	// never complete.
	t.Run("stuck operations time out", func(t *testing.T) {
		t.Parallel()

		// Start a goroutine that listens for and accepts TCP connections, reads requests, and
		// responds with {"ok": 1}. The first 2 connections simulate "stuck" connections and never
		// respond or close.
		l, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()

			for i := 0; ; i++ {
				conn, err := l.Accept()
				if err != nil {
					// The listen loop is cancelled by closing the listener, so there will always be
					// an error here. Log the error to make debugging easier in case of unexpected
					// errors.
					t.Logf("Accept error: %v", err)
					return
				}

				// Only close connections when the listener loop returns to prevent closing "stuck"
				// connections while the test is running.
				defer conn.Close()

				wg.Add(1)
				go func(i int) {
					defer wg.Done()

					buf := make([]byte, 256)
					for {
						if _, err := conn.Read(buf); err != nil {
							// The connection read/write loop is cancelled by closing the connection,
							// so may be an expected error here. Log the error to make debugging
							// easier in case of unexpected errors.
							t.Logf("Read error: %v", err)
							return
						}

						// For the first 2 connections, read the request but never respond and don't
						// close the connection. That simulates the behavior of a "stuck" connection.
						if i < 2 {
							return
						}

						// Delay for 10ms so that systems with limited timing granularity (e.g. some
						// older versions of Windows) can measure a non-zero latency.
						time.Sleep(10 * time.Millisecond)

						if _, err := conn.Write(makeHelloReply()); err != nil {
							// The connection read/write loop is cancelled by closing the connection,
							// so may be an expected error here. Log the error to make debugging
							// easier in case of unexpected errors.
							t.Logf("Write error: %v", err)
							return
						}
					}
				}(i)
			}
		}()

		rtt := newRTTMonitor(&rttConfig{
			interval:       10 * time.Millisecond,
			connectTimeout: 100 * time.Millisecond,
			createConnectionFn: func() *connection {
				return newConnection(address.Address(l.Addr().String()))
			},
			createOperationFn: func(conn *mnet.Connection) *operation.Hello {
				return operation.NewHello().Deployment(driver.SingleConnectionDeployment{C: conn})
			},
		})
		rtt.connect()

		assert.Eventuallyf(
			t,
			func() bool { return rtt.EWMA() > 0 },
			1*time.Second,
			10*time.Millisecond,
			"expected EWMA() to return a positive duration within 1 second")
		assert.Eventuallyf(
			t,
			func() bool { return rtt.Min() > 0 },
			1*time.Second,
			10*time.Millisecond,
			"expected Min() to return a positive duration within 1 second")

		rtt.disconnect()
		l.Close()
		wg.Wait()
	})
}

// makeArithmeticSamples will make an arithmetic sequence of time.Duration
// samples starting at the lower value as ms and ending at the upper value as
// ms. For example, if lower=1 and upder=4, then the return value will be
// [1ms, 2ms, 3ms, 4ms].
func makeArithmeticSamples(lower, upper int) []time.Duration {
	samples := []time.Duration{}
	for i := 0; i < upper-lower+1; i++ {
		samples = append(samples, time.Duration(lower+i)*time.Millisecond)
	}

	return samples
}

func TestRTTMonitor_appendMovingMin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		samples []time.Duration
		want    []time.Duration
	}{
		{
			name:    "singleton",
			samples: makeArithmeticSamples(1, 1),
			want:    makeArithmeticSamples(1, 1),
		},
		{
			name:    "multiplicity",
			samples: makeArithmeticSamples(1, 2),
			want:    makeArithmeticSamples(1, 2),
		},
		{
			name:    "exceed maxRTTSamples",
			samples: makeArithmeticSamples(1, 11),
			want:    makeArithmeticSamples(2, 11),
		},
		{
			name:    "exceed maxRTTSamples but only with negative values",
			samples: makeArithmeticSamples(-1, 9),
			want:    makeArithmeticSamples(0, 9),
		},
	}

	for _, test := range tests {
		test := test // capture the range variable

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			rtt := &rttMonitor{
				movingMin: list.New(),
			}

			for _, sample := range test.samples {
				rtt.appendMovingMin(sample)
			}

			pos := 0
			for e := rtt.movingMin.Front(); e != nil; e = e.Next() {
				assert.Equal(t, test.want[pos], e.Value)

				pos++
			}
		})
	}
}

func TestRTTMonitor_min(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		samples []time.Duration
		want    time.Duration
	}{
		{
			name:    "empty",
			samples: []time.Duration{},
			want:    0,
		},
		{
			name:    "one",
			samples: makeArithmeticSamples(1, 1),
			want:    0,
		},
		{
			name:    "two",
			samples: makeArithmeticSamples(1, 2),
			want:    1 * time.Millisecond,
		},
		{
			name:    "non-unit lower bound",
			samples: makeArithmeticSamples(2, 9),
			want:    2 * time.Millisecond,
		},
		{
			name:    "negative lower bound with 2 values",
			samples: []time.Duration{-1, 1},
			want:    0,
		},
		{
			name: "negative lower bound with 3 values",
			samples: []time.Duration{
				-1 * time.Millisecond,
				1 * time.Millisecond,
				2 * time.Millisecond,
			},
			want: 1 * time.Millisecond,
		},
		{
			name: "non-sequential",
			samples: []time.Duration{
				2 * time.Millisecond,
				1 * time.Millisecond,
			},
			want: 1 * time.Millisecond,
		},
	}

	for _, test := range tests {
		test := test // capture the range variable

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			rtt := &rttMonitor{
				movingMin: list.New(),
			}

			for _, sample := range test.samples {
				rtt.appendMovingMin(sample)
			}

			assert.Equal(t, test.want, rtt.min())
		})
	}
}

func TestRTTMonitor_stddev(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		samples []time.Duration
		want    float64
	}{
		{
			name:    "empty",
			samples: []time.Duration{},
			want:    0,
		},
		{
			name:    "one",
			samples: makeArithmeticSamples(1, 1),
			want:    0,
		},
		{
			name:    "below maxRTTSamples",
			samples: makeArithmeticSamples(1, 5),
			want:    0,
		},
		{
			name:    "equal maxRTTSamples",
			samples: makeArithmeticSamples(1, 10),
			want:    2.872281e+06,
		},
		{
			name:    "exceed maxRTTSamples",
			samples: makeArithmeticSamples(1, 15),
			want:    2.872281e+06,
		},
		{
			name: "non-sequential",
			samples: []time.Duration{
				2 * time.Millisecond,
				1 * time.Millisecond,
				4 * time.Millisecond,
				3 * time.Millisecond,
				7 * time.Millisecond,
				12 * time.Millisecond,
				6 * time.Millisecond,
				8 * time.Millisecond,
				5 * time.Millisecond,
				13 * time.Millisecond,
			},
			want: 3.806573e+06,
		},
	}

	for _, test := range tests {
		test := test // capture the range variable

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			rtt := &rttMonitor{
				movingMin: list.New(),
			}
			for _, sample := range test.samples {
				rtt.appendMovingMin(sample)
			}
			assert.Equal(t, test.want, float64(rtt.stddev()))
		})
	}
}
