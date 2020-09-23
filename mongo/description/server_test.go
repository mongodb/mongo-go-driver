// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package description

import (
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/tag"
)

func TestServer(t *testing.T) {
	t.Run("equals", func(t *testing.T) {
		defaultServer := Server{}
		// Only some of the Server fields affect equality
		testCases := []struct {
			name   string
			server Server
			equal  bool
		}{
			{"empty", Server{}, true},
			{"address", Server{Addr: address.Address("foo")}, true},
			{"arbiters", Server{Arbiters: []string{"foo"}}, false},
			{"rtt", Server{AverageRTT: time.Second}, true},
			{"compression", Server{Compression: []string{"foo"}}, true},
			{"canonicalAddr", Server{CanonicalAddr: address.Address("foo")}, false},
			{"electionID", Server{ElectionID: primitive.NewObjectID()}, false},
			{"heartbeatInterval", Server{HeartbeatInterval: time.Second}, true},
			{"hosts", Server{Hosts: []string{"foo"}}, false},
			{"lastError", Server{LastError: errors.New("foo")}, false},
			{"lastUpdateTime", Server{LastUpdateTime: time.Now()}, true},
			{"lastWriteTime", Server{LastWriteTime: time.Now()}, true},
			{"maxBatchCount", Server{MaxBatchCount: 1}, true},
			{"maxDocumentSize", Server{MaxDocumentSize: 1}, true},
			{"maxMessageSize", Server{MaxMessageSize: 1}, true},
			{"members", Server{Members: []address.Address{address.Address("foo")}}, true},
			{"passives", Server{Passives: []string{"foo"}}, false},
			{"primary", Server{Primary: address.Address("foo")}, false},
			{"readOnly", Server{ReadOnly: true}, true},
			{"sessionTimeoutMinutes", Server{SessionTimeoutMinutes: 1}, false},
			{"setName", Server{SetName: "foo"}, false},
			{"setVersion", Server{SetVersion: 1}, false},
			{"tags", Server{Tags: tag.Set{tag.Tag{"foo", "bar"}}}, false},
			{"topologyVersion", Server{TopologyVersion: &TopologyVersion{primitive.NewObjectID(), 0}}, false},
			{"kind", Server{Kind: Standalone}, false},
			{"wireVersion", Server{WireVersion: &VersionRange{1, 2}}, false},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				actual := defaultServer.Equal(tc.server)
				assert.Equal(t, actual, tc.equal, "expected %v, got %v", tc.equal, actual)
			})
		}
	})
}
