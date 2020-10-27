// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	errorInterrupted int32 = 11601
)

// TerminateOpenSessions executes a killAllSessions command to ensure that sesssions left open on the server by a test
// do not cause future tests to hang.
func TerminateOpenSessions(ctx context.Context) error {
	if mtest.CompareServerVersions(mtest.ServerVersion(), "3.6") < 0 {
		return nil
	}

	commandFn := func(ctx context.Context, client *mongo.Client) error {
		cmd := bson.D{
			{"killAllSessions", bson.A{}},
		}

		err := client.Database("admin").RunCommand(ctx, cmd).Err()
		if ce, ok := err.(mongo.CommandError); ok && ce.Code == errorInterrupted {
			// Workaround for SERVER-38335 on server versions below 4.2.
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

// PerformDistinctWorkaround executes a non-transactional "distinct" command against each mongos in a sharded cluster.
func PerformDistinctWorkaround(ctx context.Context) error {
	commandFn := func(ctx context.Context, client *mongo.Client) error {
		for _, coll := range Entities(ctx).Collections() {
			newColl := client.Database(coll.Database().Name()).Collection(coll.Name())
			_, err := newColl.Distinct(ctx, "x", bson.D{})
			if err != nil {
				ns := fmt.Sprintf("%s.%s", coll.Database().Name(), coll.Name())
				return fmt.Errorf("error running distinct for collection %q: %v", ns, err)
			}
		}

		return nil
	}

	return runAgainstAllMongoses(ctx, commandFn)
}

func RunCommandOnHost(ctx context.Context, host string, commandFn func(context.Context, *mongo.Client) error) error {
	clientOpts := options.Client().
		ApplyURI(mtest.ClusterURI()).
		SetHosts([]string{host})

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("error creating client to host %q: %v", host, err)
	}
	defer client.Disconnect(ctx)

	return commandFn(ctx, client)
}

func runAgainstAllMongoses(ctx context.Context, commandFn func(context.Context, *mongo.Client) error) error {
	for _, host := range mtest.ClusterConnString().Hosts {
		if err := RunCommandOnHost(ctx, host, commandFn); err != nil {
			return fmt.Errorf("error executing callback against host %q: %v", host, err)
		}
	}
	return nil
}
