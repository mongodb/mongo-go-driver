// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package model

import (
	"bytes"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

var supportedWireVersions = NewRange(2, 6)
var minSupportedMongoDBVersion = "2.6"

// NewFSM creates a new FSM.
func NewFSM() *FSM {
	return &FSM{}
}

// FSM is a finite state machine for transitioning cluster states.
type FSM struct {
	Cluster
	SetName string

	maxElectionID objectid.ObjectID
	maxSetVersion uint32
}

// Apply uses the server model to transition states.
func (fsm *FSM) Apply(s *Server) error {

	newServers := make([]*Server, len(fsm.Servers))
	copy(newServers, fsm.Servers)

	fsm.Cluster = Cluster{
		Kind:    fsm.Kind,
		Servers: newServers,
	}

	if _, ok := fsm.findServer(s.Addr); !ok {
		return nil
	}

	if s.WireVersion != nil {
		if s.WireVersion.Max < supportedWireVersions.Min {
			return fmt.Errorf(
				"server at %s reports wire version %d, but this version of the Go driver requires "+
					"at least %d (MongoDB %s)",
				s.Addr.String(),
				s.WireVersion.Max,
				supportedWireVersions.Min,
				minSupportedMongoDBVersion,
			)
		}

		if s.WireVersion.Min > supportedWireVersions.Max {
			return fmt.Errorf(
				"server at %s requires wire version %d, but this version of the Go driver only "+
					"supports up to %d",
				s.Addr.String(),
				s.WireVersion.Min,
				supportedWireVersions.Max,
			)
		}
	}

	switch fsm.Kind {
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

	return nil
}

func (fsm *FSM) applyToReplicaSetNoPrimary(s *Server) {
	switch s.Kind {
	case Standalone, Mongos:
		fsm.removeServerByAddr(s.Addr)
	case RSPrimary:
		fsm.updateRSFromPrimary(s)
	case RSSecondary, RSArbiter, RSMember:
		fsm.updateRSWithoutPrimary(s)
	case Unknown, RSGhost:
		fsm.replaceServer(s)
	}
}

func (fsm *FSM) applyToReplicaSetWithPrimary(s *Server) {
	switch s.Kind {
	case Standalone, Mongos:
		fsm.removeServerByAddr(s.Addr)
		fsm.checkIfHasPrimary()
	case RSPrimary:
		fsm.updateRSFromPrimary(s)
	case RSSecondary, RSArbiter, RSMember:
		fsm.updateRSWithPrimaryFromMember(s)
	case Unknown, RSGhost:
		fsm.replaceServer(s)
		fsm.checkIfHasPrimary()
	}
}

func (fsm *FSM) applyToSharded(s *Server) {
	switch s.Kind {
	case Mongos, Unknown:
		fsm.replaceServer(s)
	case Standalone, RSPrimary, RSSecondary, RSArbiter, RSMember, RSGhost:
		fsm.removeServerByAddr(s.Addr)
	}
}

func (fsm *FSM) applyToSingle(s *Server) {
	switch s.Kind {
	case Unknown:
		fsm.replaceServer(s)
	case Standalone, Mongos:
		if fsm.SetName != "" {
			fsm.removeServerByAddr(s.Addr)
			return
		}

		fsm.replaceServer(s)
	case RSPrimary, RSSecondary, RSArbiter, RSMember, RSGhost:
		if fsm.SetName != "" && fsm.SetName != s.SetName {
			fsm.removeServerByAddr(s.Addr)
			return
		}

		fsm.replaceServer(s)
	}
}

func (fsm *FSM) applyToUnknown(s *Server) {
	switch s.Kind {
	case Mongos:
		fsm.setKind(Sharded)
		fsm.replaceServer(s)
	case RSPrimary:
		fsm.updateRSFromPrimary(s)
	case RSSecondary, RSArbiter, RSMember:
		fsm.setKind(ReplicaSetNoPrimary)
		fsm.updateRSWithoutPrimary(s)
	case Standalone:
		fsm.updateUnknownWithStandalone(s)
	case Unknown, RSGhost:
		fsm.replaceServer(s)
	}
}

func (fsm *FSM) checkIfHasPrimary() {
	if _, ok := fsm.findPrimary(); ok {
		fsm.setKind(ReplicaSetWithPrimary)
	} else {
		fsm.setKind(ReplicaSetNoPrimary)
	}
}

func (fsm *FSM) updateRSFromPrimary(s *Server) {
	if fsm.SetName == "" {
		fsm.SetName = s.SetName
	} else if fsm.SetName != s.SetName {
		fsm.removeServerByAddr(s.Addr)
		fsm.checkIfHasPrimary()
		return
	}

	if s.SetVersion != 0 && !bytes.Equal(s.ElectionID[:], objectid.NilObjectID[:]) {
		if fsm.maxSetVersion > s.SetVersion || bytes.Compare(fsm.maxElectionID[:], s.ElectionID[:]) == 1 {
			fsm.replaceServer(&Server{
				Addr:      s.Addr,
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
		fsm.setServer(j, &Server{
			Addr:      fsm.Servers[j].Addr,
			LastError: fmt.Errorf("was a primary, but a new primary was discovered"),
		})
	}

	fsm.replaceServer(s)

	for j := len(fsm.Servers) - 1; j >= 0; j-- {
		found := false
		for _, member := range s.Members {
			if member == fsm.Servers[j].Addr {
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

func (fsm *FSM) updateRSWithPrimaryFromMember(s *Server) {
	if fsm.SetName != s.SetName {
		fsm.removeServerByAddr(s.Addr)
		fsm.checkIfHasPrimary()
		return
	}

	if s.Addr != s.CanonicalAddr {
		fsm.removeServerByAddr(s.Addr)
		fsm.checkIfHasPrimary()
		return
	}

	fsm.replaceServer(s)

	if _, ok := fsm.findPrimary(); !ok {
		fsm.setKind(ReplicaSetNoPrimary)
	}
}

func (fsm *FSM) updateRSWithoutPrimary(s *Server) {
	if fsm.SetName == "" {
		fsm.SetName = s.SetName
	} else if fsm.SetName != s.SetName {
		fsm.removeServerByAddr(s.Addr)
		return
	}

	for _, member := range s.Members {
		if _, ok := fsm.findServer(member); !ok {
			fsm.addServer(member)
		}
	}

	if s.Addr != s.CanonicalAddr {
		fsm.removeServerByAddr(s.Addr)
		return
	}

	fsm.replaceServer(s)
}

func (fsm *FSM) updateUnknownWithStandalone(s *Server) {
	if len(fsm.Servers) > 1 {
		fsm.removeServerByAddr(s.Addr)
		return
	}

	fsm.setKind(Single)
	fsm.replaceServer(s)
}

func (fsm *FSM) addServer(addr Addr) {
	fsm.Servers = append(fsm.Servers, &Server{
		Addr: addr.Canonicalize(),
	})
}

func (fsm *FSM) findPrimary() (int, bool) {
	for i, s := range fsm.Servers {
		if s.Kind == RSPrimary {
			return i, true
		}
	}

	return 0, false
}

func (fsm *FSM) findServer(addr Addr) (int, bool) {
	canon := addr.Canonicalize()
	for i, s := range fsm.Servers {
		if canon == s.Addr {
			return i, true
		}
	}

	return 0, false
}

func (fsm *FSM) removeServer(i int) {
	fsm.Servers = append(fsm.Servers[:i], fsm.Servers[i+1:]...)
}

func (fsm *FSM) removeServerByAddr(addr Addr) {
	if i, ok := fsm.findServer(addr); ok {
		fsm.removeServer(i)
	}
}

func (fsm *FSM) replaceServer(s *Server) bool {
	if i, ok := fsm.findServer(s.Addr); ok {
		fsm.setServer(i, s)
		return true
	}
	return false
}

func (fsm *FSM) setServer(i int, s *Server) {
	fsm.Servers[i] = s
}

func (fsm *FSM) setKind(k ClusterKind) {
	fsm.Kind = k
}
