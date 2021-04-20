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
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
)

var (
	connectionClosedReasons = map[string]string{
		"stale":      event.ReasonStale,
		"idle":       event.ReasonIdle,
		"error":      event.ReasonConnectionErrored,
		"poolClosed": event.ReasonPoolClosed,
	}

	checkOutFailedReasons = map[string]string{
		"poolClosed":      event.ReasonPoolClosed,
		"timeout":         event.ReasonTimedOut,
		"connectionError": event.ReasonConnectionErrored,
	}
)

type commandMonitoringEvent struct {
	CommandStartedEvent *struct {
		Command      bson.Raw `bson:"command"`
		CommandName  *string  `bson:"commandName"`
		DatabaseName *string  `bson:"databaseName"`
		HasServiceID *bool    `bson:"hasServiceId"`
	} `bson:"commandStartedEvent"`

	CommandSucceededEvent *struct {
		CommandName  *string  `bson:"commandName"`
		Reply        bson.Raw `bson:"reply"`
		HasServiceID *bool    `bson:"hasServiceId"`
	} `bson:"commandSucceededEvent"`

	CommandFailedEvent *struct {
		CommandName  *string `bson:"commandName"`
		HasServiceID *bool   `bson:"hasServiceId"`
	} `bson:"commandFailedEvent"`
}

type cmapEvent struct {
	ConnectionCreatedEvent *struct{} `bson:"connectionCreatedEvent"`

	ConnectionReadyEvent *struct{} `bson:"connectionReadyEvent"`

	ConnectionClosedEvent *struct {
		Reason *string `bson:"reason"`
	} `bson:"connectionClosedEvent"`

	ConnectionCheckedOutEvent *struct{} `bson:"connectionCheckedOutEvent"`

	ConnectionCheckOutFailedEvent *struct {
		Reason *string `bson:"reason"`
	} `bson:"connectionCheckOutFailedEvent"`

	ConnectionCheckedInEvent *struct{} `bson:"connectionCheckedInEvent"`

	PoolClearedEvent *struct {
		HasServiceID *bool `bson:"hasServiceId"`
	} `bson:"poolClearedEvent"`
}

type expectedEvents struct {
	ClientID      string `bson:"client"`
	CommandEvents []commandMonitoringEvent
	CMAPEvents    []cmapEvent
}

var _ bson.Unmarshaler = (*expectedEvents)(nil)

func (e *expectedEvents) UnmarshalBSON(data []byte) error {
	// The data to be unmarshalled looks like {client: <client ID>, eventType: <string>, events: [event0, event1, ...]}.
	// We use the "eventType" value to determine which struct field should be used to deserialize the "events" array.

	var temp struct {
		ClientID  string                 `bson:"client"`
		EventType string                 `bson:"eventType"`
		Events    bson.RawValue          `bson:"events"`
		Extra     map[string]interface{} `bson:",inline"`
	}
	if err := bson.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("error unmarshalling to temporary expectedEvents object: %v", err)
	}
	if len(temp.Extra) > 0 {
		return fmt.Errorf("unrecognized fields for expectedEvents: %v", temp.Extra)
	}

	e.ClientID = temp.ClientID
	if temp.Events.Type != bsontype.Array {
		return fmt.Errorf("expected 'events' to be an array but got a %q", temp.Events.Type)
	}

	var target interface{}
	switch temp.EventType {
	case "command", "":
		target = &e.CommandEvents
	case "cmap":
		target = &e.CMAPEvents
	default:
		return fmt.Errorf("unrecognized 'eventType' value for expectedEvents: %q", temp.EventType)
	}

	if err := temp.Events.Unmarshal(target); err != nil {
		return fmt.Errorf("error unmarshalling events array: %v", err)
	}
	return nil
}

func verifyEvents(ctx context.Context, expectedEvents *expectedEvents) error {
	client, err := entities(ctx).client(expectedEvents.ClientID)
	if err != nil {
		return err
	}

	switch {
	case expectedEvents.CommandEvents != nil:
		return verifyCommandEvents(ctx, client, expectedEvents.CommandEvents)
	case expectedEvents.CMAPEvents != nil:
		return verifyCMAPEvents(client, expectedEvents.CMAPEvents)
	}
	return nil
}

