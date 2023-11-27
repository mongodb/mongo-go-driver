// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mnet

import (
	"context"
	"io"

	"go.mongodb.org/mongo-driver/mongo/address"
	"go.mongodb.org/mongo-driver/mongo/description"
)

type WireMessageReader interface {
	Read(ctx context.Context) ([]byte, error)
}

type WireMessageWriter interface {
	Write(ctx context.Context, wm []byte) error
}

type WireMessageReadWriteCloser interface {
	WireMessageReader
	WireMessageWriter
	io.Closer
}

type Describer interface {
	Description() description.Server
	ID() string
	ServerConnectionID() *int64
	DriverConnectionID() int64
	Address() address.Address
	Stale() bool
}

type Streamer interface {
	SetStreaming(bool)
	CurrentlyStreaming() bool
	SupportsStreaming() bool
}

// Compressor is an interface used to compress wire messages. If a Connection
// supports compression it should implement this interface as well. The
// CompressWireMessage method will be called during the execution of an
// operation if the wire message is allowed to be compressed.
type Compressor interface {
	CompressWireMessage(src, dst []byte) ([]byte, error)
}

type Pinned interface {
	PinToCursor() error
	PinToTransaction() error
	UnpinFromCursor() error
	UnpinFromTransaction() error
}

type Connection struct {
	WireMessageReadWriteCloser
	Describer
	Streamer
	Compressor
	Pinned
}
