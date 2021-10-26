package topology

import (
	"bytes"
	"context"
	"io"
	"math"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/drivertest"
	"go.mongodb.org/mongo-driver/x/mongo/driver/operation"
)

func makeHelloReply() []byte {
	didx, doc := bsoncore.AppendDocumentStart(nil)
	doc = bsoncore.AppendInt32Element(doc, "ok", 1)
	doc, _ = bsoncore.AppendDocumentEnd(doc, didx)
	return drivertest.MakeReply(doc)
}

var _ net.Conn = &slowConn{}

// slowConn is a net.Conn that delays all calls to Read() by the given delay duration. After the
// delay, Read() reads the given response. Calls to Write() reset the read buffer, so subsequent
// Read() calls read from the beginning of the provided response.
type slowConn struct {
	reader *bytes.Reader
	delay  time.Duration
}

func newSlowConn(response []byte, delay time.Duration) *slowConn {
	return &slowConn{
		reader: bytes.NewReader(response),
		delay:  delay,
	}
}

func (c *slowConn) Read(b []byte) (int, error) {
	time.Sleep(c.delay)
	return c.reader.Read(b)
}

func (c *slowConn) Write(b []byte) (int, error) {
	_, err := c.reader.Seek(0, io.SeekStart)
	return len(b), err
}

func (c *slowConn) Close() error                       { return nil }
func (c *slowConn) LocalAddr() net.Addr                { return nil }
func (c *slowConn) RemoteAddr() net.Addr               { return nil }
func (c *slowConn) SetDeadline(_ time.Time) error      { return nil }
func (c *slowConn) SetReadDeadline(_ time.Time) error  { return nil }
func (c *slowConn) SetWriteDeadline(_ time.Time) error { return nil }

func TestRTTMonitor(t *testing.T) {
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
				wantSamplesLen: 5,
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
			return newSlowConn(makeHelloReply(), 10*time.Millisecond), nil
		})
		rtt := newRTTMonitor(&rttConfig{
			interval: 10 * time.Second,
			createConnectionFn: func() (*connection, error) {
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

	t.Run("measures the average and minimum RTT", func(t *testing.T) {
		t.Parallel()

		dialer := DialerFunc(func(_ context.Context, _, _ string) (net.Conn, error) {
			return newSlowConn(makeHelloReply(), 10*time.Millisecond), nil
		})
		rtt := newRTTMonitor(&rttConfig{
			interval: 10 * time.Millisecond,
			createConnectionFn: func() (*connection, error) {
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
			func() bool { return rtt.getRTT() > 0 && rtt.getMinRTT() > 0 },
			1*time.Second,
			10*time.Millisecond,
			"expected getRTT() and getMinRTT() to return positive durations within 1 second")
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
