// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"sync"
	"sync/atomic"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// generationStats represents the version of a pool. It tracks the generation number as well as the number of
// connections that have been created in the generation.
type generationStats struct {
	generation uint64
	numConns   uint64
}

// poolGenerationMap tracks the version for each server ID present in a pool. For deployments that are not behind a load
// balancer, there is only one server ID: primitive.NilObjectID. For load-balanced deployments, each server behind the
// load balancer will have a unique server ID.
type poolGenerationMap struct {
	// state must be accessed using the atomic package.
	state         int32
	generationMap map[primitive.ObjectID]*generationStats

	sync.Mutex
}

func newPoolGenerationMap() *poolGenerationMap {
	pgm := &poolGenerationMap{
		generationMap: make(map[primitive.ObjectID]*generationStats),
	}
	pgm.generationMap[primitive.NilObjectID] = &generationStats{}
	return pgm
}

func (p *poolGenerationMap) connect() {
	atomic.StoreInt32(&p.state, connected)
}

func (p *poolGenerationMap) disconnect() {
	atomic.StoreInt32(&p.state, disconnected)
}

// addConnection increments the connection count for the generation associated with the given server ID and returns the
// generation number for the connection.
func (p *poolGenerationMap) addConnection(serverIDPtr *primitive.ObjectID) uint64 {
	serverID := getServerID(serverIDPtr)
	p.Lock()
	defer p.Unlock()

	stats, ok := p.generationMap[serverID]
	if ok {
		// If the serverID is already being tracked, we only need to increment the connection count.
		stats.numConns++
		return stats.generation
	}

	// If the serverID is untracked, create a new entry with a starting generation number of 0.
	stats = &generationStats{
		numConns: 1,
	}
	p.generationMap[serverID] = stats
	return 0
}

func (p *poolGenerationMap) removeConnection(serverIDPtr *primitive.ObjectID) {
	serverID := getServerID(serverIDPtr)
	p.Lock()
	defer p.Unlock()

	stats, ok := p.generationMap[serverID]
	if !ok {
		return
	}

	// If the serverID is being tracked, decrement the connection count and delete this serverID to prevent the map
	// from growing unboundedly. This case would happen if a server behind a load-balancer was permanently removed
	// and its connections were pruned after a network error or idle timeout.
	stats.numConns--
	if stats.numConns == 0 {
		delete(p.generationMap, serverID)
	}
}

func (p *poolGenerationMap) clear(serverIDPtr *primitive.ObjectID) {
	serverID := getServerID(serverIDPtr)
	p.Lock()
	defer p.Unlock()

	if stats, ok := p.generationMap[serverID]; ok {
		stats.generation++
	}
}

func (p *poolGenerationMap) stale(serverIDPtr *primitive.ObjectID, knownGeneration uint64) bool {
	// If the map has been disconnected, all connections should be considered stale to ensure that they're closed.
	if atomic.LoadInt32(&p.state) == disconnected {
		return true
	}

	serverID := getServerID(serverIDPtr)
	p.Lock()
	defer p.Unlock()

	if stats, ok := p.generationMap[serverID]; ok {
		return knownGeneration < stats.generation
	}
	return false
}

func (p *poolGenerationMap) getGeneration(serverIDPtr *primitive.ObjectID) uint64 {
	serverID := getServerID(serverIDPtr)
	p.Lock()
	defer p.Unlock()

	if stats, ok := p.generationMap[serverID]; ok {
		return stats.generation
	}
	return 0
}

func getServerID(oid *primitive.ObjectID) primitive.ObjectID {
	if oid == nil {
		return primitive.NilObjectID
	}
	return *oid
}
