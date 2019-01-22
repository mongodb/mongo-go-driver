package mongo

import (
	"context"

	"github.com/mongodb/mongo-go-driver/x/bsonx/bsoncore"
)

// BatchCursor is the interface implemented by types that can provide batches of document results.
// The Cursor type is built on top of this type.
type BatchCursor interface {
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

	Close(context.Context) error
}

// PostBatchResumeTokener is an extension to the BatchCursor when used for change streams to enable
// retrieving PostBatchResumeTokens in 4.2+.
type PostBatchResumeTokener interface {
	// PostBatchResumeToken returns the PostBatchResumeToken from the latest cursor response. If
	// there was no PostBatchResumeToken on the cursor response, this method must return nil.
	PostBatchResumeToken() bsoncore.Document
}
