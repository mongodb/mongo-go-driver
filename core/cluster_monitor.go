package core

//go:generate go run spec_cluster_monitor_internal_test_generator.go

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"

	"github.com/10gen/mongo-go-driver/core/desc"

	"gopkg.in/mgo.v2/bson"
)

// StartClusterMonitor begins monitoring a cluster.
func StartClusterMonitor(opts ClusterOptions) (*ClusterMonitor, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	opts.fillDefaults()

	m := &ClusterMonitor{
		subscribers:          make(map[int]chan *desc.Cluster),
		changes:              make(chan *desc.Server),
		desc:                 &desc.Cluster{},
		fsm:                  &clusterMonitorFSM{},
		servers:              make(map[desc.Endpoint]*ServerMonitor),
		serverOptionsFactory: opts.ServerOptionsFactory,
	}

	if opts.ReplicaSetName != "" {
		m.fsm.setName = opts.ReplicaSetName
		m.fsm.ClusterType = desc.ReplicaSetNoPrimary
	}
	if opts.ConnectionMode == SingleMode {
		m.fsm.ClusterType = desc.Single
	}

	for _, ep := range opts.Servers {
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

// ClusterMonitor continuously monitors the cluster for changes
// and reacts accordingly, adding or removing servers as necessary.
type ClusterMonitor struct {
	descLock sync.Mutex
	desc     *desc.Cluster

	changes chan *desc.Server
	fsm     *clusterMonitorFSM

	subscribers         map[int]chan *desc.Cluster
	subscriptionsClosed bool
	subscriberLock      sync.Mutex

	serversLock          sync.Mutex
	serversClosed        bool
	servers              map[desc.Endpoint]*ServerMonitor
	serverOptionsFactory ServerOptionsFactory
}

// Stop turns the monitor off.
func (m *ClusterMonitor) Stop() {
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
func (m *ClusterMonitor) Subscribe() (<-chan *desc.Cluster, func(), error) {
	// create channel and populate with current state
	ch := make(chan *desc.Cluster, 1)
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

func (m *ClusterMonitor) startMonitoringEndpoint(endpoint desc.Endpoint) {
	if _, ok := m.servers[endpoint]; ok {
		// already monitoring this guy
		return
	}

	opts := m.serverOptionsFactory(endpoint)
	serverM, _ := StartServerMonitor(opts)

	m.servers[endpoint] = serverM

	ch, _, _ := serverM.Subscribe()

	go func() {
		for d := range ch {
			m.changes <- d
		}
	}()
}

func (m *ClusterMonitor) stopMonitoringEndpoint(endpoint desc.Endpoint, server *ServerMonitor) {
	server.Stop()
	delete(m.servers, endpoint)
}

func (m *ClusterMonitor) apply(d *desc.Server) *desc.Cluster {
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

type clusterMonitorFSM struct {
	desc.Cluster

	maxElectionID bson.ObjectId
	maxSetVersion uint32
	setName       string
}

func (fsm *clusterMonitorFSM) apply(d *desc.Server) {

	newServers := make([]*desc.Server, len(fsm.Servers))
	copy(newServers, fsm.Servers)

	fsm.Cluster = desc.Cluster{
		ClusterType: fsm.ClusterType,
		Servers:     newServers,
	}

	if _, ok := fsm.findServer(d.Endpoint); !ok {
		return
	}

	switch fsm.ClusterType {
	case desc.UnknownClusterType:
		fsm.applyToUnknownClusterType(d)
	case desc.Sharded:
		fsm.applyToShardedClusterType(d)
	case desc.ReplicaSetNoPrimary:
		fsm.applyToReplicaSetNoPrimary(d)
	case desc.ReplicaSetWithPrimary:
		fsm.applyToReplicaSetWithPrimary(d)
	case desc.Single:
		fsm.applyToSingle(d)
	}
}

func (fsm *clusterMonitorFSM) applyToReplicaSetNoPrimary(d *desc.Server) {
	switch d.ServerType {
	case desc.Standalone, desc.Mongos:
		fsm.removeServerByEndpoint(d.Endpoint)
	case desc.RSPrimary:
		fsm.updateRSFromPrimary(d)
	case desc.RSSecondary, desc.RSArbiter, desc.RSMember:
		fsm.updateRSWithoutPrimary(d)
	case desc.UnknownServerType, desc.RSGhost:
		fsm.replaceServer(d)
	}
}

func (fsm *clusterMonitorFSM) applyToReplicaSetWithPrimary(d *desc.Server) {
	switch d.ServerType {
	case desc.Standalone, desc.Mongos:
		fsm.removeServerByEndpoint(d.Endpoint)
		fsm.checkIfHasPrimary()
	case desc.RSPrimary:
		fsm.updateRSFromPrimary(d)
	case desc.RSSecondary, desc.RSArbiter, desc.RSMember:
		fsm.updateRSWithPrimaryFromMember(d)
	case desc.UnknownServerType, desc.RSGhost:
		fsm.replaceServer(d)
		fsm.checkIfHasPrimary()
	}
}

func (fsm *clusterMonitorFSM) applyToShardedClusterType(d *desc.Server) {
	switch d.ServerType {
	case desc.Mongos, desc.UnknownServerType:
		fsm.replaceServer(d)
	case desc.Standalone, desc.RSPrimary, desc.RSSecondary, desc.RSArbiter, desc.RSMember, desc.RSGhost:
		fsm.removeServerByEndpoint(d.Endpoint)
	}
}

func (fsm *clusterMonitorFSM) applyToSingle(d *desc.Server) {
	switch d.ServerType {
	case desc.UnknownServerType:
		fsm.replaceServer(d)
	case desc.Standalone, desc.Mongos:
		if fsm.setName != "" {
			fsm.removeServerByEndpoint(d.Endpoint)
			return
		}

		fsm.replaceServer(d)
	case desc.RSPrimary, desc.RSSecondary, desc.RSArbiter, desc.RSMember, desc.RSGhost:
		if fsm.setName != "" && fsm.setName != d.SetName {
			fsm.removeServerByEndpoint(d.Endpoint)
			return
		}

		fsm.replaceServer(d)
	}
}

func (fsm *clusterMonitorFSM) applyToUnknownClusterType(d *desc.Server) {
	switch d.ServerType {
	case desc.Mongos:
		fsm.setType(desc.Sharded)
		fsm.replaceServer(d)
	case desc.RSPrimary:
		fsm.updateRSFromPrimary(d)
	case desc.RSSecondary, desc.RSArbiter, desc.RSMember:
		fsm.setType(desc.ReplicaSetNoPrimary)
		fsm.updateRSWithoutPrimary(d)
	case desc.Standalone:
		fsm.updateUnknownWithStandalone(d)
	case desc.UnknownServerType, desc.RSGhost:
		fsm.replaceServer(d)
	}
}

func (fsm *clusterMonitorFSM) checkIfHasPrimary() {
	if _, ok := fsm.findPrimary(); ok {
		fsm.setType(desc.ReplicaSetWithPrimary)
	} else {
		fsm.setType(desc.ReplicaSetNoPrimary)
	}
}

func (fsm *clusterMonitorFSM) updateRSFromPrimary(d *desc.Server) {
	if fsm.setName == "" {
		fsm.setName = d.SetName
	} else if fsm.setName != d.SetName {
		fsm.removeServerByEndpoint(d.Endpoint)
		fsm.checkIfHasPrimary()
		return
	}

	if d.SetVersion != 0 && d.ElectionID != "" {
		if fsm.maxSetVersion > d.SetVersion || fsm.maxElectionID > d.ElectionID {
			fsm.replaceServer(&desc.Server{
				Endpoint:  d.Endpoint,
				LastError: fmt.Errorf("was a primary, but its set version or election id is stale"),
			})
			fsm.checkIfHasPrimary()
			return
		}

		fsm.maxElectionID = d.ElectionID
	}

	if d.SetVersion > fsm.maxSetVersion {
		fsm.maxSetVersion = d.SetVersion
	}

	if j, ok := fsm.findPrimary(); ok {
		fsm.setServer(j, &desc.Server{
			Endpoint:  fsm.Servers[j].Endpoint,
			LastError: fmt.Errorf("was a primary, but a new primary was discovered"),
		})
	}

	fsm.replaceServer(d)

	for j := len(fsm.Servers) - 1; j >= 0; j-- {
		server := fsm.Servers[j]
		found := false
		for _, member := range d.Members {
			if member == server.Endpoint {
				found = true
				break
			}
		}
		if !found {
			fsm.removeServer(j)
		}
	}

	for _, member := range d.Members {
		if _, ok := fsm.findServer(member); !ok {
			fsm.addServer(member)
		}
	}

	fsm.checkIfHasPrimary()
}

func (fsm *clusterMonitorFSM) updateRSWithPrimaryFromMember(d *desc.Server) {
	if fsm.setName != d.SetName {
		fsm.removeServerByEndpoint(d.Endpoint)
		fsm.checkIfHasPrimary()
		return
	}

	if d.Endpoint != d.CanonicalEndpoint {
		fsm.removeServerByEndpoint(d.Endpoint)
		fsm.checkIfHasPrimary()
		return
	}

	fsm.replaceServer(d)

	if _, ok := fsm.findPrimary(); !ok {
		fsm.setType(desc.ReplicaSetNoPrimary)
	}
}

func (fsm *clusterMonitorFSM) updateRSWithoutPrimary(d *desc.Server) {
	if fsm.setName == "" {
		fsm.setName = d.SetName
	} else if fsm.setName != d.SetName {
		fsm.removeServerByEndpoint(d.Endpoint)
		return
	}

	for _, member := range d.Members {
		if _, ok := fsm.findServer(member); !ok {
			fsm.addServer(member)
		}
	}

	if d.Endpoint != d.CanonicalEndpoint {
		fsm.removeServerByEndpoint(d.Endpoint)
		return
	}

	fsm.replaceServer(d)
}

func (fsm *clusterMonitorFSM) updateUnknownWithStandalone(d *desc.Server) {
	if len(fsm.Servers) > 1 {
		fsm.removeServerByEndpoint(d.Endpoint)
		return
	}

	fsm.setType(desc.Single)
	fsm.replaceServer(d)
}

func (fsm *clusterMonitorFSM) addServer(endpoint desc.Endpoint) {
	fsm.Servers = append(fsm.Servers, &desc.Server{
		Endpoint: endpoint,
	})
}

func (fsm *clusterMonitorFSM) findPrimary() (int, bool) {
	for i, s := range fsm.Servers {
		if s.ServerType == desc.RSPrimary {
			return i, true
		}
	}

	return 0, false
}

func (fsm *clusterMonitorFSM) findServer(endpoint desc.Endpoint) (int, bool) {
	for i, s := range fsm.Servers {
		if endpoint == s.Endpoint {
			return i, true
		}
	}

	return 0, false
}

func (fsm *clusterMonitorFSM) removeServer(i int) {
	fsm.Servers = append(fsm.Servers[:i], fsm.Servers[i+1:]...)
}

func (fsm *clusterMonitorFSM) removeServerByEndpoint(endpoint desc.Endpoint) {
	if i, ok := fsm.findServer(endpoint); ok {
		fsm.removeServer(i)
	}
}

func (fsm *clusterMonitorFSM) replaceServer(d *desc.Server) bool {
	if i, ok := fsm.findServer(d.Endpoint); ok {
		fsm.setServer(i, d)
		return true
	}
	return false
}

func (fsm *clusterMonitorFSM) setServer(i int, d *desc.Server) {
	fsm.Servers[i] = d
}

func (fsm *clusterMonitorFSM) setType(clusterType desc.ClusterType) {
	fsm.ClusterType = clusterType
}
