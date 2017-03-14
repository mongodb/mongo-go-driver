package server

//go:generate go run monitor_rtt_spec_internal_test_generator.go

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/internal"
	"github.com/10gen/mongo-go-driver/model"
	"github.com/10gen/mongo-go-driver/msg"
)

const minHeartbeatFreqMS = 500 * time.Millisecond

// StartMonitor returns a new Monitor.
func StartMonitor(addr model.Addr, opts ...Option) (*Monitor, error) {
	cfg := newConfig(opts...)

	done := make(chan struct{}, 1)
	checkNow := make(chan struct{}, 1)
	m := &Monitor{
		cfg:  cfg,
		addr: addr,
		current: &model.Server{
			Addr: addr,
		},
		subscribers: make(map[int64]chan *model.Server),
		done:        done,
		checkNow:    checkNow,
	}

	var updateServer = func(heartbeatTimer, rateLimitTimer *time.Timer) {
		// wait if last heartbeat was less than
		// minHeartbeatFreqMS ago
		<-rateLimitTimer.C

		// get an updated server description
		model := m.heartbeat()
		m.currentLock.Lock()
		m.current = model
		m.currentLock.Unlock()

		// send the update to all subscribers
		m.subscriberLock.Lock()
		for _, ch := range m.subscribers {
			select {
			case <-ch:
				// drain the channel if not empty
			default:
				// do nothing if chan already empty
			}
			ch <- model
		}
		m.subscriberLock.Unlock()

		// restart the timers
		rateLimitTimer.Stop()
		rateLimitTimer.Reset(minHeartbeatFreqMS)
		heartbeatTimer.Stop()
		heartbeatTimer.Reset(cfg.heartbeatInterval)
	}

	go func() {
		heartbeatTimer := time.NewTimer(0)
		rateLimitTimer := time.NewTimer(0)
		for {
			select {
			case <-heartbeatTimer.C:
				updateServer(heartbeatTimer, rateLimitTimer)

			case <-checkNow:
				updateServer(heartbeatTimer, rateLimitTimer)

			case <-done:
				heartbeatTimer.Stop()
				rateLimitTimer.Stop()
				m.subscriberLock.Lock()
				for id, ch := range m.subscribers {
					close(ch)
					delete(m.subscribers, id)
				}
				m.subscriptionsClosed = true
				m.subscriberLock.Unlock()
				return
			}
		}
	}()

	return m, nil
}

// Monitor holds a channel that delivers updates to a server.
type Monitor struct {
	cfg *config

	subscribers         map[int64]chan *model.Server
	lastSubscriberID    int64
	subscriptionsClosed bool
	subscriberLock      sync.Mutex

	conn          conn.Connection
	current       *model.Server
	currentLock   sync.Mutex
	checkNow      chan struct{}
	done          chan struct{}
	addr          model.Addr
	averageRTT    time.Duration
	averageRTTSet bool
}

// Addr returns the address this monitor is monitoring.
func (m *Monitor) Addr() model.Addr {
	return m.addr
}

// Stop turns off the monitor.
func (m *Monitor) Stop() {
	close(m.done)
}

// Subscribe returns a channel on which all updated server descriptions
// will be sent. The channel will have a buffer size of one, and
// will be pre-populated with the current description.
// Subscribe also returns a function that, when called, will close
// the subscription channel and remove it from the list of subscriptions.
func (m *Monitor) Subscribe() (<-chan *model.Server, func(), error) {
	// create channel and populate with current state
	ch := make(chan *model.Server, 1)
	m.currentLock.Lock()
	ch <- m.current
	m.currentLock.Unlock()

	// add channel to subscribers
	m.subscriberLock.Lock()
	if m.subscriptionsClosed {
		close(ch)
		return nil, nil, errors.New("cannot subscribe to monitor after stopping it")
	}
	m.lastSubscriberID++
	id := m.lastSubscriberID
	m.subscribers[id] = ch
	m.subscriberLock.Unlock()

	unsubscribe := func() {
		m.subscriberLock.Lock()
		if !m.subscriptionsClosed {
			close(ch)
			delete(m.subscribers, id)
		}
		m.subscriberLock.Unlock()
	}

	return ch, unsubscribe, nil
}

// RequestImmediateCheck will cause the Monitor to send
// a heartbeat to the server right away, instead of waiting for
// the heartbeat timeout.
func (m *Monitor) RequestImmediateCheck() {
	select {
	case m.checkNow <- struct{}{}:
	default:
	}
}

func (m *Monitor) describeServer(ctx context.Context) (*internal.IsMasterResult, *internal.BuildInfoResult, error) {
	isMasterReq := msg.NewCommand(
		msg.NextRequestID(),
		"admin",
		true,
		bson.D{{Name: "ismaster", Value: 1}},
	)
	buildInfoReq := msg.NewCommand(
		msg.NextRequestID(),
		"admin",
		true,
		bson.D{{Name: "buildInfo", Value: 1}},
	)

	var isMasterResult internal.IsMasterResult
	var buildInfoResult internal.BuildInfoResult
	err := conn.ExecuteCommands(ctx, m.conn, []msg.Request{isMasterReq, buildInfoReq}, []interface{}{&isMasterResult, &buildInfoResult})
	if err != nil {
		return nil, nil, err
	}

	return &isMasterResult, &buildInfoResult, nil
}

func (m *Monitor) heartbeat() *model.Server {

	const maxRetryCount = 2
	var savedErr error
	var d *model.Server
	ctx := context.Background()
	for i := 1; i <= maxRetryCount; i++ {
		if m.conn != nil && m.conn.Expired() {
			m.conn.Close()
			m.conn = nil
		}

		if m.conn == nil {
			conn, err := m.cfg.dialer(ctx, m.addr, m.cfg.connOpts...)
			if err != nil {
				savedErr = err
				if conn != nil {
					conn.Close()
				}
				m.conn = nil
				continue
			}
			m.conn = conn
		}

		now := time.Now()
		isMasterResult, buildInfoResult, err := m.describeServer(ctx)
		if err != nil {
			savedErr = err
			m.conn.Close()
			m.conn = nil
			continue
		}
		delay := time.Since(now)

		d = model.BuildServer(m.addr, isMasterResult, buildInfoResult)
		d.SetAverageRTT(m.updateAverageRTT(delay))
		d.HeartbeatInterval = m.cfg.heartbeatInterval
	}

	if d == nil {
		d = &model.Server{
			Addr:      m.addr,
			LastError: savedErr,
		}
	}

	return d
}

// updateAverageRTT calcuates the averageRTT of the server
// given its most recent RTT value
func (m *Monitor) updateAverageRTT(delay time.Duration) time.Duration {
	if !m.averageRTTSet {
		m.averageRTT = delay
	} else {
		alpha := 0.2
		m.averageRTT = time.Duration(alpha*float64(delay) + (1-alpha)*float64(m.averageRTT))
	}
	return m.averageRTT
}
