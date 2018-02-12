// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package model

import (
	"fmt"
	"time"

	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/mongodb/mongo-go-driver/mongo/internal"
)

// UnsetRTT is the unset value for a round trip time.
const UnsetRTT = -1 * time.Millisecond

// Server is a description of a server.
type Server struct {
	Addr Addr

	AverageRTT        time.Duration
	AverageRTTSet     bool
	CanonicalAddr     Addr
	ElectionID        objectid.ObjectID
	GitVersion        string
	HeartbeatInterval time.Duration
	LastError         error
	LastUpdateTime    time.Time
	LastWriteTime     time.Time
	MaxBatchCount     uint16
	MaxDocumentSize   uint32
	MaxMessageSize    uint32
	Members           []Addr
	ReadOnly          bool
	SetName           string
	SetVersion        uint32
	Tags              TagSet
	Kind              ServerKind
	WireVersion       *Range
	Version           Version
}

// SetAverageRTT sets the average round trip time.
func (i *Server) SetAverageRTT(rtt time.Duration) {
	i.AverageRTT = rtt
	if rtt == UnsetRTT {
		i.AverageRTTSet = false
	} else {
		i.AverageRTTSet = true
	}
}

// BuildServer builds a server.Server from an endpoint, IsMasterResult, and a BuildInfoResult.
func BuildServer(addr Addr, isMasterResult *internal.IsMasterResult, buildInfoResult *internal.BuildInfoResult) *Server {
	i := &Server{
		Addr: addr,

		CanonicalAddr:   Addr(isMasterResult.Me).Canonicalize(),
		ElectionID:      isMasterResult.ElectionID,
		LastUpdateTime:  time.Now().UTC(),
		LastWriteTime:   isMasterResult.LastWriteTimestamp,
		MaxBatchCount:   isMasterResult.MaxWriteBatchSize,
		MaxDocumentSize: isMasterResult.MaxBSONObjectSize,
		MaxMessageSize:  isMasterResult.MaxMessageSizeBytes,
		SetName:         isMasterResult.SetName,
		SetVersion:      isMasterResult.SetVersion,
		Tags:            NewTagSetFromMap(isMasterResult.Tags),
	}

	if buildInfoResult != nil {
		i.GitVersion = buildInfoResult.GitVersion
		i.Version.Desc = buildInfoResult.Version
		i.Version.Parts = buildInfoResult.VersionArray
	}

	if i.CanonicalAddr == "" {
		i.CanonicalAddr = addr
	}

	if isMasterResult.OK != 1 {
		i.LastError = fmt.Errorf("not ok")
		return i
	}

	for _, host := range isMasterResult.Hosts {
		i.Members = append(i.Members, Addr(host).Canonicalize())
	}

	for _, passive := range isMasterResult.Passives {
		i.Members = append(i.Members, Addr(passive).Canonicalize())
	}

	for _, arbiter := range isMasterResult.Arbiters {
		i.Members = append(i.Members, Addr(arbiter).Canonicalize())
	}

	i.Kind = Standalone

	if isMasterResult.IsReplicaSet {
		i.Kind = RSGhost
	} else if isMasterResult.SetName != "" {
		if isMasterResult.IsMaster {
			i.Kind = RSPrimary
		} else if isMasterResult.Hidden {
			i.Kind = RSMember
		} else if isMasterResult.Secondary {
			i.Kind = RSSecondary
		} else if isMasterResult.ArbiterOnly {
			i.Kind = RSArbiter
		} else {
			i.Kind = RSMember
		}
	} else if isMasterResult.Msg == "isdbgrid" {
		i.Kind = Mongos
	}

	i.WireVersion = &Range{
		Min: isMasterResult.MinWireVersion,
		Max: isMasterResult.MaxWireVersion,
	}

	return i
}
