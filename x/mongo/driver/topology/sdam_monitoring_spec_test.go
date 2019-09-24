// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"path"
	"sync"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	testhelpers "go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver/address"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"go.mongodb.org/mongo-driver/x/mongo/driver/description"
	"go.mongodb.org/mongo-driver/x/mongo/driver/uuid"
)

var publishedEvents []interface{}
var lock sync.Mutex

type topologyDescription struct {
	TopologyType string              `json:"topologyType"`
	Servers      []serverDescription `json:"servers"`
	SetName      string              `json:"setName,omitempty"`
}

type serverDescription struct {
	Address  string   `json:"address"`
	Arbiters []string `json:"arbiters"`
	Hosts    []string `json:"hosts"`
	Passives []string `json:"passives"`
	Primary  string   `json:"primary,omitempty"`
	SetName  string   `json:"setName,omitempty"`
	Type     string   `json:"type"`
}

type topologyOpeningEvent struct {
	TopologyID string `json:"topologyId"`
}

type serverOpeningEvent struct {
	Address    string `json:"address"`
	TopologyID string `json:"topologyId"`
}

type topologyDescriptionChangedEvent struct {
	TopologyID          string              `json:"topologyId"`
	PreviousDescription topologyDescription `json:"previousDescription"`
	NewDescription      topologyDescription `json:"newDescription"`
}

type serverDescriptionChangedEvent struct {
	Address             string            `json:"address"`
	TopologyID          string            `json:"topologyId"`
	PreviousDescription serverDescription `json:"previousDescription"`
	NewDescription      serverDescription `json:"newDescription"`
}

type serverClosedEvent struct {
	Address    string `json:"address"`
	TopologyID string `json:"topologyId"`
}

type monitoringEvent struct {
	TopologyOpeningEvent            *topologyOpeningEvent            `json:"topology_opening_event,omitempty"`
	ServerOpeningEvent              *serverOpeningEvent              `json:"server_opening_event,omitempty"`
	TopologyDescriptionChangedEvent *topologyDescriptionChangedEvent `json:"topology_description_changed_event,omitempty"`
	ServerDescriptionChangedEvent   *serverDescriptionChangedEvent   `json:"server_description_changed_event,omitempty"`
	ServerClosedEvent               *serverClosedEvent               `json:"server_closed_event,omitempty"`
}

type monitoringOutcome struct {
	Events []monitoringEvent
}

type monitoringPhase struct {
	Responses []response
	Outcome   monitoringOutcome
}

type monitoringTestCase struct {
	Description string
	URI         string
	Phases      []monitoringPhase
}

func serverDescriptionChanged(ctx context.Context, e *event.ServerDescriptionChangedEvent) {
	lock.Lock()
	publishedEvents = append(publishedEvents, *e)
	lock.Unlock()
}

func serverOpening(ctx context.Context, e *event.ServerOpeningEvent) {
	lock.Lock()
	publishedEvents = append(publishedEvents, *e)
	lock.Unlock()
}

func topologyDescriptionChanged(ctx context.Context, e *event.TopologyDescriptionChangedEvent) {
	lock.Lock()
	publishedEvents = append(publishedEvents, *e)
	lock.Unlock()
}

func topologyOpening(ctx context.Context, e *event.TopologyOpeningEvent) {
	lock.Lock()
	publishedEvents = append(publishedEvents, *e)
	lock.Unlock()
}

func serverClosed(ctx context.Context, e *event.ServerClosedEvent) {
	lock.Lock()
	publishedEvents = append(publishedEvents, *e)
	lock.Unlock()
}

// setUpServers will initialize a topology's servers
// without connecting them
func setUpServers(t *testing.T, top *Topology) {
	topFunc := func(desc description.Server) {
		top.apply(context.TODO(), desc)
	}

	top.serversLock.Lock()
	for _, a := range top.cfg.seedList {
		addr := address.Address(a).Canonicalize()
		top.fsm.Servers = append(top.fsm.Servers, description.Server{Addr: addr})

		srvr, err := NewServer(addr, top.cfg.serverOpts...)
		assert.Nil(t, err, "error creating new server at address %s\n", addr)

		srvr.updateTopologyCallback.Store(topFunc)
		srvr.topologyID.Store(top.id)
		top.servers[addr] = srvr
	}
	top.serversLock.Unlock()
}

