package mongo

import (
	"context"

	"github.com/mongodb/mongo-go-driver/x/bsonx/bsoncore"
)

// batchCursor is the interface implemented by types that can provide batches of document results.
// The Cursor type is built on top of this type.
type batchCursor interface {
	// ID returns the ID of the cursor.
	ID() int64

	// Next returns true if there is a batch available.
	Next(context.Context) bool

	// Batch will return a DocumentSequence for the current batch of documents. The returned
	// DocumentSequence is only valid until the next call to Next or Close.
	Batch() *bsoncore.DocumentSequence

	// Err returns the last error encountered.
	Err() error

	// Close closes the cursor.
	Close(context.Context) error
}
