package driver

import (
	"context"
	"errors"
	"strings"

	"github.com/mongodb/mongo-go-driver/x/bsonx/bsoncore"
)

// ListCollectionsBatchCursor is a special batch cursor returned from ListCollections that properly
// handles current and legacy ListCollections operations.
type ListCollectionsBatchCursor struct {
	legacy       bool
	bc           *BatchCursor
	currentBatch []byte
	err          error
}

// NewListCollectionsBatchCursor creates a new non-legacy ListCollectionsCursor.
func NewListCollectionsBatchCursor(bc *BatchCursor) (*ListCollectionsBatchCursor, error) {
	if bc == nil {
		return nil, errors.New("batch cursor must not be nil")
	}
	return &ListCollectionsBatchCursor{bc: bc}, nil
}

// NewLegacyListCollectionsBatchCursor creates a new legacy ListCollectionsCursor.
func NewLegacyListCollectionsBatchCursor(bc *BatchCursor) (*ListCollectionsBatchCursor, error) {
	if bc == nil {
		return nil, errors.New("batch cursor must not be nil")
	}
	return &ListCollectionsBatchCursor{legacy: true, bc: bc}, nil
}

// ID returns the cursor ID for this batch cursor.
func (lcbc *ListCollectionsBatchCursor) ID() int64 {
	return lcbc.bc.ID()
}

// Next indicates if there is another batch available. Returning false does not necessarily indicate
// that the cursor is closed. This method will return false when an empty batch is returned.
//
// If Next returns true, there is a valid batch of documents available. If Next returns false, there
// is not a valid batch of documents available.
func (lcbc *ListCollectionsBatchCursor) Next(ctx context.Context) bool {
	if !lcbc.bc.Next(ctx) {
		return false
	}

	if !lcbc.legacy {
		lcbc.currentBatch = lcbc.bc.currentBatch
		return true
	}

	batch := lcbc.bc.currentBatch
	lcbc.currentBatch = lcbc.currentBatch[:0]
	var doc bsoncore.Document
	var ok bool
	for {
		doc, batch, ok = bsoncore.ReadDocument(batch)
		if !ok {
			break
		}

		doc, lcbc.err = lcbc.projectNameElement(doc)
		if lcbc.err != nil {
			return false
		}
		lcbc.currentBatch = append(lcbc.currentBatch, doc...)
	}

	return true
}

// Batch will append the current batch of documents to dst. RequiredBytes can be called to determine
// the length of the current batch of documents.
//
// If there is no batch available, this method does nothing.
func (lcbc *ListCollectionsBatchCursor) Batch(dst []byte) []byte {
	return append(dst, lcbc.currentBatch...)
}

// RequiredBytes returns the number of bytes required for the current batch.
func (lcbc *ListCollectionsBatchCursor) RequiredBytes() int { return len(lcbc.currentBatch) }

// Err returns the latest error encountered.
func (lcbc *ListCollectionsBatchCursor) Err() error {
	if lcbc.err != nil {
		return lcbc.err
	}
	return lcbc.bc.Err()
}

// Close closes this batch cursor.
func (lcbc *ListCollectionsBatchCursor) Close(ctx context.Context) error { return lcbc.bc.Close(ctx) }

// project out the database name for a legacy server
func (*ListCollectionsBatchCursor) projectNameElement(rawDoc bsoncore.Document) (bsoncore.Document, error) {
	elems, err := rawDoc.Elements()
	if err != nil {
		return nil, err
	}

	var filteredElems []byte
	for _, elem := range elems {
		key := elem.Key()
		if key != "name" {
			filteredElems = append(filteredElems, elem...)
			continue
		}

		name := elem.Value().StringValue()
		collName := name[strings.Index(name, ".")+1:]
		filteredElems = bsoncore.AppendStringElement(filteredElems, "name", collName)
	}

	var filteredDoc []byte
	filteredDoc = bsoncore.BuildDocument(filteredDoc, filteredElems)
	return filteredDoc, nil
}
