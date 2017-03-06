package cluster

//go:generate go run monitor_fsm_spec_internal_test_generator.go

import (
	"fmt"

	"github.com/10gen/mongo-go-driver/bson"
	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/server"
)

type monitorFSM struct {
	Desc

	maxElectionID bson.ObjectId
	maxSetVersion uint32
	setName       string
}

func (fsm *monitorFSM) apply(s *server.Desc) {

	newServers := make([]*server.Desc, len(fsm.Servers))
	copy(newServers, fsm.Servers)

	fsm.Desc = Desc{
		Type:    fsm.Type,
		Servers: newServers,
	}

	if _, ok := fsm.findServer(s.Endpoint); !ok {
		return
	}

	switch fsm.Type {
	case Unknown:
		fsm.applyToUnknown(s)
	case Sharded:
		fsm.applyToSharded(s)
	case ReplicaSetNoPrimary:
		fsm.applyToReplicaSetNoPrimary(s)
	case ReplicaSetWithPrimary:
		fsm.applyToReplicaSetWithPrimary(s)
	case Single:
		fsm.applyToSingle(s)
	}
}

func (fsm *monitorFSM) applyToReplicaSetNoPrimary(s *server.Desc) {
	switch s.Type {
	case server.Standalone, server.Mongos:
		fsm.removeServerByEndpoint(s.Endpoint)
	case server.RSPrimary:
		fsm.updateRSFromPrimary(s)
	case server.RSSecondary, server.RSArbiter, server.RSMember:
		fsm.updateRSWithoutPrimary(s)
	case server.Unknown, server.RSGhost:
		fsm.replaceServer(s)
	}
}

func (fsm *monitorFSM) applyToReplicaSetWithPrimary(s *server.Desc) {
	switch s.Type {
	case server.Standalone, server.Mongos:
		fsm.removeServerByEndpoint(s.Endpoint)
		fsm.checkIfHasPrimary()
	case server.RSPrimary:
		fsm.updateRSFromPrimary(s)
	case server.RSSecondary, server.RSArbiter, server.RSMember:
		fsm.updateRSWithPrimaryFromMember(s)
	case server.Unknown, server.RSGhost:
		fsm.replaceServer(s)
		fsm.checkIfHasPrimary()
	}
}

func (fsm *monitorFSM) applyToSharded(s *server.Desc) {
	switch s.Type {
	case server.Mongos, server.Unknown:
		fsm.replaceServer(s)
	case server.Standalone, server.RSPrimary, server.RSSecondary, server.RSArbiter, server.RSMember, server.RSGhost:
		fsm.removeServerByEndpoint(s.Endpoint)
	}
}

func (fsm *monitorFSM) applyToSingle(s *server.Desc) {
	switch s.Type {
	case server.Unknown:
		fsm.replaceServer(s)
	case server.Standalone, server.Mongos:
		if fsm.setName != "" {
			fsm.removeServerByEndpoint(s.Endpoint)
			return
		}

		fsm.replaceServer(s)
	case server.RSPrimary, server.RSSecondary, server.RSArbiter, server.RSMember, server.RSGhost:
		if fsm.setName != "" && fsm.setName != s.SetName {
			fsm.removeServerByEndpoint(s.Endpoint)
			return
		}

		fsm.replaceServer(s)
	}
}

func (fsm *monitorFSM) applyToUnknown(s *server.Desc) {
	switch s.Type {
	case server.Mongos:
		fsm.setType(Sharded)
		fsm.replaceServer(s)
	case server.RSPrimary:
		fsm.updateRSFromPrimary(s)
	case server.RSSecondary, server.RSArbiter, server.RSMember:
		fsm.setType(ReplicaSetNoPrimary)
		fsm.updateRSWithoutPrimary(s)
	case server.Standalone:
		fsm.updateUnknownWithStandalone(s)
	case server.Unknown, server.RSGhost:
		fsm.replaceServer(s)
	}
}

func (fsm *monitorFSM) checkIfHasPrimary() {
	if _, ok := fsm.findPrimary(); ok {
		fsm.setType(ReplicaSetWithPrimary)
	} else {
		fsm.setType(ReplicaSetNoPrimary)
	}
}

