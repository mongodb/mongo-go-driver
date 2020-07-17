// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	testhelpers "go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/address"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
	"go.mongodb.org/mongo-driver/x/mongo/driver/description"
)

type response struct {
	Host     string
	IsMaster IsMaster
}

type IsMaster struct {
	Arbiters                     []string           `bson:"arbiters,omitempty"`
	ArbiterOnly                  bool               `bson:"arbiterOnly,omitempty"`
	ClusterTime                  bson.Raw           `bson:"$clusterTime,omitempty"`
	Compression                  []string           `bson:"compression,omitempty"`
	ElectionID                   primitive.ObjectID `bson:"electionId,omitempty"`
	Hidden                       bool               `bson:"hidden,omitempty"`
	Hosts                        []string           `bson:"hosts,omitempty"`
	IsMaster                     bool               `bson:"ismaster,omitempty"`
	IsReplicaSet                 bool               `bson:"isreplicaset,omitempty"`
	LastWrite                    *lastWriteDate     `bson:"lastWrite,omitempty"`
	LogicalSessionTimeoutMinutes uint32             `bson:"logicalSessionTimeoutMinutes,omitempty"`
	MaxBSONObjectSize            uint32             `bson:"maxBsonObjectSize,omitempty"`
	MaxMessageSizeBytes          uint32             `bson:"maxMessageSizeBytes,omitempty"`
	MaxWriteBatchSize            uint32             `bson:"maxWriteBatchSize,omitempty"`
	Me                           string             `bson:"me,omitempty"`
	MaxWireVersion               int32              `bson:"maxWireVersion,omitempty"`
	MinWireVersion               int32              `bson:"minWireVersion,omitempty"`
	Msg                          string             `bson:"msg,omitempty"`
	OK                           int32              `bson:"ok"`
	Passives                     []string           `bson:"passives,omitempty"`
	ReadOnly                     bool               `bson:"readOnly,omitempty"`
	SaslSupportedMechs           []string           `bson:"saslSupportedMechs,omitempty"`
	Secondary                    bool               `bson:"secondary,omitempty"`
	SetName                      string             `bson:"setName,omitempty"`
	SetVersion                   uint32             `bson:"setVersion,omitempty"`
	Tags                         map[string]string  `bson:"tags,omitempty"`
	TopologyVersion              *topologyVersion   `bson:"topologyVersion,omitempty"`
}

type lastWriteDate struct {
	LastWriteDate time.Time `bson:"lastWriteDate"`
}

type server struct {
	Type            string
	SetName         string
	SetVersion      uint32
	ElectionID      *primitive.ObjectID `bson:"electionId"`
	MinWireVersion  *int32
	MaxWireVersion  *int32
	TopologyVersion *topologyVersion
	Pool            *testPool
}

type topologyVersion struct {
	ProcessID primitive.ObjectID `bson:"processId"`
	Counter   int64
}

type testPool struct {
	Generation uint64
}

type applicationError struct {
	Address        string
	Generation     *uint64
	MaxWireVersion *int32
	When           string
	Type           string
	Response       bsoncore.Document
}

type outcome struct {
	Servers                      map[string]server
	TopologyType                 string
	SetName                      string
	LogicalSessionTimeoutMinutes uint32
	MaxSetVersion                uint32
	MaxElectionID                primitive.ObjectID `bson:"maxElectionId"`
	Compatible                   *bool
}

type phase struct {
	Description       string
	Responses         []response
	ApplicationErrors []applicationError
	Outcome           outcome
}

type testCase struct {
	Description string
	URI         string
	Phases      []phase
}

const testsDir string = "../../../../data/server-discovery-and-monitoring/"

func (r *response) UnmarshalBSON(buf []byte) error {
	doc := bson.Raw(buf)
	if err := doc.Index(0).Value().Unmarshal(&r.Host); err != nil {
		return fmt.Errorf("error unmarshalling Host: %v", err)
	}

	if err := doc.Index(1).Value().Unmarshal(&r.IsMaster); err != nil {
		return fmt.Errorf("error unmarshalling IsMaster: %v", err)
	}

	return nil
}

