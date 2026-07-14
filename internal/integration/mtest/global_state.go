// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mtest

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/failpoint"
	"go.mongodb.org/mongo-driver/v2/internal/integtest"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/connstring"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/topology"
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

// Serverless returns whether the test is running against a serverless instance.
func Serverless() bool {
	return testContext.serverless
}

// SingleMongosLoadBalancerURI returns the URI for a load balancer fronting a single mongos. This will only be set
// if the cluster is load balanced.
func SingleMongosLoadBalancerURI() string {
	return testContext.singleMongosLoadBalancerURI
}

// MultiMongosLoadBalancerURI returns the URI for a load balancer fronting multiple mongoses. This will only be set
// if the cluster is load balanced.
func MultiMongosLoadBalancerURI() string {
	return testContext.multiMongosLoadBalancerURI
}

// ClusterConnString returns the parsed ConnString for the cluster.
func ClusterConnString() *connstring.ConnString {
	return testContext.connString
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

// SetFailPoint configures the provided fail point on the cluster under test using the provided Client.
func SetFailPoint(fp failpoint.FailPoint, client *mongo.Client) error {
	admin := client.Database("admin")
	if err := admin.RunCommand(context.Background(), fp).Err(); err != nil {
		return fmt.Errorf("error creating fail point: %w", err)
	}
	return nil
}

// SetRawFailPoint configures the fail point represented by the fp parameter on the cluster under test using the
// provided Client
func SetRawFailPoint(fp bson.Raw, client *mongo.Client) error {
	admin := client.Database("admin")
	if err := admin.RunCommand(context.Background(), fp).Err(); err != nil {
		return fmt.Errorf("error creating fail point: %w", err)
	}
	return nil
}

// AdvanceConfigClusterTime advances the config server's cluster time to the
// mongos cluster time so a change stream open doesn't block waiting for it.
func AdvanceConfigClusterTime(ctx context.Context) error {
	admin := GlobalClient().Database("admin")

	pingRes, err := admin.RunCommand(ctx, bson.D{{Key: "ping", Value: 1}}).Raw()
	if err != nil {
		return fmt.Errorf("error running ping command: %w", err)
	}
	if _, err := pingRes.LookupErr("$clusterTime"); err != nil {
		return fmt.Errorf("error looking up $clusterTime in ping response: %w", err)
	}

	var shardMap struct {
		Map struct {
			Config string `bson:"config"`
		} `bson:"map"`
	}
	if err := admin.RunCommand(ctx, bson.D{{Key: "getShardMap", Value: 1}}).Decode(&shardMap); err != nil {
		return fmt.Errorf("error running getShardMap command: %w", err)
	}
	_, hosts, ok := strings.Cut(shardMap.Map.Config, "/")
	if !ok {
		return fmt.Errorf("error parsing config shard map: %v", shardMap)
	}
	configHost, _, _ := strings.Cut(hosts, ",")

	clientOpts := options.Client().
		ApplyURI(ClusterURI()).
		SetHosts([]string{configHost}).
		SetDirect(true)
	integtest.AddTestServerAPIVersion(clientOpts)

	cfgClient, err := mongo.Connect(clientOpts)
	if err != nil {
		return fmt.Errorf("error connecting to config server: %w", err)
	}
	defer func() { _ = cfgClient.Disconnect(ctx) }()

	note := bson.D{{Key: "cursor-test", Value: "advance config server cluster time"}}
	return cfgClient.Database("admin").RunCommand(ctx,
		bson.D{{Key: "appendOplogNote", Value: 1}, {Key: "data", Value: note}}).Err()
}
