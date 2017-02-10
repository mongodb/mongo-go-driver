package core

//go:generate go run spec_rtt_internal_test_generator.go

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/10gen/mongo-go-driver/core/desc"

	"sync"
	"time"
)

// StartServerMonitor returns a new ServerMonitor containing a channel
// that will send a ServerDesc everytime it is updated.
func StartServerMonitor(opts ServerOptions) (*ServerMonitor, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	opts.fillDefaults()

	done := make(chan struct{}, 1)
	m := &ServerMonitor{
		subscribers:    make(map[int]chan *desc.Server),
		done:           done,
		connectionOpts: opts.ConnectionOptions,
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
				timer.Reset(opts.HeartbeatInterval)
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

// ServerMonitor holds a channel that delivers updates to a server.
type ServerMonitor struct {
	subscribers         map[int]chan *desc.Server
	subscriptionsClosed bool
	subscriberLock      sync.Mutex

	conn           ConnectionCloser
	connectionOpts ConnectionOptions
	desc           *desc.Server
	descLock       sync.Mutex
	done           chan struct{}
	averageRTT     time.Duration
	averageRTTSet  bool
}

// Stop turns off the monitor.
func (m *ServerMonitor) Stop() {
	close(m.done)
}

// Subscribe returns a channel on which all updated server descriptions
// will be sent. The channel will have a buffer size of one, and
// will be pre-populated with the current ServerDesc.
// Subscribe also returns a function that, when called, will close
// the subscription channel and remove it from the list of subscriptions.
func (m *ServerMonitor) Subscribe() (<-chan *desc.Server, func(), error) {
	// create channel and populate with current state
	ch := make(chan *desc.Server, 1)
	m.descLock.Lock()
	ch <- m.desc
	m.descLock.Unlock()

	// add channel to subscribers
	m.subscriberLock.Lock()
	if m.subscriptionsClosed {
		return nil, nil, errors.New("Cannot subscribe to monitor after stopping it")
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
func (m *ServerMonitor) heartbeat() *desc.Server {
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
			conn, err := DialConnection(m.connectionOpts)
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

		d = buildServerDesc(m.connectionOpts.Endpoint, isMasterResult, buildInfoResult)
		d.SetAverageRTT(m.updateAverageRTT(delay))
	}

	if d == nil {
		d = &desc.Server{
			Endpoint:  m.connectionOpts.Endpoint,
			LastError: savedErr,
		}
	}

	return d
}

// updateAverageRTT calcuates the averageRTT of the server
// given its most recent RTT value
func (m *ServerMonitor) updateAverageRTT(delay time.Duration) time.Duration {
	if !m.averageRTTSet {
		m.averageRTT = delay
	} else {
		alpha := 0.2
		m.averageRTT = time.Duration(alpha*float64(delay) + (1-alpha)*float64(m.averageRTT))
	}
	return m.averageRTT
}

func buildServerDesc(endpoint desc.Endpoint, isMasterResult *isMasterResult, buildInfoResult *buildInfoResult) *desc.Server {
	desc := &desc.Server{
		Endpoint: endpoint,

		CanonicalEndpoint:  desc.Endpoint(isMasterResult.Me),
		ElectionID:         isMasterResult.ElectionID,
		LastWriteTimestamp: isMasterResult.LastWriteTimestamp,
		MaxBatchCount:      isMasterResult.MaxWriteBatchSize,
		MaxDocumentSize:    isMasterResult.MaxBSONObjectSize,
		MaxMessageSize:     isMasterResult.MaxMessageSizeBytes,
		Members:            isMasterResult.Members(),
		ServerType:         isMasterResult.ServerType(),
		SetName:            isMasterResult.SetName,
		SetVersion:         isMasterResult.SetVersion,
		Tags:               nil, // TODO: get tags
		WireVersion: desc.Range{
			Min: isMasterResult.MinWireVersion,
			Max: isMasterResult.MaxWireVersion,
		},
		Version: desc.NewVersionWithDesc(buildInfoResult.Version, buildInfoResult.VersionArray...),
	}

	if desc.CanonicalEndpoint == "" {
		desc.CanonicalEndpoint = endpoint
	}

	if !isMasterResult.OK {
		desc.LastError = fmt.Errorf("not ok")
	}

	return desc
}