func setUpTopology(t *testing.T, uri string) *Topology {
	cs, err := connstring.ParseAndValidate(uri)
	assert.Nil(t, err, "Parse error: %v", err)

	// Disable server monitoring because the hosts in the SDAM spec tests don't actually exist, so the server monitor
	// can race with the test and mark the server Unknown when it fails to connect, which causes tests to fail.
	serverOpts := []ServerOption{
		withMonitoringDisabled(func(bool) bool {
			return true
		}),
	}
	topo, err := New(
		WithConnString(func(connstring.ConnString) connstring.ConnString {
			return cs
		}),
		WithServerOptions(func(opts ...ServerOption) []ServerOption {
			return append(opts, serverOpts...)
		}),
	)
	assert.Nil(t, err, "topology.New error: %v", err)

	// add servers to topology without starting heartbeat goroutines
	topo.serversLock.Lock()
	for _, a := range topo.cfg.seedList {
		addr := address.Address(a).Canonicalize()
		topo.fsm.Servers = append(topo.fsm.Servers, description.Server{Addr: addr})

		svr, err := NewServer(addr, topo.cfg.serverOpts...)
		assert.Nil(t, err, "NewServer error: %v", err)
		atomic.StoreInt32(&svr.connectionstate, connected)
		svr.desc.Store(description.NewDefaultServer(svr.address))
		svr.updateTopologyCallback.Store(topo.updateCallback)

		topo.servers[addr] = svr
	}
	topo.desc.Store(description.Topology{Servers: topo.fsm.Servers})
	topo.serversLock.Unlock()

	atomic.StoreInt32(&topo.connectionstate, connected)

	return topo
}

func applyResponses(t *testing.T, topo *Topology, responses []response, sub *driver.Subscription) {
	select {
	case <-sub.Updates:
	default:
	}
	for _, response := range responses {
		doc, err := bson.Marshal(response.IsMaster)
		assert.Nil(t, err, "Marshal error: %v", err)

		addr := address.Address(response.Host)
		desc := description.NewServer(addr, bsoncore.Document(doc))
		server, ok := topo.servers[addr]
		if ok {
			server.updateDescription(desc)
		} else {
			// for tests that check that server descriptions that aren't in the topology aren't applied
			topo.apply(context.Background(), desc)
		}
		select {
		case <-sub.Updates:
		default:
			return
		}
	}
}

type netErr struct {
	timeout bool
}

func (n netErr) Error() string {
	return "error"
}

func (n netErr) Timeout() bool {
	return n.timeout
}

func (n netErr) Temporary() bool {
	return false
}

var _ net.Error = (*netErr)(nil)

// helper method to create an error from the test response
func getError(rdr bsoncore.Document) error {
	var errmsg, codeName string
	var code int32
	var labels []string
	var tv *description.TopologyVersion
	var wcError driver.WriteCommandError
	elems, err := rdr.Elements()
	if err != nil {
		return err
	}

	for _, elem := range elems {
		switch elem.Key() {
		case "ok":
			switch elem.Value().Type {
			case bson.TypeInt32:
				if elem.Value().Int32() == 1 {
					return nil
				}
			case bson.TypeInt64:
				if elem.Value().Int64() == 1 {
					return nil
				}
			case bson.TypeDouble:
				if elem.Value().Double() == 1 {
					return nil
				}
			}
		case "errmsg":
			if str, okay := elem.Value().StringValueOK(); okay {
				errmsg = str
			}
		case "codeName":
			if str, okay := elem.Value().StringValueOK(); okay {
				codeName = str
			}
		case "code":
			if c, okay := elem.Value().Int32OK(); okay {
				code = c
			}
		case "errorLabels":
			if arr, okay := elem.Value().ArrayOK(); okay {
				elems, err := arr.Elements()
				if err != nil {
					continue
				}
				for _, elem := range elems {
					if str, ok := elem.Value().StringValueOK(); ok {
						labels = append(labels, str)
					}
				}

			}
		case "writeErrors":
			arr, exists := elem.Value().ArrayOK()
			if !exists {
				break
			}
			vals, err := arr.Values()
			if err != nil {
				continue
			}
			for _, val := range vals {
				var we driver.WriteError
				doc, exists := val.DocumentOK()
				if !exists {
					continue
				}
				if index, exists := doc.Lookup("index").AsInt64OK(); exists {
					we.Index = index
				}
				if code, exists := doc.Lookup("code").AsInt64OK(); exists {
					we.Code = code
				}
				if msg, exists := doc.Lookup("errmsg").StringValueOK(); exists {
					we.Message = msg
				}
				wcError.WriteErrors = append(wcError.WriteErrors, we)
			}
		case "writeConcernError":
			doc, exists := elem.Value().DocumentOK()
			if !exists {
				break
			}
			wcError.WriteConcernError = new(driver.WriteConcernError)
			if code, exists := doc.Lookup("code").AsInt64OK(); exists {
				wcError.WriteConcernError.Code = code
			}
			if name, exists := doc.Lookup("codeName").StringValueOK(); exists {
				wcError.WriteConcernError.Name = name
			}
			if msg, exists := doc.Lookup("errmsg").StringValueOK(); exists {
				wcError.WriteConcernError.Message = msg
			}
			if info, exists := doc.Lookup("errInfo").DocumentOK(); exists {
				wcError.WriteConcernError.Details = make([]byte, len(info))
				copy(wcError.WriteConcernError.Details, info)
			}
			if errLabels, exists := doc.Lookup("errorLabels").ArrayOK(); exists {
				elems, err := errLabels.Elements()
				if err != nil {
					continue
				}
				for _, elem := range elems {
					if str, ok := elem.Value().StringValueOK(); ok {
						labels = append(labels, str)
					}
				}
			}
		case "topologyVersion":
			doc, ok := elem.Value().DocumentOK()
			if !ok {
				break
			}
			version, err := description.NewTopologyVersion(doc)
			if err == nil {
				tv = version
			}
		}
	}

	if errmsg == "" {
		errmsg = "command failed"
	}

	return driver.Error{
		Code:            code,
		Message:         errmsg,
		Name:            codeName,
		Labels:          labels,
		TopologyVersion: tv,
	}
}

