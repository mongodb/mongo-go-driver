package driverx

import "go.mongodb.org/mongo-driver/x/bsonx/bsoncore"

// Batch contains the necessary information to batch split an operation. This is only used for write
// oeprations.
type Batches struct {
	Identifier string
	Documents  []bsoncore.Document
	Current    []bsoncore.Document
	Ordered    *bool
}

// Valid returns true if Batches contains both an identifier and the length of Documents is greater
// than zero.
func (b *Batches) Valid() bool { return b != nil && b.Identifier != "" && len(b.Documents) > 0 }

// ClearBatch clears the Current batch. This must be called before AdvanceBatch will advance to the
// next batch.
func (b *Batches) ClearBatch() { b.Current = b.Current[:0] }

// Next splits the next batch using maxCount and targetBbatchSize. This method will do nothing if
// the current batch has not been cleared. We do this so that when this is called during execute we
// can call it without first needing to check if we already have a batch, which makes the code
// simpler and makes retrying easier.
func (b *Batches) AdvanceBatch(maxCount, targetBatchSize int) error {
	if len(b.Current) > 0 {
		return nil
	}
	if targetBatchSize > reservedCommandBufferBytes {
		targetBatchSize -= reservedCommandBufferBytes
	}

	if maxCount <= 0 {
		maxCount = 1
	}

	splitAfter := 0
	size := 1
	for _, doc := range b.Documents {
		if len(doc) > targetBatchSize {
			return ErrDocumentTooLarge
		}
		if size+len(doc) > targetBatchSize {
			break
		}

		size += len(doc)
		splitAfter += 1
	}

	b.Current, b.Documents = b.Documents[:splitAfter], b.Documents[splitAfter:]
	return nil
}
