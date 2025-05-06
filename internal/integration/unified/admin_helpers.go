// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/internal/integration/mtest"
	"go.mongodb.org/mongo-driver/v2/internal/integtest"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ignoredKillAllSessionsErrors = []int{
		11601, // Interrupted, for SERVER-38335 on server versions below 4.2
		13,    // Unauthorized, for SERVER-54216 on atlas
	}
)

// terminateOpenSessions executes a killAllSessions command to ensure that sessions left open on the server by a test
// do not cause future tests to hang.
func terminateOpenSessions(ctx context.Context) error {
	if mtest.CompareServerVersions(mtest.ServerVersion(), "3.6") < 0 {
		return nil
	}

	commandFn := func(ctx context.Context, client *mongo.Client) error {
		cmd := bson.D{
			{"killAllSessions", bson.A{}},
		}

		err := client.Database("admin").RunCommand(ctx, cmd).Err()
		if se, ok := err.(mongo.ServerError); ok {
			for _, code := range ignoredKillAllSessionsErrors {
				if se.HasErrorCode(code) {
					err = nil
					break
				}
			}
		}

		// for SERVER-54216 on atlas
		if err != nil && strings.Contains(err.Error(), "(AtlasError) (Unauthorized)") {
			err = nil
		}

		return err
	}

	// For sharded clusters, this has to run against all mongos nodes. Otherwise, it can just against on the primary.
	if mtest.ClusterTopologyKind() != mtest.Sharded {
		return commandFn(ctx, mtest.GlobalClient())
	}
	return runAgainstAllMongoses(ctx, commandFn)
}

// performDistinctWorkaround executes a non-transactional "distinct" command against each mongos in a sharded cluster.
func performDistinctWorkaround(ctx context.Context) error {
	commandFn := func(ctx context.Context, client *mongo.Client) error {
		for _, coll := range entities(ctx).collections() {
			newColl := client.Database(coll.Database().Name()).Collection(coll.Name())
			err := newColl.Distinct(ctx, "x", bson.D{}).Err()
			if err != nil {
				ns := fmt.Sprintf("%s.%s", coll.Database().Name(), coll.Name())
				return fmt.Errorf("error running distinct for collection %q: %w", ns, err)
			}
		}

		return nil
	}

	return runAgainstAllMongoses(ctx, commandFn)
}

func runCommandOnHost(ctx context.Context, host string, commandFn func(context.Context, *mongo.Client) error) error {
	clientOpts := options.Client().
		ApplyURI(mtest.ClusterURI()).
		SetHosts([]string{host})
	integtest.AddTestServerAPIVersion(clientOpts)

	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return fmt.Errorf("error creating client to host %q: %w", host, err)
	}
	defer func() { _ = client.Disconnect(ctx) }()

	return commandFn(ctx, client)
}

func runAgainstAllMongoses(ctx context.Context, commandFn func(context.Context, *mongo.Client) error) error {
	for _, host := range mtest.ClusterConnString().Hosts {
		if err := runCommandOnHost(ctx, host, commandFn); err != nil {
			return fmt.Errorf("error executing callback against host %q: %w", host, err)
		}
	}
	return nil
}