func applyErrors(t *testing.T, topo *Topology, errors []applicationError) {
	for _, appErr := range errors {
		var currError error
		switch appErr.Type {
		case "command":
			currError = getError(appErr.Response)
		case "network":
			currError = driver.Error{
				Labels:  []string{driver.NetworkError},
				Wrapped: ConnectionError{Wrapped: netErr{false}},
			}
		case "timeout":
			currError = driver.Error{
				Labels:  []string{driver.NetworkError},
				Wrapped: ConnectionError{Wrapped: netErr{true}},
			}
		default:
			t.Fatalf("unrecognized error type: %v", appErr.Type)
		}
		server, ok := topo.servers[address.Address(appErr.Address)]
		assert.True(t, ok, "server not found: %v", appErr.Address)

		desc := server.Description()
		versionRange := description.NewVersionRange(0, *appErr.MaxWireVersion)
		desc.WireVersion = &versionRange

		generation := atomic.LoadUint64(&server.pool.generation)
		if appErr.Generation != nil {
			generation = uint64(*appErr.Generation)
		}
		//use generation number to check conn stale
		innerConn := connection{
			desc:       desc,
			generation: generation,
			pool:       server.pool,
		}
		conn := Connection{connection: &innerConn}

		switch appErr.When {
		case "beforeHandshakeCompletes":
			server.ProcessHandshakeError(currError, generation)
		case "afterHandshakeCompletes":
			server.ProcessError(currError, &conn)
		default:
			t.Fatalf("unrecognized applicationError.When value: %v", appErr.When)
		}
	}
}