// setUpMonitoring will create a new topology and servers
// without connecting them
func setUpTopology(t *testing.T, uri string) *Topology {
	var serverOptions []ServerOption
	var options []Option
	sdam := &event.SdamMonitor{
		ServerDescriptionChanged:   serverDescriptionChanged,
		ServerOpening:              serverOpening,
		TopologyDescriptionChanged: topologyDescriptionChanged,
		TopologyOpening:            topologyOpening,
		ServerClosed:               serverClosed,
	}

	cs, err := connstring.Parse(uri)
	assert.Nil(t, err, "error parsing connection string with uri %s\n", uri)

	// configure server and topology
	serverOptions = append(
		serverOptions,
		WithServerSdamMonitor(func(*event.SdamMonitor) *event.SdamMonitor { return sdam }),
	)
	options = append(
		options,
		WithConnString(func(connstring.ConnString) connstring.ConnString { return cs }),
		WithSeedList(func(...string) []string { return cs.Hosts }),
		WithServerOptions(func(...ServerOption) []ServerOption { return serverOptions }),
		WithTopologySdamMonitor(func(*event.SdamMonitor) *event.SdamMonitor { return sdam }),
	)

	top, err := New(options...)
	assert.Nil(t, err, "error creating new topology\n")

	setUpServers(t, top)

	return top
}

func compareServerDescriptions(t *testing.T,
	e serverDescription, a event.ServerDescription) {
	assert.Equal(t, e.Address, string(a.Address),
		"expected server address %s, got %s", e.Address, a.Address)

	assert.Equal(t, len(e.Hosts), len(a.Hosts),
		"expected %d hosts, got %d", len(e.Hosts), len(a.Hosts))
	for idx, eh := range e.Hosts {
		ah := a.Hosts[idx]
		assert.Equal(t, eh, string(ah), "expected host %s, got %s", eh, ah)
	}

	assert.Equal(t, len(e.Passives), len(a.Passives),
		"expected %d hosts, got %d", len(e.Passives), len(a.Passives))
	for idx, ep := range e.Passives {
		ap := a.Passives[idx]
		assert.Equal(t, ep, string(ap), "expected passive %s, got %s", ep, ap)
	}

	assert.Equal(t, e.Primary, string(a.Primary),
		"expected primary %s, got %s", e.Primary, a.Primary)
	assert.Equal(t, e.SetName, a.SetName,
		"expected set name %s, got %s", e.SetName, a.SetName)

	// PossiblePrimary is only relevant to single-threaded drivers.
	if e.Type == "PossiblePrimary" {
		e.Type = "Unknown"
	}
	assert.Equal(t, e.Type, a.Kind.String(),
		"expected server kind %s, got %s", e.Type, a.Kind.String())
}

func compareTopologyDescriptions(t *testing.T,
	e topologyDescription, a event.TopologyDescription) {
	assert.Equal(t, e.TopologyType, a.Kind.String(),
		"expected topology kind %s, got %s", e.TopologyType, a.Kind.String())
	assert.Equal(t, len(e.Servers), len(a.Servers),
		"expected %d servers, got %d", len(e.Servers), len(a.Servers))

	for idx, es := range e.Servers {
		as := a.Servers[idx]
		compareServerDescriptions(t, es, as)
	}

	assert.Equal(t, e.SetName, a.SetName,
		"expected set name %s, got %s", e.SetName, a.SetName)
}

