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
	"go.mongodb.org/mongo-driver/event"
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
	if len(expectedEvents.Events) == 0 {
		if len(started) != 0 {
			return fmt.Errorf("expected no events but got started events %v", stringifyStartedEvents(started))
		}
		if len(succeeded) != 0 {
			return fmt.Errorf("expected no events but got succeeded events %v", stringifySucceededEvents(succeeded))
		}
		if len(failed) != 0 {
			return fmt.Errorf("expected no events but got failed events %v", stringifyFailedEvents(failed))
		}
		return nil
	}

	for idx, evt := range expectedEvents.Events {
		switch {
		case evt.CommandStartedEvent != nil:
			if len(started) == 0 {
				return newEventVerificationError(idx, "no CommandStartedEvent published")
			}

			actual := started[0]
			started = started[1:]

			expected := evt.CommandStartedEvent
			if expected.CommandName != nil && *expected.CommandName != actual.CommandName {
				return newEventVerificationError(idx, "expected command name %q, got %q", *expected.CommandName,
					actual.CommandName)
			}
			if expected.DatabaseName != nil && *expected.DatabaseName != actual.DatabaseName {
				return newEventVerificationError(idx, "expected database name %q, got %q", *expected.DatabaseName,
					actual.DatabaseName)
			}
			if expected.Command != nil {
				expectedDoc := DocumentToRawValue(expected.Command)
				actualDoc := DocumentToRawValue(actual.Command)
				if err := VerifyValuesMatch(ctx, expectedDoc, actualDoc, true); err != nil {
					return newEventVerificationError(idx, "error comparing command documents: %v", err)
				}
			}
		case evt.CommandSucceededEvent != nil:
			if len(succeeded) == 0 {
				return newEventVerificationError(idx, "no CommandSucceededEvent published")
			}

			actual := succeeded[0]
			succeeded = succeeded[1:]

			expected := evt.CommandSucceededEvent
			if expected.CommandName != nil && *expected.CommandName != actual.CommandName {
				return newEventVerificationError(idx, "expected command name %q, got %q", *expected.CommandName,
					actual.CommandName)
			}
			if expected.Reply != nil {
				expectedDoc := DocumentToRawValue(expected.Reply)
				actualDoc := DocumentToRawValue(actual.Reply)
				if err := VerifyValuesMatch(ctx, expectedDoc, actualDoc, true); err != nil {
					return newEventVerificationError(idx, "error comparing reply documents: %v", err)
				}
			}
		case evt.CommandFailedEvent != nil:
			if len(failed) == 0 {
				return newEventVerificationError(idx, "no CommandFailedEvent published")
			}

			actual := failed[0]
			failed = failed[1:]

			expected := evt.CommandFailedEvent
			if expected.CommandName != nil && *expected.CommandName != actual.CommandName {
				return newEventVerificationError(idx, "expected command name %q, got %q", *expected.CommandName,
					actual.CommandName)
			}
		default:
			return newEventVerificationError(idx, "no expected event set on CommandMonitoringEvent instance")
		}
	}

	// Verify that there are no remaining events.
	if len(started) > 0 {
		return fmt.Errorf("extra started events published: %v", stringifyStartedEvents(started))
	}
	if len(succeeded) > 0 {
		return fmt.Errorf("extra succeeded events published: %v", stringifySucceededEvents(succeeded))
	}
	if len(failed) > 0 {
		return fmt.Errorf("extra failed events published: %v", stringifyFailedEvents(failed))
	}

	return nil
}

func newEventVerificationError(idx int, msg string, args ...interface{}) error {
	fullMsg := fmt.Sprintf(msg, args...)
	return fmt.Errorf("event comparison failed at index %d: %s", idx, fullMsg)
}

func stringifyStartedEvents(events []*event.CommandStartedEvent) []string {
	converted := make([]string, 0, len(events))
	for _, evt := range events {
		converted = append(converted, fmt.Sprintf("%s", evt.Command))
	}

	return converted
}

func stringifySucceededEvents(events []*event.CommandSucceededEvent) []string {
	converted := make([]string, 0, len(events))
	for _, evt := range events {
		converted = append(converted, fmt.Sprintf("command name: %s, reply: %s", evt.CommandName, evt.Reply))
	}

	return converted
}

func stringifyFailedEvents(events []*event.CommandFailedEvent) []string {
	converted := make([]string, 0, len(events))
	for _, evt := range events {
		converted = append(converted, fmt.Sprintf("command name: %s, failure: %s", evt.CommandName, evt.Failure))
	}

	return converted
}