func runTest(t *testing.T, directory string, filename string) {
	filepath := path.Join(testsDir, directory, filename)
	content, err := ioutil.ReadFile(filepath)
	assert.Nil(t, err, "ReadFile error: %v", err)

	// Remove ".json" from filename.
	filename = filename[:len(filename)-5]
	testName := directory + "/" + filename + ":"

	t.Run(testName, func(t *testing.T) {
		var test testCase
		err = bson.UnmarshalExtJSON(content, false, &test)
		assert.Nil(t, err, "Unmarshal error: %v", err)
		topo := setUpTopology(t, test.URI)
		sub, err := topo.Subscribe()
		assert.Nil(t, err, "subscribe error: %v", err)

		for _, phase := range test.Phases {
			applyResponses(t, topo, phase.Responses, sub)
			applyErrors(t, topo, phase.ApplicationErrors)
			if phase.Outcome.Compatible == nil || *phase.Outcome.Compatible {
				assert.True(t, topo.fsm.compatible.Load().(bool), "Expected servers to be compatible")
			} else {
				assert.False(t, topo.fsm.compatible.Load().(bool), "Expected servers to not be compatible")
				continue
			}
			desc := topo.Description()

			assert.Equal(t, phase.Outcome.TopologyType, desc.Kind.String(),
				"expected TopologyType to be %v, got %v", phase.Outcome.TopologyType, desc.Kind.String())
			assert.Equal(t, phase.Outcome.SetName, topo.fsm.SetName,
				"expected SetName to be %v, got %v", phase.Outcome.SetName, topo.fsm.SetName)
			assert.Equal(t, len(phase.Outcome.Servers), len(desc.Servers),
				"expected %v servers, got %v", len(phase.Outcome.Servers), len(desc.Servers))
			assert.Equal(t, phase.Outcome.LogicalSessionTimeoutMinutes, desc.SessionTimeoutMinutes,
				"expected SessionTimeoutMinutes to be %v, got %v", phase.Outcome.LogicalSessionTimeoutMinutes, desc.SessionTimeoutMinutes)
			assert.Equal(t, phase.Outcome.MaxSetVersion, topo.fsm.maxSetVersion,
				"expected maxSetVersion to be %v, got %v", phase.Outcome.MaxSetVersion, topo.fsm.maxSetVersion)
			assert.Equal(t, phase.Outcome.MaxElectionID, topo.fsm.maxElectionID,
				"expected maxElectionID to be %v, got %v", phase.Outcome.MaxElectionID, topo.fsm.maxElectionID)

			for addr, server := range phase.Outcome.Servers {
				fsmServer, ok := desc.Server(address.Address(addr))
				assert.True(t, ok, "Couldn't find server %v", addr)

				assert.Equal(t, address.Address(addr), fsmServer.Addr,
					"expected server address to be %v, got %v", address.Address(addr), fsmServer.Addr)
				assert.Equal(t, server.SetName, fsmServer.SetName,
					"expected server SetName to be %v, got %v", server.SetName, fsmServer.SetName)
				assert.Equal(t, server.SetVersion, fsmServer.SetVersion,
					"expected server SetVersion to be %v, got %v", server.SetVersion, fsmServer.SetVersion)
				if server.ElectionID != nil {
					assert.Equal(t, *server.ElectionID, fsmServer.ElectionID,
						"expected server ElectionID to be %v, got %v", *server.ElectionID, fsmServer.ElectionID)
				}
				if server.TopologyVersion != nil {

					assert.NotNil(t, fsmServer.TopologyVersion, "expected server TopologyVersion not to be nil")
					assert.Equal(t, server.TopologyVersion.ProcessID, fsmServer.TopologyVersion.ProcessID,
						"expected server TopologyVersion ProcessID to be %v, got %v", server.TopologyVersion.ProcessID, fsmServer.TopologyVersion.ProcessID)
					assert.Equal(t, server.TopologyVersion.Counter, fsmServer.TopologyVersion.Counter,
						"expected server TopologyVersion Counter to be %v, got %v", server.TopologyVersion.Counter, fsmServer.TopologyVersion.Counter)
				} else {
					assert.Nil(t, fsmServer.TopologyVersion, "expected server TopologyVersion to be nil")
				}

				// PossiblePrimary is only relevant to single-threaded drivers.
				if server.Type == "PossiblePrimary" {
					server.Type = "Unknown"
				}

				assert.Equal(t, server.Type, fsmServer.Kind.String(),
					"expected server Type to be %v, got %v", server.Type, fsmServer.Kind.String())
				if server.Pool != nil {
					topo.serversLock.Lock()
					actualServer := topo.servers[address.Address(addr)]
					topo.serversLock.Unlock()
					actualGeneration := atomic.LoadUint64(&actualServer.pool.generation)
					assert.Equal(t, server.Pool.Generation, actualGeneration,
						"expected server pool generation to be %v, got %v", server.Pool.Generation, actualGeneration)
				}
			}
		}
	})
}

// Test case for all SDAM spec tests.
func TestSDAMSpec(t *testing.T) {
	for _, subdir := range []string{"single", "rs", "sharded", "errors"} {
		for _, file := range testhelpers.FindJSONFilesInDir(t, path.Join(testsDir, subdir)) {
			runTest(t, subdir, file)
		}
	}
}
