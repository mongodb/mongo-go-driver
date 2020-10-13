// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mtest

import (
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

// AuthEnabled returns whether or not the cluster requires auth.
func AuthEnabled() bool {
	return testContext.authEnabled
}

// SSLEnabled returns whether or not the cluster requires SSL.
func SSLEnabled() bool {
	return testContext.sslEnabled
}

// ClusterTopologyKind returns the topology kind of the cluster under test.
func ClusterTopologyKind() TopologyKind {
	return testContext.topoKind
}

// ClusterURI returns the connection string for the cluster.
func ClusterURI() string {
	return testContext.connString.Original
}

// GlobalClient returns a Client connected to the cluster configured with read concern majority, write concern majority,
// and read preference primary.
func GlobalClient() *mongo.Client {
	return testContext.client
}

// GlobalTopology returns a Topology that's connected to the cluster.
func GlobalTopology() *topology.Topology {
	return testContext.topo
}

// ServerVersion returns the server version of the cluster. This assumes that all nodes in the cluster have the same
// version.
func ServerVersion() string {
	return testContext.serverVersion
}
