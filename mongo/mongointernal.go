// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build mongointernal

package mongo

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/session"
)

// NewSessionWithID returns a Session with the given sessionID document. The
// sessionID is a BSON document with key "id" containing a 16-byte UUID (binary
// subtype 4).
//
// Sessions returned by NewSessionWithID are never added to the driver's session
// pool. Calling EndSession on a Session returned by NewSessionWithID will
// panic.
//
// NewSessionWithID is intended only for internal use and may be changed or
// removed at any time.
func NewSessionWithID(client *Client, sessionID bson.Raw) *Session {
	return &Session{
		clientSession: &session.Client{
			Server: &session.Server{
				SessionID: bsoncore.Document(sessionID),
				LastUsed:  time.Now(),
			},
			ClientID: client.id,
		},
		client:     client,
		deployment: client.deployment,
	}
}
