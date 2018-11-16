// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"context"
	"errors"
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"github.com/mongodb/mongo-go-driver/internal"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"github.com/mongodb/mongo-go-driver/x/bsonx/bsoncore"
	"github.com/mongodb/mongo-go-driver/x/network/connection"
	"github.com/mongodb/mongo-go-driver/x/network/description"
	"github.com/mongodb/mongo-go-driver/x/network/wiremessage"
	"github.com/stretchr/testify/assert"
)

func TestCursorNextDoesNotPanicIfContextisNil(t *testing.T) {
	// all collection/cursor iterators should take contexts, but
	// permit passing nils for contexts, which should not
	// panic.
	//
	// While more through testing might be ideal this check
	// prevents a regression of GODRIVER-298

	c := cursor{
		batch: []bson.RawValue{
			{Type: bsontype.String, Value: bsoncore.AppendString(nil, "a")},
			{Type: bsontype.String, Value: bsoncore.AppendString(nil, "b")},
		},
	}

	var iterNext bool
	assert.NotPanics(t, func() {
		iterNext = c.Next(nil)
	})
	assert.True(t, iterNext)
}

func TestCursorLoopsUntilDocAvailable(t *testing.T) {
	// Next should loop until at least one doc is available
	// Here, the mock pool and connection implementations (below) write
	// empty batch responses a few times before returning a non-empty batch

	s := createDefaultConnectedServer(t, false)
	c := cursor{
		id:     1,
		batch:  []bson.RawValue{},
		server: s,
	}

	assert.True(t, c.Next(nil))
}

func TestCursorReturnsFalseOnContextCancellation(t *testing.T) {
	// Next should return false if an error occurs
	// here the error is the Context being cancelled

	s := createDefaultConnectedServer(t, false)
	c := cursor{
		id:     1,
		batch:  []bson.RawValue{},
		server: s,
	}

	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	assert.False(t, c.Next(ctx))
}

func TestCursorNextReturnsFalseIfErrorOccurred(t *testing.T) {
	// Next should return false if an error occurs
	// here the error is an invalid namespace (""."")

	s := createDefaultConnectedServer(t, true)
	c := cursor{
		id:     1,
		batch:  []bson.RawValue{},
		server: s,
	}
	assert.False(t, c.Next(nil))
}

func TestCursorNextReturnsFalseIfResIdZeroAndNoMoreDocs(t *testing.T) {
	// Next should return false if the cursor id is 0 and there are no documents in the next batch

	c := cursor{id: 0, batch: []bson.RawValue{}}
	assert.False(t, c.Next(nil))
}

func createDefaultConnectedServer(t *testing.T, willErr bool) *Server {
	s, err := ConnectServer(nil, "127.0.0.1")
	s.pool = &mockPool{t: t, willErr: willErr}
	if err != nil {
		assert.Fail(t, "Server creation failed")
	}
	desc := description.Server{
		WireVersion: &description.VersionRange{
			Max: 6,
		},
	}
	s.desc.Store(desc)

	return s
}

func createOKBatchReplyDoc(id int64, batchDocs bsonx.Arr) bsonx.Doc {
	return bsonx.Doc{
		{"ok", bsonx.Int32(1)},
		{
			"cursor",
			bsonx.Document(bsonx.Doc{
				{"id", bsonx.Int64(id)},
				{"nextBatch", bsonx.Array(batchDocs)},
			}),
		}}
}

// Mock Pool implementation
type mockPool struct {
	t       *testing.T
	willErr bool
	writes  int // the number of wire messages written so far
}

func (m *mockPool) Get(ctx context.Context) (connection.Connection, *description.Server, error) {
	m.writes++
	return &mockConnection{willErr: m.willErr, writes: m.writes}, nil, nil
}

func (*mockPool) Connect(ctx context.Context) error {
	return nil
}

func (*mockPool) Disconnect(ctx context.Context) error {
	return nil
}

func (*mockPool) Drain() error {
	return nil
}

// Mock Connection implementation that
type mockConnection struct {
	t       *testing.T
	willErr bool
	writes  int // the number of wire messages written so far
}

// this mock will not actually write anything
func (*mockConnection) WriteWireMessage(ctx context.Context, wm wiremessage.WireMessage) error {
	select {
	case <-ctx.Done():
		return errors.New("intentional mock error")
	default:
		return nil
	}
}

// mock a read by returning an empty cursor result until
func (m *mockConnection) ReadWireMessage(ctx context.Context) (wiremessage.WireMessage, error) {
	if m.writes < 4 {
		// write empty batch
		d := createOKBatchReplyDoc(2, bsonx.Arr{})

		return internal.MakeReply(m.t, d), nil
	} else if m.willErr {
		// write error
		return nil, errors.New("intentional mock error")
	} else {
		// write non-empty batch
		d := createOKBatchReplyDoc(2, bsonx.Arr{bsonx.String("a")})

		return internal.MakeReply(m.t, d), nil
	}
}

func (*mockConnection) Close() error {
	return nil
}

func (*mockConnection) Expired() bool {
	return false
}

func (*mockConnection) Alive() bool {
	return true
}

func (*mockConnection) ID() string {
	return ""
}
