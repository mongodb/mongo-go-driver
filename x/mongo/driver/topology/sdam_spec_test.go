// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	testhelpers "go.mongodb.org/mongo-driver/internal/testutil/helpers"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
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
}

type lastWriteDate struct {
	LastWriteDate time.Time `bson:"lastWriteDate"`
}

type server struct {
	Type           string
	SetName        string
	SetVersion     uint32
	ElectionID     *primitive.ObjectID `bson:"electionId"`
	MinWireVersion *int32
	MaxWireVersion *int32
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
	Responses []response
	Outcome   outcome
}

type testCase struct {
	Description string
	URI         string
	Phases      []phase
}

const testsDir string = "../../../../data/server-discovery-and-monitoring/"

func (r *response) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&r.Host, &r.IsMaster}
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return err
	}

	if len(tmp) != 2 {
		return fmt.Errorf("'response' JSON array must have exactly two elements")
	}

	return nil
}

func setUpFSM(t *testing.T, uri string) *fsm {
	fsm := newFSM()

	cs, err := connstring.Parse(uri)
	assert.Nil(t, err, "Parse error: %v", err)

	fsm.SetName = cs.ReplicaSet
	if fsm.SetName != "" {
		fsm.Kind = description.ReplicaSetNoPrimary
	} else if len(cs.Hosts) == 1 {
		fsm.Kind = description.Single
	}

	for _, host := range cs.Hosts {
		fsm.Servers = append(fsm.Servers, description.Server{Addr: address.Address(host).Canonicalize()})
	}

	return fsm
}

func applyResponses(f *fsm, responses []response) error {
	for _, response := range responses {
		doc, err := bson.Marshal(response.IsMaster)
		if err != nil {
			return err
		}
		server := description.NewServer(address.Address(response.Host), bsoncore.Document(doc))
		_, err = f.apply(server)

		if err != nil {
			return err
		}
	}

	return nil
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
		err = json.Unmarshal(content, &test)
		assert.Nil(t, err, "Unmarshal error: %v", err)
		f := setUpFSM(t, test.URI)

		for _, phase := range test.Phases {
			err = applyResponses(f, phase.Responses)
			if phase.Outcome.Compatible == nil || *phase.Outcome.Compatible {
				assert.Nil(t, err, "error: %v", err)
			} else {
				assert.NotNil(t, err, "Expected error")
				continue
			}

			assert.Equal(t, phase.Outcome.TopologyType, f.Kind.String(),
				"expected TopologyType to be %v, got %v", phase.Outcome.TopologyType, f.Kind.String())
			assert.Equal(t, phase.Outcome.SetName, f.SetName,
				"expected SetName to be %v, got %v", phase.Outcome.SetName, f.SetName)
			assert.Equal(t, len(phase.Outcome.Servers), len(f.Servers),
				"expected %v servers, got %v", len(phase.Outcome.Servers), len(f.Servers))
			assert.Equal(t, phase.Outcome.LogicalSessionTimeoutMinutes, f.SessionTimeoutMinutes,
				"expected SessionTimeoutMinutes to be %v, got %v", phase.Outcome.LogicalSessionTimeoutMinutes, f.SessionTimeoutMinutes)
			assert.Equal(t, phase.Outcome.MaxSetVersion, f.maxSetVersion,
				"expected maxSetVersion to be %v, got %v", phase.Outcome.MaxSetVersion, f.maxSetVersion)
			assert.Equal(t, phase.Outcome.MaxElectionID, f.maxElectionID,
				"expected maxElectionID to be %v, got %v", phase.Outcome.MaxElectionID, f.maxElectionID)

			for addr, server := range phase.Outcome.Servers {
				fsmServer, ok := f.Server(address.Address(addr))
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

				// PossiblePrimary is only relevant to single-threaded drivers.
				if server.Type == "PossiblePrimary" {
					server.Type = "Unknown"
				}

				assert.Equal(t, server.Type, fsmServer.Kind.String(),
					"expected server Type to be %v, got %v", server.Type, fsmServer.Kind.String())
			}
		}
	})
}

// Test case for all SDAM spec tests.
func TestSDAMSpec(t *testing.T) {
	for _, subdir := range []string{"single", "rs", "sharded"} {
		for _, file := range testhelpers.FindJSONFilesInDir(t, path.Join(testsDir, subdir)) {
			runTest(t, subdir, file)
		}
	}
}
