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
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
)

func executeTestRunnerOperation(ctx context.Context, operation *Operation) error {
	args := operation.Arguments

	switch operation.Name {
	case "failPoint":
		clientID := LookupString(args, "client")
		client, err := Entities(ctx).Client(clientID)
		if err != nil {
			return err
		}

		fpDoc := args.Lookup("failPoint").Document()
		if err := mtest.SetRawFailPoint(fpDoc, client.Client); err != nil {
			return err
		}
		return AddFailPoint(ctx, fpDoc.Index(0).Value().StringValue(), client.Client)
	case "targetedFailPoint":
		sessID := LookupString(args, "session")
		sess, err := Entities(ctx).Session(sessID)
		if err != nil {
			return err
		}

		clientSession := extractClientSession(sess)
		if clientSession.PinnedServer == nil {
			return fmt.Errorf("session is not pinned to a server")
		}

		targetHost := clientSession.PinnedServer.Addr.String()
		fpDoc := args.Lookup("failPoint").Document()
		commandFn := func(ctx context.Context, client *mongo.Client) error {
			return mtest.SetRawFailPoint(fpDoc, client)
		}

		if err := RunCommandOnHost(ctx, targetHost, commandFn); err != nil {
			return err
		}
		return AddTargetedFailPoint(ctx, fpDoc.Index(0).Value().StringValue(), targetHost)
	case "assertSessionTransactionState":
		sessID := LookupString(args, "session")
		sess, err := Entities(ctx).Session(sessID)
		if err != nil {
			return err
		}

		var expectedState session.TransactionState
		switch stateStr := LookupString(args, "state"); stateStr {
		case "none":
			expectedState = session.None
		case "starting":
			expectedState = session.Starting
		case "in_progress":
			expectedState = session.InProgress
		case "committed":
			expectedState = session.Committed
		case "aborted":
			expectedState = session.Aborted
		default:
			return fmt.Errorf("unrecognized session state type %q", stateStr)
		}

		if actualState := extractClientSession(sess).TransactionState; actualState != expectedState {
			return fmt.Errorf("expected session state %q does not match actual state %q", expectedState, actualState)
		}
		return nil
	case "assertSessionPinned":
		return verifySessionPinnedState(ctx, LookupString(args, "session"), true)
	case "assertSessionUnpinned":
		return verifySessionPinnedState(ctx, LookupString(args, "session"), false)
	case "assertSameLsidOnLastTwoCommands":
		return verifyLastTwoLsidsEqual(ctx, LookupString(args, "client"), true)
	case "assertDifferentLsidOnLastTwoCommands":
		return verifyLastTwoLsidsEqual(ctx, LookupString(args, "client"), false)
	case "assertSessionDirty":
		return verifySessionDirtyState(ctx, LookupString(args, "session"), true)
	case "assertSessionNotDirty":
		return verifySessionDirtyState(ctx, LookupString(args, "session"), false)
	case "assertCollectionExists":
		db := LookupString(args, "databaseName")
		coll := LookupString(args, "collectionName")
		return verifyCollectionExists(ctx, db, coll, true)
	case "assertCollectionNotExists":
		db := LookupString(args, "databaseName")
		coll := LookupString(args, "collectionName")
		return verifyCollectionExists(ctx, db, coll, false)
	case "assertIndexExists":
		db := LookupString(args, "databaseName")
		coll := LookupString(args, "collectionName")
		index := LookupString(args, "indexName")
		return verifyIndexExists(ctx, db, coll, index, true)
	case "assertIndexNotExists":
		db := LookupString(args, "databaseName")
		coll := LookupString(args, "collectionName")
		index := LookupString(args, "indexName")
		return verifyIndexExists(ctx, db, coll, index, false)
	default:
		return fmt.Errorf("unrecognized testRunner operation %q", operation.Name)
	}
}

func extractClientSession(sess mongo.Session) *session.Client {
	return sess.(mongo.XSession).ClientSession()
}

func verifySessionPinnedState(ctx context.Context, sessionID string, expectedPinned bool) error {
	sess, err := Entities(ctx).Session(sessionID)
	if err != nil {
		return err
	}

	if isPinned := extractClientSession(sess).PinnedServer != nil; expectedPinned != isPinned {
		return fmt.Errorf("session pinned state mismatch; expected to be pinned: %v, is pinned: %v", expectedPinned, isPinned)
	}
	return nil
}

func verifyLastTwoLsidsEqual(ctx context.Context, clientID string, expectedEqual bool) error {
	client, err := Entities(ctx).Client(clientID)
	if err != nil {
		return err
	}

	allEvents := client.StartedEvents()
	if len(allEvents) < 2 {
		return fmt.Errorf("client has recorded fewer than two command started events")
	}
	lastTwoEvents := allEvents[len(allEvents)-2:]

	firstID, err := lastTwoEvents[0].Command.LookupErr("lsid")
	if err != nil {
		return fmt.Errorf("first command has no 'lsid' field: %v", client.started[0].Command)
	}
	secondID, err := lastTwoEvents[1].Command.LookupErr("lsid")
	if err != nil {
		return fmt.Errorf("first command has no 'lsid' field: %v", client.started[1].Command)
	}

	areEqual := firstID.Equal(secondID)
	if expectedEqual && !areEqual {
		return fmt.Errorf("expected last two lsids to be equal, but got %s and %s", firstID, secondID)
	}
	if !expectedEqual && areEqual {
		return fmt.Errorf("expected last two lsids to be different but both were %s", firstID)
	}
	return nil
}

func verifySessionDirtyState(ctx context.Context, sessionID string, expectedDirty bool) error {
	sess, err := Entities(ctx).Session(sessionID)
	if err != nil {
		return err
	}

	if isDirty := extractClientSession(sess).Dirty; expectedDirty != isDirty {
		return fmt.Errorf("session dirty state mismatch; expected to be dirty: %v, is dirty: %v", expectedDirty, isDirty)
	}
	return nil
}

func verifyCollectionExists(ctx context.Context, dbName, collName string, expectedExists bool) error {
	db := mtest.GlobalClient().Database(dbName)
	collections, err := db.ListCollectionNames(ctx, bson.M{"name": collName})
	if err != nil {
		return fmt.Errorf("error running ListCollectionNames: %v", err)
	}

	if exists := len(collections) == 1; expectedExists != exists {
		ns := fmt.Sprintf("%s.%s", dbName, collName)
		return fmt.Errorf("collection existence mismatch; expected namespace %q to exist: %v, exists: %v", ns,
			expectedExists, exists)
	}
	return nil
}

func verifyIndexExists(ctx context.Context, dbName, collName, indexName string, expectedExists bool) error {
	iv := mtest.GlobalClient().Database(dbName).Collection(collName).Indexes()
	cursor, err := iv.List(ctx)
	if err != nil {
		return fmt.Errorf("error running IndexView.List: %v", err)
	}
	defer cursor.Close(ctx)

	var exists bool
	for cursor.Next(ctx) {
		if LookupString(cursor.Current, "name") == indexName {
			exists = true
			break
		}
	}
	if expectedExists != exists {
		ns := fmt.Sprintf("%s.%s", dbName, collName)
		return fmt.Errorf("index existence mismatch: expected index %q to exist in namespace %q: %v, exists: %v",
			indexName, ns, expectedExists, exists)
	}
	return nil
}
