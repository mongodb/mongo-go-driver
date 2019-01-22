package mongo

import (
	"context"
)

// batchCursor is the interface implemented by types that can provide batches of document results.
// The Cursor type is built on top of this type.
type batchCursor interface {
	// ID returns the ID of the cursor.
	ID() int64

	// Next returns true if there is a batch available.
	Next(context.Context) bool

	// Batch appends the current batch of documents to dst. RequiredBytes can be used to determine
	// the length of the current batch of documents.
	//
	// If there is no batch available, this method should do nothing.
	Batch(dst []byte) []byte

	// RequiredBytes returns the number of bytes required fo rthe current batch.
	RequiredBytes() int

	// Err returns the last error encountered.
	Err() error

	// Close closes the cursor.
	Close(context.Context) error
}