func compareEvents(t *testing.T, events []monitoringEvent) {
	var emptyUUID uuid.UUID

	for idx, me := range events {

		if me.TopologyOpeningEvent != nil {
			actual, ok := publishedEvents[idx].(event.TopologyOpeningEvent)
			assert.True(t, ok, "expected type %T, got %T", event.TopologyOpeningEvent{}, actual)
			assert.False(t, uuid.Equal(uuid.UUID(actual.ID), emptyUUID), "expected topology id")
		}
		if me.ServerOpeningEvent != nil {
			actual, ok := publishedEvents[idx].(event.ServerOpeningEvent)
			assert.True(t, ok, "expected type %T, got %T", event.ServerOpeningEvent{}, actual)

			evt := me.ServerOpeningEvent
			assert.Equal(t, evt.Address, string(actual.Address),
				"expected address %s, got %s", evt.Address, actual.Address)
			// TODO: assert.False(t, uuid.Equal(uuid.UUID(actual.ID), emptyUUID), "expected topology id %v", actual.ID)
		}
		if me.TopologyDescriptionChangedEvent != nil {
			actual, ok := publishedEvents[idx].(event.TopologyDescriptionChangedEvent)
			assert.True(t, ok, "expected type %T, got %T", event.TopologyDescriptionChangedEvent{}, actual)

			evt := me.TopologyDescriptionChangedEvent
			compareTopologyDescriptions(t, evt.PreviousDescription, actual.PreviousDescription)
			compareTopologyDescriptions(t, evt.NewDescription, actual.NewDescription)
			assert.False(t, uuid.Equal(uuid.UUID(actual.ID), emptyUUID), "expected topology id")
		}
		if me.ServerDescriptionChangedEvent != nil {
			actual, ok := publishedEvents[idx].(event.ServerDescriptionChangedEvent)
			assert.True(t, ok, "expected type %T, got %T", event.ServerDescriptionChangedEvent{}, actual)

			evt := me.ServerDescriptionChangedEvent
			assert.Equal(t, evt.Address, string(actual.Address),
				"expected server address %s, got %s", evt.Address, actual.Address)
			compareServerDescriptions(t, evt.PreviousDescription, actual.PreviousDescription)
			compareServerDescriptions(t, evt.NewDescription, actual.NewDescription)
			assert.False(t, uuid.Equal(uuid.UUID(actual.ID), emptyUUID), "expected topology id")
		}
		if me.ServerClosedEvent != nil {
			actual, ok := publishedEvents[idx].(event.ServerClosedEvent)
			assert.True(t, ok, "expected type %T, got %T", event.ServerClosedEvent{}, actual)

			evt := me.ServerClosedEvent
			assert.Equal(t, evt.Address, string(actual.Address),
				"expected server address %s, got %s", evt.Address, actual.Address)
			assert.False(t, uuid.Equal(uuid.UUID(actual.ID), emptyUUID), "expected topology id")
		}
	}

}

func runMonitoringTest(t *testing.T, directory string, filename string) {
	filepath := path.Join(testsDir, directory, filename)
	content, err := ioutil.ReadFile(filepath)
	assert.Nil(t, err, "error reading file %v\n", err)

	// Remove ".json" from filename.
	filename = filename[:len(filename)-5]
	testName := directory + "/" + filename + ":"

	t.Run(testName, func(t *testing.T) {
		var test monitoringTestCase

		assert.Nil(t, json.Unmarshal(content, &test), "error unmarshaling json\n")

		// create topology and servers
		top := setUpTopology(t, test.URI)

		for _, phase := range test.Phases {
			for _, response := range phase.Responses {
				doc, err := bson.Marshal(response.IsMaster)
				assert.Nil(t, err, "error marshaling bson\n")

				serverDesc := description.NewServer(address.Address(response.Host), bsoncore.Document(doc))
				srvr := top.servers[address.Address(response.Host)]
				srvr.updateDescription(serverDesc, false)
			}

			lock.Lock()
			defer lock.Unlock()

			assert.Equal(t, len(phase.Outcome.Events), len(publishedEvents),
				"expected %d published events, got %d\n",
				len(phase.Outcome.Events), len(publishedEvents))

			compareEvents(t, phase.Outcome.Events)
		}
		publishedEvents = nil
	})
}

// Test case for all SDAM spec tests.
func TestSDAMMonitoringSpec(t *testing.T) {
	for _, file := range testhelpers.FindJSONFilesInDir(t, path.Join(testsDir, "monitoring")) {
		runMonitoringTest(t, "monitoring", file)
	}
}
