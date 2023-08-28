// Copyright (C) MongoDB, Inc. 2021-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package operation

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/internal/logger"
	"go.mongodb.org/mongo-driver/mongo/description"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
	"go.mongodb.org/mongo-driver/x/mongo/driver/session"
)

// Command is used to run a generic operation.
type Command struct {
	// Command is the command document to send to the database.
	Command bsoncore.Document

	// Database is the database to run this operation against.
	Database string

	// Deployment is the deployment to use for this operation.
	Deployment driver.Deployment

	// Selector is the server selector used to retrieve a server.
	Selector description.ServerSelector

	// ReadPreference is the read preference used with this operation.
	ReadPreference *readpref.ReadPref

	// Clock is the cluster clock for this operation.
	Clock *session.ClusterClock

	// Session is the session for this operation.
	Session *session.Client

	// Monitor is the monitor to use for APM events.
	Monitor *event.CommandMonitor

	// Crypt is the Crypt object to use for automatic encryption and decryption.
	Crypt driver.Crypt

	// ServerAPI is the server API version for this operation.
	ServerAPI *driver.ServerAPIOptions

	// Timeout is the timeout for this operation.
	Timeout *time.Duration

	// Logger is the logger for this operation.
	Logger *logger.Logger

	// CreateCursor controls whether or not executing the command creates a
	// cursor from the database response. It must be set to true to run commands
	// that return a cursor.
	CreateCursor bool

	// CursorOpts are the options to use when creating the cursor from the
	// database response.
	CursorOpts driver.CursorOptions

	resultResponse bsoncore.Document
	resultCursor   *driver.BatchCursor
}

// Result returns the result of executing this operation.
func (c *Command) Result() bsoncore.Document { return c.resultResponse }

// ResultCursor returns the BatchCursor that was constructed using the command response. If the operation was not
// configured to create a cursor (i.e. it was created using NewCommand rather than NewCursorCommand), this function
// will return nil and an error.
func (c *Command) ResultCursor() (*driver.BatchCursor, error) {
	if !c.CreateCursor {
		return nil, errors.New("command operation was not configured to create a cursor, but a result cursor was requested")
	}
	return c.resultCursor, nil
}

// Execute runs this operations and returns an error if the operation did not execute successfully.
func (c *Command) Execute(ctx context.Context) error {
	if c.Deployment == nil {
		return errors.New("the Command operation must have a Deployment set before Execute can be called")
	}

	// TODO(GODRIVER-2649): Actually pass readConcern to underlying driver.Operation.
	return driver.Operation{
		CommandFn: func(dst []byte, desc description.SelectedServer) ([]byte, error) {
			return append(dst, c.Command[4:len(c.Command)-1]...), nil
		},
		ProcessResponseFn: func(info driver.ResponseInfo) error {
			c.resultResponse = info.ServerResponse

			if c.CreateCursor {
				cursorRes, err := driver.NewCursorResponse(info)
				if err != nil {
					return err
				}

				c.resultCursor, err = driver.NewBatchCursor(cursorRes, c.Session, c.Clock, c.CursorOpts)
				return err
			}

			return nil
		},
		// TODO(GODRIVER-2649): Pass read concern to the operation.
		Client:         c.Session,
		Clock:          c.Clock,
		CommandMonitor: c.Monitor,
		Database:       c.Database,
		Deployment:     c.Deployment,
		ReadPreference: c.ReadPreference,
		Selector:       c.Selector,
		Crypt:          c.Crypt,
		ServerAPI:      c.ServerAPI,
		Timeout:        c.Timeout,
		Logger:         c.Logger,
	}.Execute(ctx)
}
