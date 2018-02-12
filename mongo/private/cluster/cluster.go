// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package cluster

import (
	"context"
	"errors"
	"math/rand"
	"sync"

	"github.com/mongodb/mongo-go-driver/mongo/internal"
	"github.com/mongodb/mongo-go-driver/mongo/model"
	"github.com/mongodb/mongo-go-driver/mongo/private/ops"
	"github.com/mongodb/mongo-go-driver/mongo/private/server"
	"github.com/mongodb/mongo-go-driver/mongo/readpref"
)

// ErrClusterClosed occurs on an attempt to use a closed
// cluster.
var ErrClusterClosed = errors.New("cluster is closed")

// New creates a new cluster. Internally, it
// creates a new Monitor with which to monitor the
// state of the cluster. When the Cluster is closed,
// the monitor will be stopped.
func New(opts ...Option) (*Cluster, error) {
	monitor, err := StartMonitor(opts...)
	if err != nil {
		return nil, err
	}

	cluster, err := NewWithMonitor(monitor, opts...)
	if err != nil {
		return nil, err
	}
	cluster.ownsMonitor = true
	return cluster, nil
}

// NewWithMonitor creates a new Cluster from
// an existing monitor. When the cluster is closed,
// the monitor will not be stopped. Any unspecified
// options will have their default value pulled from the monitor.
// Any monitor specific options will be ignored.
func NewWithMonitor(monitor *Monitor, opts ...Option) (*Cluster, error) {
	cfg, err := monitor.cfg.reconfig(opts...)
	if err != nil {
		return nil, err
	}
	cluster := &Cluster{
		cfg:          cfg,
		stateModel:   &model.Cluster{},
		stateServers: make(map[model.Addr]*server.Server),
		monitor:      monitor,
	}

	updates, _, _ := monitor.Subscribe()
	go func() {
		for desc := range updates {
			cluster.applyUpdate(desc)
		}
	}()

	return cluster, nil
}

// ServerSelector is a function that selects a server.
type ServerSelector func(*model.Cluster, []*model.Server) ([]*model.Server, error)

// Cluster represents a logical connection to a cluster.
type Cluster struct {
	cfg *config

	monitor      *Monitor
	ownsMonitor  bool
	stateModel   *model.Cluster
	stateLock    sync.Mutex
	stateServers map[model.Addr]*server.Server
}

// Close closes the cluster.
func (c *Cluster) Close() error {
	if c.ownsMonitor {
		c.monitor.Stop()
	}

	c.stateLock.Lock()
	defer c.stateLock.Unlock()

	if c.stateServers == nil {
		return nil
	}

	var err error
	for _, server := range c.stateServers {
		err = server.Close()
	}
	c.stateServers = nil
	c.stateModel = &model.Cluster{}

	return err
}

// Model gets a description of the cluster.
func (c *Cluster) Model() *model.Cluster {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()
	m := c.stateModel
	return m
}

// SelectServer selects a server given a selector.
// SelectServer complies with the server selection spec, and will time
// out after serverSelectionTimeout or when the parent context is done.
func (c *Cluster) SelectServer(ctx context.Context, selector ServerSelector,
	readPreference *readpref.ReadPref) (*ops.SelectedServer, error) {

	for {
		suitable, err := SelectServers(ctx, c.monitor, selector)
		if err != nil {
			return nil, err
		}

		selected := suitable[rand.Intn(len(suitable))]

		c.stateLock.Lock()
		if c.stateServers == nil {
			c.stateLock.Unlock()
			return nil, ErrClusterClosed
		}
		if server, ok := c.stateServers[selected.Addr]; ok {
			c.stateLock.Unlock()

			selectedServer := ops.SelectedServer{
				Server:   server,
				ReadPref: readPreference,
			}

			return &selectedServer, nil
		}
		c.stateLock.Unlock()

		// this is unfortunate. We have ended up here because we successfully
		// found a server that has since been removed. We need to start this process
		// over.
		continue
	}
}

// SelectServers returns a list of server descriptions matching
// a given selector. SelectServers will only time out when its
// parent context is done.
func SelectServers(ctx context.Context, m *Monitor, selector ServerSelector) ([]*model.Server, error) {
	return selectServers(ctx, m, selector)
}

func selectServers(ctx context.Context, m monitor, selector ServerSelector) ([]*model.Server, error) {
	updates, unsubscribe, _ := m.Subscribe()
	defer unsubscribe()

	var current *model.Cluster
	for {
		select {
		case <-ctx.Done():
			return nil, internal.WrapError(ctx.Err(), "server selection failed")
		case current = <-updates:
			// topology has changed
		}

		var allowedServers []*model.Server
		for _, s := range current.Servers {
			if s.Kind != model.Unknown {
				allowedServers = append(allowedServers, s)
			}
		}

		suitable, err := selector(current, allowedServers)
		if err != nil {
			return nil, err
		}

		if len(suitable) > 0 {
			return suitable, nil
		}

		m.RequestImmediateCheck()
	}
}

// applyUpdate handles updating the current description as well
// as ensure that the servers are still accurate.
func (c *Cluster) applyUpdate(cm *model.Cluster) {
	c.stateLock.Lock()
	defer c.stateLock.Unlock()

	if c.stateServers == nil {
		return
	}

	diff := model.DiffCluster(c.stateModel, cm)
	c.stateModel = cm

	for _, added := range diff.AddedServers {
		if mon, ok := c.monitor.ServerMonitor(added.Addr); ok {
			m, err := server.NewWithMonitor(mon, c.cfg.serverOpts...)
			if err != nil {
				// TODO: We need to log this... notify someone of something
				// here. The server couldn't be added because of some
				// reason.
				continue
			}
			c.stateServers[added.Addr] = m
		}
	}

	for _, removed := range diff.RemovedServers {
		if server, ok := c.stateServers[removed.Addr]; ok {
			// Ignore any error that occurs since this only gets called in a different goroutine.
			_ = server.Close()
		}

		delete(c.stateServers, removed.Addr)
	}
}
