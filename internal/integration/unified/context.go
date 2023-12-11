// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// ctxKey is used to define keys for values stored in context.Context objects.
type ctxKey string

const (
	// operationalFailPoint indicates that the test case contains an
	// operation that sets a failpoint. This information is useful in
	// determining if a client entity (file or operational level) should
	// specify a reduced value for heartbeatFrequencyMS.
	operationalFailPointKey ctxKey = "operational-fail-point"
	// entitiesKey is used to store an entityMap instance in a Context.
	entitiesKey ctxKey = "test-entities"
	// failPointsKey is used to store a map from a fail point name to the Client instance used to configure it.
	failPointsKey ctxKey = "test-failpoints"
	// targetedFailPointsKey is used to store a map from a fail point name to the host on which the fail point is set.
	targetedFailPointsKey ctxKey = "test-targeted-failpoints"
	clientLogMessagesKey  ctxKey = "test-expected-log-message-count"
	ignoreLogMessagesKey  ctxKey = "test-ignore-log-message-count"
)

// newTestContext creates a new Context derived from ctx with values initialized to store the state required for test
// execution.
func newTestContext(
	ctx context.Context,
	entityMap *EntityMap,
	clientLogMessages []*clientLogMessages,
	hasOperationalFailPoint bool,
) context.Context {
	ctx = context.WithValue(ctx, operationalFailPointKey, hasOperationalFailPoint)
	ctx = context.WithValue(ctx, entitiesKey, entityMap)
	ctx = context.WithValue(ctx, failPointsKey, make(map[string]*mongo.Client))
	ctx = context.WithValue(ctx, targetedFailPointsKey, make(map[string]string))
	ctx = context.WithValue(ctx, clientLogMessagesKey, clientLogMessages)
	return ctx
}

func addFailPoint(ctx context.Context, failPoint string, client *mongo.Client) error {
	failPoints := ctx.Value(failPointsKey).(map[string]*mongo.Client)
	if _, ok := failPoints[failPoint]; ok {
		return fmt.Errorf("fail point %q already exists in tracked fail points map", failPoint)
	}

	failPoints[failPoint] = client
	return nil
}

func addTargetedFailPoint(ctx context.Context, failPoint string, host string) error {
	failPoints := ctx.Value(targetedFailPointsKey).(map[string]string)
	if _, ok := failPoints[failPoint]; ok {
		return fmt.Errorf("fail point %q already exists in tracked targeted fail points map", failPoint)
	}

	failPoints[failPoint] = host
	return nil
}

func failPoints(ctx context.Context) map[string]*mongo.Client {
	return ctx.Value(failPointsKey).(map[string]*mongo.Client)
}

func hasOperationalFailpoint(ctx context.Context) bool {
	return ctx.Value(operationalFailPointKey).(bool)
}

func targetedFailPoints(ctx context.Context) map[string]string {
	return ctx.Value(targetedFailPointsKey).(map[string]string)
}

func entities(ctx context.Context) *EntityMap {
	return ctx.Value(entitiesKey).(*EntityMap)
}

func expectedLogMessagesCount(ctx context.Context, clientID string) int {
	messages := ctx.Value(clientLogMessagesKey).([]*clientLogMessages)

	count := 0
	for _, message := range messages {
		if message.Client == clientID {
			count += len(message.LogMessages)
		}
	}

	return count
}

func ignoreLogMessages(ctx context.Context, clientID string) []*logMessage {
	messages := ctx.Value(clientLogMessagesKey).([]*clientLogMessages)

	ignoreMessages := []*logMessage{}
	for _, message := range messages {
		if message.Client == clientID {
			ignoreMessages = append(ignoreMessages, message.IgnoreMessages...)
		}
	}

	return ignoreMessages
}
