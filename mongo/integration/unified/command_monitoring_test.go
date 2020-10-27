// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"bytes"
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

type CommandMonitoringEvent struct {
	CommandStartedEvent *struct {
		Command      bson.Raw `bson:"command"`
		CommandName  *string  `bson:"commandName"`
		DatabaseName *string  `bson:"databaseName"`
	} `bson:"commandStartedEvent"`

	CommandSucceededEvent *struct {
		CommandName *string  `bson:"commandName"`
		Reply       bson.Raw `bson:"reply"`
	} `bson:"commandSucceededEvent"`

	CommandFailedEvent *struct {
		CommandName *string `bson:"commandName"`
	} `bson:"commandFailedEvent"`
}

type ExpectedEvents struct {
	ClientID string                   `bson:"client"`
	Events   []CommandMonitoringEvent `bson:"events"`
}

func VerifyEvents(ctx context.Context, expectedEvents *ExpectedEvents) error {
	client, err := Entities(ctx).Client(expectedEvents.ClientID)
	if err != nil {
		return err
	}

	if expectedEvents.Events == nil {
		return nil
	}

	started := client.StartedEvents()
	succeeded := client.SucceededEvents()
	failed := client.FailedEvents()

	// If the Events array is nil, verify that no events were sent.
	if len(expectedEvents.Events) == 0 && (len(started)+len(succeeded)+len(failed) != 0) {
		return fmt.Errorf("expected no events to be sent but got %s", stringifyEventsForClient(client))
	}

	for idx, evt := range expectedEvents.Events {
		switch {
		case evt.CommandStartedEvent != nil:
			if len(started) == 0 {
				return newEventVerificationError(idx, client, "no CommandStartedEvent published")
			}

			actual := started[0]
			started = started[1:]

			expected := evt.CommandStartedEvent
			if expected.CommandName != nil && *expected.CommandName != actual.CommandName {
				return newEventVerificationError(idx, client, "expected command name %q, got %q", *expected.CommandName,
					actual.CommandName)
			}
			if expected.DatabaseName != nil && *expected.DatabaseName != actual.DatabaseName {
				return newEventVerificationError(idx, client, "expected database name %q, got %q", *expected.DatabaseName,
					actual.DatabaseName)
			}
			if expected.Command != nil {
				expectedDoc := DocumentToRawValue(expected.Command)
				actualDoc := DocumentToRawValue(actual.Command)
				if err := VerifyValuesMatch(ctx, expectedDoc, actualDoc, true); err != nil {
					return newEventVerificationError(idx, client, "error comparing command documents: %v", err)
				}
			}
		case evt.CommandSucceededEvent != nil:
			if len(succeeded) == 0 {
				return newEventVerificationError(idx, client, "no CommandSucceededEvent published")
			}

			actual := succeeded[0]
			succeeded = succeeded[1:]

			expected := evt.CommandSucceededEvent
			if expected.CommandName != nil && *expected.CommandName != actual.CommandName {
				return newEventVerificationError(idx, client, "expected command name %q, got %q", *expected.CommandName,
					actual.CommandName)
			}
			if expected.Reply != nil {
				expectedDoc := DocumentToRawValue(expected.Reply)
				actualDoc := DocumentToRawValue(actual.Reply)
				if err := VerifyValuesMatch(ctx, expectedDoc, actualDoc, true); err != nil {
					return newEventVerificationError(idx, client, "error comparing reply documents: %v", err)
				}
			}
		case evt.CommandFailedEvent != nil:
			if len(failed) == 0 {
				return newEventVerificationError(idx, client, "no CommandFailedEvent published")
			}

			actual := failed[0]
			failed = failed[1:]

			expected := evt.CommandFailedEvent
			if expected.CommandName != nil && *expected.CommandName != actual.CommandName {
				return newEventVerificationError(idx, client, "expected command name %q, got %q", *expected.CommandName,
					actual.CommandName)
			}
		default:
			return newEventVerificationError(idx, client, "no expected event set on CommandMonitoringEvent instance")
		}
	}

	// Verify that there are no remaining events.
	if len(started) > 0 || len(succeeded) > 0 || len(failed) > 0 {
		return fmt.Errorf("extra events published; all events for client: %s", stringifyEventsForClient(client))
	}
	return nil
}

func newEventVerificationError(idx int, client *ClientEntity, msg string, args ...interface{}) error {
	fullMsg := fmt.Sprintf(msg, args...)
	return fmt.Errorf("event comparison failed at index %d: %s; all events found for client: %s", idx, fullMsg,
		stringifyEventsForClient(client))
}

func stringifyEventsForClient(client *ClientEntity) string {
	str := bytes.NewBuffer(nil)

	str.WriteString("\n\nStarted Events\n\n")
	for _, evt := range client.StartedEvents() {
		str.WriteString(fmt.Sprintf("[%s] %s\n", evt.ConnectionID, evt.Command))
	}

	str.WriteString("\nSucceeded Events\n\n")
	for _, evt := range client.SucceededEvents() {
		str.WriteString(fmt.Sprintf("[%s] CommandName: %s, Reply: %s\n", evt.ConnectionID, evt.CommandName, evt.Reply))
	}

	str.WriteString("\nFailed Events\n\n")
	for _, evt := range client.FailedEvents() {
		str.WriteString(fmt.Sprintf("[%s] CommandName: %s, Failure: %s\n", evt.ConnectionID, evt.CommandName, evt.Failure))
	}

	return str.String()
}
