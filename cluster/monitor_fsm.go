package cluster

//go:generate go run monitor_fsm_spec_internal_test_generator.go

import (
	"fmt"

	"github.com/10gen/mongo-go-driver/desc"
	"gopkg.in/mgo.v2/bson"
)

type monitorFSM struct {
	desc.Cluster

	maxElectionID bson.ObjectId
	maxSetVersion uint32
	setName       string
}

func (fsm *monitorFSM) apply(d *desc.Server) {

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

func (fsm *monitorFSM) applyToReplicaSetNoPrimary(d *desc.Server) {
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

func (fsm *monitorFSM) applyToReplicaSetWithPrimary(d *desc.Server) {
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

func (fsm *monitorFSM) applyToShardedClusterType(d *desc.Server) {
	switch d.ServerType {
	case desc.Mongos, desc.UnknownServerType:
		fsm.replaceServer(d)
	case desc.Standalone, desc.RSPrimary, desc.RSSecondary, desc.RSArbiter, desc.RSMember, desc.RSGhost:
		fsm.removeServerByEndpoint(d.Endpoint)
	}
}

func (fsm *monitorFSM) applyToSingle(d *desc.Server) {
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

func (fsm *monitorFSM) applyToUnknownClusterType(d *desc.Server) {
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

func (fsm *monitorFSM) checkIfHasPrimary() {
	if _, ok := fsm.findPrimary(); ok {
		fsm.setType(desc.ReplicaSetWithPrimary)
	} else {
		fsm.setType(desc.ReplicaSetNoPrimary)
	}
}

func (fsm *monitorFSM) updateRSFromPrimary(d *desc.Server) {
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

func (fsm *monitorFSM) updateRSWithPrimaryFromMember(d *desc.Server) {
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

func (fsm *monitorFSM) updateRSWithoutPrimary(d *desc.Server) {
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

func (fsm *monitorFSM) updateUnknownWithStandalone(d *desc.Server) {
	if len(fsm.Servers) > 1 {
		fsm.removeServerByEndpoint(d.Endpoint)
		return
	}

	fsm.setType(desc.Single)
	fsm.replaceServer(d)
}

func (fsm *monitorFSM) addServer(endpoint desc.Endpoint) {
	fsm.Servers = append(fsm.Servers, &desc.Server{
		Endpoint: endpoint,
	})
}

func (fsm *monitorFSM) findPrimary() (int, bool) {
	for i, s := range fsm.Servers {
		if s.ServerType == desc.RSPrimary {
			return i, true
		}
	}

	return 0, false
}

func (fsm *monitorFSM) findServer(endpoint desc.Endpoint) (int, bool) {
	for i, s := range fsm.Servers {
		if endpoint == s.Endpoint {
			return i, true
		}
	}

	return 0, false
}

func (fsm *monitorFSM) removeServer(i int) {
	fsm.Servers = append(fsm.Servers[:i], fsm.Servers[i+1:]...)
}

func (fsm *monitorFSM) removeServerByEndpoint(endpoint desc.Endpoint) {
	if i, ok := fsm.findServer(endpoint); ok {
		fsm.removeServer(i)
	}
}

func (fsm *monitorFSM) replaceServer(d *desc.Server) bool {
	if i, ok := fsm.findServer(d.Endpoint); ok {
		fsm.setServer(i, d)
		return true
	}
	return false
}

func (fsm *monitorFSM) setServer(i int, d *desc.Server) {
	fsm.Servers[i] = d
}

func (fsm *monitorFSM) setType(clusterType desc.ClusterType) {
	fsm.ClusterType = clusterType
}