func verifyCommandEvents(ctx context.Context, client *clientEntity, expectedEvents []commandMonitoringEvent) error {
	started := client.startedEvents()
	succeeded := client.succeededEvents()
	failed := client.failedEvents()

	// If the Events array is nil, verify that no events were sent.
	if len(expectedEvents) == 0 && (len(started)+len(succeeded)+len(failed) != 0) {
		return fmt.Errorf("expected no events to be sent but got %s", stringifyEventsForClient(client))
	}

	for idx, evt := range expectedEvents {
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
				expectedDoc := documentToRawValue(expected.Command)
				actualDoc := documentToRawValue(actual.Command)
				if err := verifyValuesMatch(ctx, expectedDoc, actualDoc, true); err != nil {
					return newEventVerificationError(idx, client, "error comparing command documents: %v", err)
				}
			}
			if expected.HasServiceID != nil {
				if err := verifyServiceID(*expected.HasServiceID, actual.ServiceID); err != nil {
					return newEventVerificationError(idx, client, "error verifying serviceID: %v", err)
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
				expectedDoc := documentToRawValue(expected.Reply)
				actualDoc := documentToRawValue(actual.Reply)
				if err := verifyValuesMatch(ctx, expectedDoc, actualDoc, true); err != nil {
					return newEventVerificationError(idx, client, "error comparing reply documents: %v", err)
				}
			}
			if expected.HasServiceID != nil {
				if err := verifyServiceID(*expected.HasServiceID, actual.ServiceID); err != nil {
					return newEventVerificationError(idx, client, "error verifying serviceID: %v", err)
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
			if expected.HasServiceID != nil {
				if err := verifyServiceID(*expected.HasServiceID, actual.ServiceID); err != nil {
					return newEventVerificationError(idx, client, "error verifying serviceID: %v", err)
				}
			}
		default:
			return newEventVerificationError(idx, client, "no expected event set on commandMonitoringEvent instance")
		}
	}

	// Verify that there are no remaining events.
	if len(started) > 0 || len(succeeded) > 0 || len(failed) > 0 {
		return fmt.Errorf("extra events published; all events for client: %s", stringifyEventsForClient(client))
	}
	return nil
}

func verifyCMAPEvents(client *clientEntity, expected []cmapEvent) error {
	pooled := client.poolEvents()
	if len(expected) == 0 && len(pooled) != 0 {
		return fmt.Errorf("expected no cmap events to be sent but got %s", stringifyEventsForClient(client))
	}

	for idx, evt := range expected {
		var err error

		switch {
		case evt.ConnectionCreatedEvent != nil:
			if _, pooled, err = getNextPoolEvent(pooled, event.ConnectionCreated); err != nil {
				return newEventVerificationError(idx, client, err.Error())
			}
		case evt.ConnectionReadyEvent != nil:
			if _, pooled, err = getNextPoolEvent(pooled, event.ConnectionReady); err != nil {
				return newEventVerificationError(idx, client, err.Error())
			}
		case evt.ConnectionClosedEvent != nil:
			var actual *event.PoolEvent
			if actual, pooled, err = getNextPoolEvent(pooled, event.ConnectionClosed); err != nil {
				return newEventVerificationError(idx, client, err.Error())
			}

			if reason := evt.ConnectionClosedEvent.Reason; reason != nil {
				expectedReason, ok := connectionClosedReasons[*reason]
				if !ok {
					return newEventVerificationError(idx, client, "unrecognized reason %q", *reason)
				}
				if expectedReason != actual.Reason {
					return newEventVerificationError(idx, client, "expected reason %q, got %q", expectedReason, actual.Reason)
				}
			}
		case evt.ConnectionCheckedOutEvent != nil:
			if _, pooled, err = getNextPoolEvent(pooled, event.GetSucceeded); err != nil {
				return newEventVerificationError(idx, client, err.Error())
			}
		case evt.ConnectionCheckOutFailedEvent != nil:
			var actual *event.PoolEvent
			if actual, pooled, err = getNextPoolEvent(pooled, event.GetFailed); err != nil {
				return newEventVerificationError(idx, client, err.Error())
			}

			if reason := evt.ConnectionCheckOutFailedEvent.Reason; reason != nil {
				expectedReason, ok := checkOutFailedReasons[*reason]
				if !ok {
					return newEventVerificationError(idx, client, "unrecognized reason %q", *reason)
				}
				if expectedReason != actual.Reason {
					return newEventVerificationError(idx, client, "expected reason %q, got %q", expectedReason, actual.Reason)
				}
			}
		case evt.ConnectionCheckedInEvent != nil:
			if _, pooled, err = getNextPoolEvent(pooled, event.ConnectionReturned); err != nil {
				return newEventVerificationError(idx, client, err.Error())
			}
		case evt.PoolClearedEvent != nil:
			var actual *event.PoolEvent
			if actual, pooled, err = getNextPoolEvent(pooled, event.PoolCleared); err != nil {
				return newEventVerificationError(idx, client, err.Error())
			}
			if expectServiceID := evt.PoolClearedEvent.HasServiceID; expectServiceID != nil {
				if err := verifyServiceID(*expectServiceID, actual.ServiceID); err != nil {
					return newEventVerificationError(idx, client, "error verifying serviceID: %v", err)
				}
			}
		default:
			return newEventVerificationError(idx, client, "no expected event set on cmapEvent instance")
		}
	}

	// Verify that there are no remaining events.
	if len(pooled) > 0 {
		return fmt.Errorf("extra events published; all events for client: %s", stringifyEventsForClient(client))
	}
	return nil
}

func getNextPoolEvent(events []*event.PoolEvent, expectedType string) (*event.PoolEvent, []*event.PoolEvent, error) {
	if len(events) == 0 {
		return nil, nil, fmt.Errorf("no %q event published", expectedType)
	}

	evt := events[0]
	if evt.Type != expectedType {
		return nil, nil, fmt.Errorf("expected pool event of type %q, got %q", expectedType, evt.Type)
	}
	return evt, events[1:], nil
}

func verifyServiceID(expectServiceID bool, serviceID *primitive.ObjectID) error {
	if eventHasID := serviceID != nil; expectServiceID != eventHasID {
		return fmt.Errorf("expected event to have server ID: %v, event has server ID %v", expectServiceID, serviceID)
	}
	return nil
}

func newEventVerificationError(idx int, client *clientEntity, msg string, args ...interface{}) error {
	fullMsg := fmt.Sprintf(msg, args...)
	return fmt.Errorf("event comparison failed at index %d: %s; all events found for client: %s", idx, fullMsg,
		stringifyEventsForClient(client))
}

func stringifyEventsForClient(client *clientEntity) string {
	str := bytes.NewBuffer(nil)

	str.WriteString("\n\nStarted Events\n\n")
	for _, evt := range client.startedEvents() {
		str.WriteString(fmt.Sprintf("[%s] %s\n", evt.ConnectionID, evt.Command))
	}

	str.WriteString("\nSucceeded Events\n\n")
	for _, evt := range client.succeededEvents() {
		str.WriteString(fmt.Sprintf("[%s] CommandName: %s, Reply: %s\n", evt.ConnectionID, evt.CommandName, evt.Reply))
	}

	str.WriteString("\nFailed Events\n\n")
	for _, evt := range client.failedEvents() {
		str.WriteString(fmt.Sprintf("[%s] CommandName: %s, Failure: %s\n", evt.ConnectionID, evt.CommandName, evt.Failure))
	}

	str.WriteString("\nPool Events\n\n")
	for _, evt := range client.poolEvents() {
		str.WriteString(fmt.Sprintf("[%s] Event Type: %q\n", evt.Address, evt.Type))
	}

	return str.String()
}
