package mnet

import (
	"context"
	"io"
)

// WireMessageReader represents a Connection where server operations can be
// read from.
type WireMessageReader interface {
	Read(ctx context.Context) ([]byte, error)
}

// WireMessageWriter represents a Connection where server operations can be
// written to.
type WireMessageWriter interface {
	Write(ctx context.Context, wm []byte) error
}

// WireMessageReadWriteCloser represents a Connection where server operations
// can read from, written to, and closed.
type WireMessageReadWriteCloser interface {
	WireMessageReader
	WireMessageWriter
	io.Closer
}

type defaultIO struct{}

// Assert defaultIO implements WireMessageReadWriteCloser at compile-time.
var _ WireMessageReadWriteCloser = &defaultIO{}

// TODO: Add Logic
func (*defaultIO) Read(context.Context) ([]byte, error) { return nil, nil }
func (*defaultIO) Write(context.Context, []byte) error  { return nil }
func (*defaultIO) Close() error                         { return nil }
