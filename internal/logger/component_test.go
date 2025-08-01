// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package logger

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/assert"
)

func verifySerialization(t *testing.T, got, want KeyValues) {
	t.Helper()

	for i := 0; i < len(got); i += 2 {
		assert.Equal(t, want[i], got[i], "key position mismatch")
		assert.Equal(t, want[i+1], got[i+1], "value position mismatch for %q", want[i])
	}
}

func TestSerializeCommand(t *testing.T) {
	t.Parallel()

	serverConnectionID := int64(100)
	serviceID := bson.NewObjectID()

	tests := []struct {
		name               string
		cmd                Command
		extraKeysAndValues []any
		want               KeyValues
	}{
		{
			name: "empty",
			want: KeyValues{
				KeyCommandName, "",
				KeyDatabaseName, "",
				KeyDriverConnectionID, int64(0),
				KeyMessage, "",
				KeyOperationID, int32(0),
				KeyRequestID, int64(0),
				KeyServerHost, "",
			},
		},
		{
			name: "complete Command object",
			cmd: Command{
				DriverConnectionID: 1,
				Name:               "foo",
				DatabaseName:       "db",
				Message:            "bar",
				OperationID:        2,
				RequestID:          3,
				ServerHost:         "localhost",
				ServerPort:         "27017",
				ServerConnectionID: &serverConnectionID,
				ServiceID:          &serviceID,
			},
			want: KeyValues{
				KeyCommandName, "foo",
				KeyDatabaseName, "db",
				KeyDriverConnectionID, int64(1),
				KeyMessage, "bar",
				KeyOperationID, int32(2),
				KeyRequestID, int64(3),
				KeyServerHost, "localhost",
				KeyServerPort, int64(27017),
				KeyServerConnectionID, serverConnectionID,
				KeyServiceID, serviceID.Hex(),
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := SerializeCommand(test.cmd, test.extraKeysAndValues...)
			verifySerialization(t, got, test.want)
		})
	}
}

func TestSerializeConnection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		conn               Connection
		extraKeysAndValues []any
		want               KeyValues
	}{
		{
			name: "empty",
			want: KeyValues{
				KeyMessage, "",
				KeyServerHost, "",
			},
		},
		{
			name: "complete Connection object",
			conn: Connection{
				Message:    "foo",
				ServerHost: "localhost",
				ServerPort: "27017",
			},
			want: KeyValues{
				"message", "foo",
				"serverHost", "localhost",
				"serverPort", int64(27017),
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := SerializeConnection(test.conn, test.extraKeysAndValues...)
			verifySerialization(t, got, test.want)
		})
	}
}

func TestSerializeServer(t *testing.T) {
	t.Parallel()

	topologyID := bson.NewObjectID()
	serverConnectionID := int64(100)

	tests := []struct {
		name               string
		srv                Server
		extraKeysAndValues []any
		want               KeyValues
	}{
		{
			name: "empty",
			want: KeyValues{
				KeyDriverConnectionID, int64(0),
				KeyMessage, "",
				KeyServerHost, "",
				KeyTopologyID, bson.ObjectID{}.Hex(),
			},
		},
		{
			name: "complete Server object",
			srv: Server{
				DriverConnectionID: 1,
				TopologyID:         topologyID,
				Message:            "foo",
				ServerConnectionID: &serverConnectionID,
				ServerHost:         "localhost",
				ServerPort:         "27017",
			},
			want: KeyValues{
				KeyDriverConnectionID, int64(1),
				KeyMessage, "foo",
				KeyServerHost, "localhost",
				KeyTopologyID, topologyID.Hex(),
				KeyServerConnectionID, serverConnectionID,
				KeyServerPort, int64(27017),
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := SerializeServer(test.srv, test.extraKeysAndValues...)
			verifySerialization(t, got, test.want)
		})
	}

}

func TestSerializeTopology(t *testing.T) {
	t.Parallel()

	topologyID := bson.NewObjectID()

	tests := []struct {
		name               string
		topo               Topology
		extraKeysAndValues []any
		want               KeyValues
	}{
		{
			name: "empty",
			want: KeyValues{
				KeyTopologyID, bson.ObjectID{}.Hex(),
			},
		},
		{
			name: "complete Server object",
			topo: Topology{
				ID:      topologyID,
				Message: "foo",
			},
			want: KeyValues{
				KeyTopologyID, topologyID.Hex(),
				KeyMessage, "foo",
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := SerializeTopology(test.topo, test.extraKeysAndValues...)
			verifySerialization(t, got, test.want)
		})
	}

}
