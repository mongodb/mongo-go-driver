// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package description

import (
	"fmt"
	"time"

	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/mongodb/mongo-go-driver/core/addr"
	"github.com/mongodb/mongo-go-driver/core/result"
	"github.com/mongodb/mongo-go-driver/core/tag"
)

// UnsetRTT is the unset value for a round trip time.
const UnsetRTT = -1 * time.Millisecond

// SelectedServer represents a selected server that is a member of a topology.
type SelectedServer struct {
	Server
	Kind TopologyKind
}

// Server represents a description of a server. This is created from an isMaster
// command.
type Server struct {
	Addr addr.Addr

	AverageRTT        time.Duration
	AverageRTTSet     bool
	CanonicalAddr     addr.Addr
	ElectionID        objectid.ObjectID
	GitVersion        string
	HeartbeatInterval time.Duration
	LastError         error
	LastUpdateTime    time.Time
	LastWriteTime     time.Time
	MaxBatchCount     uint16
	MaxDocumentSize   uint32
	MaxMessageSize    uint32
	Members           []addr.Addr
	ReadOnly          bool
	SetName           string
	SetVersion        uint32
	Tags              tag.Set
	Kind              ServerKind
	WireVersion       *VersionRange
	Version           Version
}

// NewServer creates a new server description from the given parameters.
func NewServer(address addr.Addr, isMaster result.IsMaster, buildInfo result.BuildInfo) Server {
	i := Server{
		Addr: address,

		CanonicalAddr:   addr.Addr(isMaster.Me).Canonicalize(),
		ElectionID:      isMaster.ElectionID,
		LastUpdateTime:  time.Now().UTC(),
		LastWriteTime:   isMaster.LastWriteTimestamp,
		MaxBatchCount:   isMaster.MaxWriteBatchSize,
		MaxDocumentSize: isMaster.MaxBSONObjectSize,
		MaxMessageSize:  isMaster.MaxMessageSizeBytes,
		SetName:         isMaster.SetName,
		SetVersion:      isMaster.SetVersion,
		Tags:            tag.NewTagSetFromMap(isMaster.Tags),
	}

	if !buildInfo.IsZero() {
		i.GitVersion = buildInfo.GitVersion
		i.Version.Desc = buildInfo.Version
		i.Version.Parts = buildInfo.VersionArray
	}

	if i.CanonicalAddr == "" {
		i.CanonicalAddr = address
	}

	if isMaster.OK != 1 {
		i.LastError = fmt.Errorf("not ok")
		return i
	}

	for _, host := range isMaster.Hosts {
		i.Members = append(i.Members, addr.Addr(host).Canonicalize())
	}

	for _, passive := range isMaster.Passives {
		i.Members = append(i.Members, addr.Addr(passive).Canonicalize())
	}

	for _, arbiter := range isMaster.Arbiters {
		i.Members = append(i.Members, addr.Addr(arbiter).Canonicalize())
	}

	i.Kind = Standalone

	if isMaster.IsReplicaSet {
		i.Kind = RSGhost
	} else if isMaster.SetName != "" {
		if isMaster.IsMaster {
			i.Kind = RSPrimary
		} else if isMaster.Hidden {
			i.Kind = RSMember
		} else if isMaster.Secondary {
			i.Kind = RSSecondary
		} else if isMaster.ArbiterOnly {
			i.Kind = RSArbiter
		} else {
			i.Kind = RSMember
		}
	} else if isMaster.Msg == "isdbgrid" {
		i.Kind = Mongos
	}

	i.WireVersion = &VersionRange{
		Min: isMaster.MinWireVersion,
		Max: isMaster.MaxWireVersion,
	}

	return i
}

// SetAverageRTT sets the average round trip time for this server description.
func (s Server) SetAverageRTT(rtt time.Duration) Server {
	s.AverageRTT = rtt
	if rtt == UnsetRTT {
		s.AverageRTTSet = false
	} else {
		s.AverageRTTSet = true
	}

	return s
}