func (fsm *monitorFSM) updateRSFromPrimary(s *server.Desc) {
	if fsm.setName == "" {
		fsm.setName = s.SetName
	} else if fsm.setName != s.SetName {
		fsm.removeServerByEndpoint(s.Endpoint)
		fsm.checkIfHasPrimary()
		return
	}

	if s.SetVersion != 0 && s.ElectionID != "" {
		if fsm.maxSetVersion > s.SetVersion || fsm.maxElectionID > s.ElectionID {
			fsm.replaceServer(&server.Desc{
				Endpoint:  s.Endpoint,
				LastError: fmt.Errorf("was a primary, but its set version or election id is stale"),
			})
			fsm.checkIfHasPrimary()
			return
		}

		fsm.maxElectionID = s.ElectionID
	}

	if s.SetVersion > fsm.maxSetVersion {
		fsm.maxSetVersion = s.SetVersion
	}

	if j, ok := fsm.findPrimary(); ok {
		fsm.setServer(j, &server.Desc{
			Endpoint:  fsm.Servers[j].Endpoint,
			LastError: fmt.Errorf("was a primary, but a new primary was discovered"),
		})
	}

	fsm.replaceServer(s)

	for j := len(fsm.Servers) - 1; j >= 0; j-- {
		server := fsm.Servers[j]
		found := false
		for _, member := range s.Members {
			if member == server.Endpoint {
				found = true
				break
			}
		}
		if !found {
			fsm.removeServer(j)
		}
	}

	for _, member := range s.Members {
		if _, ok := fsm.findServer(member); !ok {
			fsm.addServer(member)
		}
	}

	fsm.checkIfHasPrimary()
}

func (fsm *monitorFSM) updateRSWithPrimaryFromMember(s *server.Desc) {
	if fsm.setName != s.SetName {
		fsm.removeServerByEndpoint(s.Endpoint)
		fsm.checkIfHasPrimary()
		return
	}

	if s.Endpoint != s.CanonicalEndpoint {
		fsm.removeServerByEndpoint(s.Endpoint)
		fsm.checkIfHasPrimary()
		return
	}

	fsm.replaceServer(s)

	if _, ok := fsm.findPrimary(); !ok {
		fsm.setType(ReplicaSetNoPrimary)
	}
}

func (fsm *monitorFSM) updateRSWithoutPrimary(s *server.Desc) {
	if fsm.setName == "" {
		fsm.setName = s.SetName
	} else if fsm.setName != s.SetName {
		fsm.removeServerByEndpoint(s.Endpoint)
		return
	}

	for _, member := range s.Members {
		if _, ok := fsm.findServer(member); !ok {
			fsm.addServer(member)
		}
	}

	if s.Endpoint != s.CanonicalEndpoint {
		fsm.removeServerByEndpoint(s.Endpoint)
		return
	}

	fsm.replaceServer(s)
}

func (fsm *monitorFSM) updateUnknownWithStandalone(s *server.Desc) {
	if len(fsm.Servers) > 1 {
		fsm.removeServerByEndpoint(s.Endpoint)
		return
	}

	fsm.setType(Single)
	fsm.replaceServer(s)
}

func (fsm *monitorFSM) addServer(endpoint conn.Endpoint) {
	fsm.Servers = append(fsm.Servers, &server.Desc{
		Endpoint: endpoint,
	})
}

func (fsm *monitorFSM) findPrimary() (int, bool) {
	for i, s := range fsm.Servers {
		if s.Type == server.RSPrimary {
			return i, true
		}
	}

	return 0, false
}

func (fsm *monitorFSM) findServer(endpoint conn.Endpoint) (int, bool) {
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

func (fsm *monitorFSM) removeServerByEndpoint(endpoint conn.Endpoint) {
	if i, ok := fsm.findServer(endpoint); ok {
		fsm.removeServer(i)
	}
}

func (fsm *monitorFSM) replaceServer(s *server.Desc) bool {
	if i, ok := fsm.findServer(s.Endpoint); ok {
		fsm.setServer(i, s)
		return true
	}
	return false
}

func (fsm *monitorFSM) setServer(i int, s *server.Desc) {
	fsm.Servers[i] = s
}

func (fsm *monitorFSM) setType(clusterType Type) {
	fsm.Type = clusterType
}
