package mtest

import (
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/x/mongo/driver/topology"
)

// AuthEnabled returns whether or not this test is running in an environment with auth.
func AuthEnabled() bool {
	return testContext.authEnabled
}

// SSLEnabled returns whether or not this test is running in an environment with SSL.
func SSLEnabled() bool {
	return testContext.sslEnabled
}

// ClusterTopologyKind returns the topology kind of the cluster under test.
func ClusterTopologyKind() TopologyKind {
	return testContext.topoKind
}

// ClusterURI returns the connection string used to create the client for this test.
func ClusterURI() string {
	return testContext.connString.Original
}

// GlobalClient returns a client configured with read concern majority, write concern majority, and read preference
// primary. The returned client is not tied to the receiver and is valid outside the lifetime of the receiver.
func (*T) GlobalClient() *mongo.Client {
	return testContext.client
}

// GlobalTopology returns the Topology backing the global Client.
func (*T) GlobalTopology() *topology.Topology {
	return testContext.topo
}

// ServerVersion returns the server version of the cluster. This assumes that all nodes in the cluster have the same
// version.
func (*T) ServerVersion() string {
	return testContext.serverVersion
}
