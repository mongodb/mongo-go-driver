package server

//go:generate go run monitor_rtt_spec_internal_test_generator.go

import (
	"errors"
	"math/rand"

	"gopkg.in/mgo.v2/bson"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/desc"
	"github.com/10gen/mongo-go-driver/internal"
	"github.com/10gen/mongo-go-driver/internal/descutil"
	"github.com/10gen/mongo-go-driver/msg"

	"sync"
	"time"
)

// StartMonitor returns a new Monitor.
func StartMonitor(endpoint desc.Endpoint, opts ...Option) (*Monitor, error) {
	cfg := newConfig(opts...)

	done := make(chan struct{}, 1)
	m := &Monitor{
		endpoint:    endpoint,
		subscribers: make(map[int]chan *desc.Server),
		done:        done,
		connOpts:    cfg.connOpts,
		dialer:      cfg.dialer,
	}

	go func() {
		timer := time.NewTimer(0)
		for {
			select {
			case <-timer.C:
				// get an updated server description
				d := m.heartbeat()
				m.descLock.Lock()
				m.desc = d
				m.descLock.Unlock()

				// send the update to all subscribers
				m.subscriberLock.Lock()
				for _, ch := range m.subscribers {
					select {
					case <-ch:
						// drain the channel if not empty
					default:
						// do nothing if chan already empty
					}
					ch <- d
				}
				m.subscriberLock.Unlock()

				// restart the heartbeat timer
				timer.Stop()
				timer.Reset(cfg.heartbeatInterval)
			case <-done:
				timer.Stop()
				m.subscriberLock.Lock()
				for id, ch := range m.subscribers {
					close(ch)
					delete(m.subscribers, id)
				}
				m.subscriptionsClosed = true
				m.subscriberLock.Lock()
				return
			}
		}
	}()

	return m, nil
}

// Monitor holds a channel that delivers updates to a server.
type Monitor struct {
	subscribers         map[int]chan *desc.Server
	subscriptionsClosed bool
	subscriberLock      sync.Mutex

	conn          conn.ConnectionCloser
	connOpts      []conn.Option
	desc          *desc.Server
	descLock      sync.Mutex
	dialer        conn.Dialer
	done          chan struct{}
	endpoint      desc.Endpoint
	averageRTT    time.Duration
	averageRTTSet bool
}

// Stop turns off the monitor.
func (m *Monitor) Stop() {
	close(m.done)
}

// Subscribe returns a channel on which all updated server descriptions
// will be sent. The channel will have a buffer size of one, and
// will be pre-populated with the current ServerDesc.
// Subscribe also returns a function that, when called, will close
// the subscription channel and remove it from the list of subscriptions.
func (m *Monitor) Subscribe() (<-chan *desc.Server, func(), error) {
	// create channel and populate with current state
	ch := make(chan *desc.Server, 1)
	m.descLock.Lock()
	ch <- m.desc
	m.descLock.Unlock()

	// add channel to subscribers
	m.subscriberLock.Lock()
	if m.subscriptionsClosed {
		return nil, nil, errors.New("cannot subscribe to monitor after stopping it")
	}
	var id int
	for {
		_, found := m.subscribers[id]
		if !found {
			break
		}
		id = rand.Int()
	}
	m.subscribers[id] = ch
	m.subscriberLock.Unlock()

	unsubscribe := func() {
		m.subscriberLock.Lock()
		close(ch)
		delete(m.subscribers, id)
		m.subscriberLock.Unlock()
	}

	return ch, unsubscribe, nil
}

func (m *Monitor) heartbeat() *desc.Server {
	const maxRetryCount = 2
	var savedErr error
	var d *desc.Server
	for i := 1; i <= maxRetryCount; i++ {
		if m.conn == nil {
			// TODO: should this use the connection dialer from
			// the options? If so, it means authentication happens
			// for heartbeat connections as well, which makes
			// sharing a monitor in a multi-tenant arrangement
			// impossible.
			conn, err := conn.Dial(m.endpoint, m.connOpts...)
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
		isMasterResult, buildInfoResult, err := describeServer(m.conn)
		if err != nil {
			savedErr = err
			m.conn.Close()
			m.conn = nil
			continue
		}
		delay := time.Since(now)

		d = descutil.BuildServerDesc(m.endpoint, isMasterResult, buildInfoResult)
		d.SetAverageRTT(m.updateAverageRTT(delay))
	}

	if d == nil {
		d = &desc.Server{
			Endpoint:  m.endpoint,
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

func describeServer(c conn.Connection) (*internal.IsMasterResult, *internal.BuildInfoResult, error) {
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
	err := conn.ExecuteCommands(c, []msg.Request{isMasterReq, buildInfoReq}, []interface{}{&isMasterResult, &buildInfoResult})
	if err != nil {
		return nil, nil, err
	}

	return &isMasterResult, &buildInfoResult, nil
}
