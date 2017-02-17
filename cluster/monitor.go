package cluster

import (
	"errors"
	"sync"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/server"
)

// StartMonitor begins monitoring a cluster.
func StartMonitor(opts ...Option) (*Monitor, error) {
	cfg := newConfig(opts...)

	m := &Monitor{
		cfg:         cfg,
		subscribers: make(map[int64]chan *Desc),
		changes:     make(chan *server.Desc),
		desc:        &Desc{},
		fsm:         &monitorFSM{},
		servers:     make(map[conn.Endpoint]*server.Monitor),
	}

	if cfg.replicaSetName != "" {
		m.fsm.setName = cfg.replicaSetName
		m.fsm.Type = ReplicaSetNoPrimary
	}
	if cfg.mode == SingleMode {
		m.fsm.Type = Single
	}

	for _, ep := range cfg.seedList {
		canonicalized := ep.Canonicalize()
		m.fsm.addServer(canonicalized)
		m.startMonitoringEndpoint(canonicalized)
	}

	go func() {
		for change := range m.changes {
			// apply the change
			d := m.apply(change)
			m.descLock.Lock()
			m.desc = d
			m.descLock.Unlock()

			// send the change to all subscribers
			m.subscriberLock.Lock()
			for _, ch := range m.subscribers {
				select {
				case <-ch:
					// drain channel if not empty
				default:
					// do nothing if chan already empty
				}
				ch <- d
			}
			m.subscriberLock.Unlock()
		}
		m.subscriberLock.Lock()
		for id, ch := range m.subscribers {
			close(ch)
			delete(m.subscribers, id)
		}
		m.subscriptionsClosed = true
		m.subscriberLock.Unlock()
	}()

	return m, nil
}

// MonitorMode indicates the mode with which to run the monitor.
type MonitorMode uint8

// MonitorMode constants.
const (
	AutomaticMode MonitorMode = iota
	SingleMode
)

// Monitor continuously monitors the cluster for changes
// and reacts accordingly, adding or removing servers as necessary.
type Monitor struct {
	cfg      *config
	descLock sync.Mutex
	desc     *Desc

	changes chan *server.Desc
	fsm     *monitorFSM

	subscribers         map[int64]chan *Desc
	lastSubscriberID    int64
	subscriptionsClosed bool
	subscriberLock      sync.Mutex

	serversLock   sync.Mutex
	serversClosed bool
	servers       map[conn.Endpoint]*server.Monitor
}

// ServerMonitor gets the server monitor for the specified endpoint. It
// is imperative that this monitor not be stopped.
func (m *Monitor) ServerMonitor(endpoint conn.Endpoint) (*server.Monitor, bool) {
	m.serversLock.Lock()
	server, ok := m.servers[endpoint]
	m.serversLock.Unlock()
	return server, ok
}

// Stop turns the monitor off.
func (m *Monitor) Stop() {
	m.serversLock.Lock()
	m.serversClosed = true
	for endpoint, server := range m.servers {
		m.stopMonitoringEndpoint(endpoint, server)
	}
	m.serversLock.Unlock()

	close(m.changes)
}

// Subscribe returns a channel on which all updated ClusterDescs
// will be sent. The channel will have a buffer size of one, and
// will be pre-populated with the current ClusterDesc.
// Subscribe also returns a function that, when called, will close
// the subscription channel and remove it from the list of subscriptions.
func (m *Monitor) Subscribe() (<-chan *Desc, func(), error) {
	// create channel and populate with current state
	ch := make(chan *Desc, 1)
	m.descLock.Lock()
	ch <- m.desc
	m.descLock.Unlock()

	// add channel to subscribers
	m.subscriberLock.Lock()
	if m.subscriptionsClosed {
		return nil, nil, errors.New("cannot subscribe to monitor after stopping it")
	}
	m.lastSubscriberID++
	id := m.lastSubscriberID
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

// RequestImmediateCheck will send heartbeats to all the servers in the
// cluster right away, instead of waiting for the heartbeat timeout.
func (m *Monitor) RequestImmediateCheck() {
	m.serversLock.Lock()
	for _, server := range m.servers {
		server.RequestImmediateCheck()
	}
	m.serversLock.Unlock()
}

func (m *Monitor) startMonitoringEndpoint(endpoint conn.Endpoint) {
	if _, ok := m.servers[endpoint]; ok {
		// already monitoring this guy
		return
	}

	serverM, _ := server.StartMonitor(endpoint, m.cfg.serverOpts...)

	m.servers[endpoint] = serverM

	ch, _, _ := serverM.Subscribe()

	go func() {
		for d := range ch {
			m.changes <- d
		}
	}()
}

func (m *Monitor) stopMonitoringEndpoint(endpoint conn.Endpoint, server *server.Monitor) {
	server.Stop()
	delete(m.servers, endpoint)
}

func (m *Monitor) apply(d *server.Desc) *Desc {
	old := m.fsm.Desc
	m.fsm.apply(d)
	new := m.fsm.Desc

	diff := Diff(&old, &new)
	m.serversLock.Lock()
	if m.serversClosed {
		// maybe return an empty desc?
		return nil
	}
	for _, oldServer := range diff.RemovedServers {
		if sm, ok := m.servers[oldServer.Endpoint]; ok {
			m.stopMonitoringEndpoint(oldServer.Endpoint, sm)
		}
	}
	for _, newServer := range diff.AddedServers {
		if _, ok := m.servers[newServer.Endpoint]; !ok {
			m.startMonitoringEndpoint(newServer.Endpoint)
		}
	}
	m.serversLock.Unlock()
	return &new
}
