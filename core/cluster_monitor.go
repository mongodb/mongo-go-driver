package core

//go:generate go run spec_cluster_monitor_internal_test_generator.go

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"

	"gopkg.in/mgo.v2/bson"
)

// StartClusterMonitor begins monitoring a cluster.
func StartClusterMonitor(opts ClusterOptions) (*ClusterMonitor, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	opts.fillDefaults()

	m := &ClusterMonitor{
		subscribers:          make(map[int]chan *ClusterDesc),
		changes:              make(chan *ServerDesc),
		desc:                 &ClusterDesc{},
		fsm:                  &clusterMonitorFSM{},
		servers:              make(map[Endpoint]*ServerMonitor),
		serverOptionsFactory: opts.ServerOptionsFactory,
	}

	if opts.ReplicaSetName != "" {
		m.fsm.setName = opts.ReplicaSetName
		m.fsm.clusterType = ReplicaSetNoPrimary
	}
	if opts.ConnectionMode == SingleMode {
		m.fsm.clusterType = Single
	}

	for _, ep := range opts.Servers {
		canonicalized := ep.Canonicalize()
		m.fsm.addServer(canonicalized)
		m.startMonitoringEndpoint(canonicalized)
	}

	go func() {
		for change := range m.changes {
			// apply the change
			desc := m.apply(change)
			m.descLock.Lock()
			m.desc = desc
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
				ch <- desc
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
	desc     *ClusterDesc

	changes chan *ServerDesc
	fsm     *clusterMonitorFSM

	subscribers         map[int]chan *ClusterDesc
	subscriptionsClosed bool
	subscriberLock      sync.Mutex

	serversLock          sync.Mutex
	serversClosed        bool
	servers              map[Endpoint]*ServerMonitor
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
func (m *ClusterMonitor) Subscribe() (<-chan *ClusterDesc, func(), error) {
	// create channel and populate with current state
	ch := make(chan *ClusterDesc, 1)
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

func (m *ClusterMonitor) startMonitoringEndpoint(endpoint Endpoint) {
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

func (m *ClusterMonitor) stopMonitoringEndpoint(endpoint Endpoint, server *ServerMonitor) {
	server.Stop()
	delete(m.servers, endpoint)
}

func (m *ClusterMonitor) apply(desc *ServerDesc) *ClusterDesc {
	old := m.fsm.ClusterDesc
	m.fsm.apply(desc)
	new := m.fsm.ClusterDesc

	diff := diffClusterDesc(&old, &new)
	m.serversLock.Lock()
	if m.serversClosed {
		// maybe return an empty desc?
		return nil
	}
	for _, oldServer := range diff.RemovedServers {
		if sm, ok := m.servers[oldServer.endpoint]; ok {
			m.stopMonitoringEndpoint(oldServer.endpoint, sm)
		}
	}
	for _, newServer := range diff.AddedServers {
		if _, ok := m.servers[newServer.endpoint]; !ok {
			m.startMonitoringEndpoint(newServer.endpoint)
		}
	}
	m.serversLock.Unlock()
	return &new
}

type clusterMonitorFSM struct {
	ClusterDesc

	maxElectionID bson.ObjectId
	maxSetVersion uint32
	setName       string
}

func (fsm *clusterMonitorFSM) apply(desc *ServerDesc) {

	newServers := make([]*ServerDesc, len(fsm.servers))
	copy(newServers, fsm.servers)

	fsm.ClusterDesc = ClusterDesc{
		clusterType: fsm.clusterType,
		servers:     newServers,
	}

	if _, ok := fsm.findServer(desc.endpoint); !ok {
		return
	}

	switch fsm.clusterType {
	case UnknownClusterType:
		fsm.applyToUnknownClusterType(desc)
	case Sharded:
		fsm.applyToShardedClusterType(desc)
	case ReplicaSetNoPrimary:
		fsm.applyToReplicaSetNoPrimary(desc)
	case ReplicaSetWithPrimary:
		fsm.applyToReplicaSetWithPrimary(desc)
	case Single:
		fsm.applyToSingle(desc)
	}
}

func (fsm *clusterMonitorFSM) applyToReplicaSetNoPrimary(desc *ServerDesc) {
	switch desc.serverType {
	case Standalone, Mongos:
		fsm.removeServerByEndpoint(desc.endpoint)
	case RSPrimary:
		fsm.updateRSFromPrimary(desc)
	case RSSecondary, RSArbiter, RSMember:
		fsm.updateRSWithoutPrimary(desc)
	case UnknownServerType, RSGhost:
		fsm.replaceServer(desc)
	}
}

func (fsm *clusterMonitorFSM) applyToReplicaSetWithPrimary(desc *ServerDesc) {
	switch desc.serverType {
	case Standalone, Mongos:
		fsm.removeServerByEndpoint(desc.endpoint)
		fsm.checkIfHasPrimary()
	case RSPrimary:
		fsm.updateRSFromPrimary(desc)
	case RSSecondary, RSArbiter, RSMember:
		fsm.updateRSWithPrimaryFromMember(desc)
	case UnknownServerType, RSGhost:
		fsm.replaceServer(desc)
		fsm.checkIfHasPrimary()
	}
}

func (fsm *clusterMonitorFSM) applyToShardedClusterType(desc *ServerDesc) {
	switch desc.serverType {
	case Mongos, UnknownServerType:
		fsm.replaceServer(desc)
	case Standalone, RSPrimary, RSSecondary, RSArbiter, RSMember, RSGhost:
		fsm.removeServerByEndpoint(desc.endpoint)
	}
}

func (fsm *clusterMonitorFSM) applyToSingle(desc *ServerDesc) {
	switch desc.serverType {
	case UnknownServerType:
		fsm.replaceServer(desc)
	case Standalone, Mongos:
		if fsm.setName != "" {
			fsm.removeServerByEndpoint(desc.endpoint)
			return
		}

		fsm.replaceServer(desc)
	case RSPrimary, RSSecondary, RSArbiter, RSMember, RSGhost:
		if fsm.setName != "" && fsm.setName != desc.setName {
			fsm.removeServerByEndpoint(desc.endpoint)
			return
		}

		fsm.replaceServer(desc)
	}
}

func (fsm *clusterMonitorFSM) applyToUnknownClusterType(desc *ServerDesc) {
	switch desc.serverType {
	case Mongos:
		fsm.setType(Sharded)
		fsm.replaceServer(desc)
	case RSPrimary:
		fsm.updateRSFromPrimary(desc)
	case RSSecondary, RSArbiter, RSMember:
		fsm.setType(ReplicaSetNoPrimary)
		fsm.updateRSWithoutPrimary(desc)
	case Standalone:
		fsm.updateUnknownWithStandalone(desc)
	case UnknownServerType, RSGhost:
		fsm.replaceServer(desc)
	}
}

func (fsm *clusterMonitorFSM) checkIfHasPrimary() {
	if _, ok := fsm.findPrimary(); ok {
		fsm.setType(ReplicaSetWithPrimary)
	} else {
		fsm.setType(ReplicaSetNoPrimary)
	}
}

func (fsm *clusterMonitorFSM) updateRSFromPrimary(desc *ServerDesc) {
	if fsm.setName == "" {
		fsm.setName = desc.setName
	} else if fsm.setName != desc.setName {
		fsm.removeServerByEndpoint(desc.endpoint)
		fsm.checkIfHasPrimary()
		return
	}

	if desc.setVersion != 0 && desc.electionID != "" {
		if fsm.maxSetVersion > desc.setVersion || fsm.maxElectionID > desc.electionID {
			fsm.replaceServer(&ServerDesc{
				endpoint:  desc.endpoint,
				lastError: fmt.Errorf("was a primary, but its set version or election id is stale"),
			})
			fsm.checkIfHasPrimary()
			return
		}

		fsm.maxElectionID = desc.electionID
	}

	if desc.setVersion > fsm.maxSetVersion {
		fsm.maxSetVersion = desc.setVersion
	}

	if j, ok := fsm.findPrimary(); ok {
		fsm.setServer(j, &ServerDesc{
			endpoint:  fsm.servers[j].endpoint,
			lastError: fmt.Errorf("was a primary, but a new primary was discovered"),
		})
	}

	fsm.replaceServer(desc)

	members := endpoints(desc.members)
	for j := len(fsm.servers) - 1; j >= 0; j-- {
		server := fsm.servers[j]
		if !members.contains(server.endpoint) {
			fsm.removeServer(j)
		}
	}

	for _, member := range members {
		if _, ok := fsm.findServer(member); !ok {
			fsm.addServer(member)
		}
	}

	fsm.checkIfHasPrimary()
}

func (fsm *clusterMonitorFSM) updateRSWithPrimaryFromMember(desc *ServerDesc) {
	if fsm.setName != desc.setName {
		fsm.removeServerByEndpoint(desc.endpoint)
		fsm.checkIfHasPrimary()
		return
	}

	if desc.endpoint != desc.canonicalEndpoint {
		fsm.removeServerByEndpoint(desc.endpoint)
		fsm.checkIfHasPrimary()
		return
	}

	fsm.replaceServer(desc)

	if _, ok := fsm.findPrimary(); !ok {
		fsm.setType(ReplicaSetNoPrimary)
	}
}

func (fsm *clusterMonitorFSM) updateRSWithoutPrimary(desc *ServerDesc) {
	if fsm.setName == "" {
		fsm.setName = desc.setName
	} else if fsm.setName != desc.setName {
		fsm.removeServerByEndpoint(desc.endpoint)
		return
	}

	for _, member := range desc.members {
		if _, ok := fsm.findServer(member); !ok {
			fsm.addServer(member)
		}
	}

	if desc.endpoint != desc.canonicalEndpoint {
		fsm.removeServerByEndpoint(desc.endpoint)
		return
	}

	fsm.replaceServer(desc)
}

func (fsm *clusterMonitorFSM) updateUnknownWithStandalone(desc *ServerDesc) {
	if len(fsm.servers) > 1 {
		fsm.removeServerByEndpoint(desc.endpoint)
		return
	}

	fsm.setType(Single)
	fsm.replaceServer(desc)
}

func (fsm *clusterMonitorFSM) addServer(endpoint Endpoint) {
	fsm.servers = append(fsm.servers, &ServerDesc{
		endpoint: endpoint,
	})
}

func (fsm *clusterMonitorFSM) findPrimary() (int, bool) {
	for i, s := range fsm.servers {
		if s.serverType == RSPrimary {
			return i, true
		}
	}

	return 0, false
}

func (fsm *clusterMonitorFSM) findServer(endpoint Endpoint) (int, bool) {
	for i, s := range fsm.servers {
		if endpoint == s.endpoint {
			return i, true
		}
	}

	return 0, false
}

func (fsm *clusterMonitorFSM) removeServer(i int) {
	fsm.servers = append(fsm.servers[:i], fsm.servers[i+1:]...)
}

func (fsm *clusterMonitorFSM) removeServerByEndpoint(endpoint Endpoint) {
	if i, ok := fsm.findServer(endpoint); ok {
		fsm.removeServer(i)
	}
}

func (fsm *clusterMonitorFSM) replaceServer(desc *ServerDesc) bool {
	if i, ok := fsm.findServer(desc.endpoint); ok {
		fsm.setServer(i, desc)
		return true
	}
	return false
}

func (fsm *clusterMonitorFSM) setServer(i int, desc *ServerDesc) {
	fsm.servers[i] = desc
}

func (fsm *clusterMonitorFSM) setType(clusterType ClusterType) {
	fsm.clusterType = clusterType
}
