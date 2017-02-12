package cluster

import (
	"errors"
	"math/rand"
	"sync"

	"github.com/10gen/mongo-go-driver/desc"
	"github.com/10gen/mongo-go-driver/server"
)

// StartMonitor begins monitoring a cluster.
func StartMonitor(opts ...Option) (*Monitor, error) {
	cfg := newConfig(opts...)

	m := &Monitor{
		subscribers: make(map[int]chan *desc.Cluster),
		changes:     make(chan *desc.Server),
		desc:        &desc.Cluster{},
		fsm:         &monitorFSM{},
		servers:     make(map[desc.Endpoint]*server.Monitor),
		serverOpts:  cfg.serverOpts,
	}

	if cfg.replicaSetName != "" {
		m.fsm.setName = cfg.replicaSetName
		m.fsm.ClusterType = desc.ReplicaSetNoPrimary
	}
	if cfg.connectionMode == SingleMode {
		m.fsm.ClusterType = desc.Single
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

// Monitor continuously monitors the cluster for changes
// and reacts accordingly, adding or removing servers as necessary.
type Monitor struct {
	descLock sync.Mutex
	desc     *desc.Cluster

	changes chan *desc.Server
	fsm     *monitorFSM

	subscribers         map[int]chan *desc.Cluster
	subscriptionsClosed bool
	subscriberLock      sync.Mutex

	serversLock   sync.Mutex
	serversClosed bool
	servers       map[desc.Endpoint]*server.Monitor
	serverOpts    []server.Option
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
func (m *Monitor) Subscribe() (<-chan *desc.Cluster, func(), error) {
	// create channel and populate with current state
	ch := make(chan *desc.Cluster, 1)
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

func (m *Monitor) startMonitoringEndpoint(endpoint desc.Endpoint) {
	if _, ok := m.servers[endpoint]; ok {
		// already monitoring this guy
		return
	}

	serverM, _ := server.StartMonitor(endpoint, m.serverOpts...)

	m.servers[endpoint] = serverM

	ch, _, _ := serverM.Subscribe()

	go func() {
		for d := range ch {
			m.changes <- d
		}
	}()
}

func (m *Monitor) stopMonitoringEndpoint(endpoint desc.Endpoint, server *server.Monitor) {
	server.Stop()
	delete(m.servers, endpoint)
}

func (m *Monitor) apply(d *desc.Server) *desc.Cluster {
	old := m.fsm.Cluster
	m.fsm.apply(d)
	new := m.fsm.Cluster

	diff := desc.DiffCluster(&old, &new)
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
