package core

import "gopkg.in/mgo.v2/bson"

// The result of a find command
type FindResult struct {
	// The cursor
	Cursor FirstBatchCursorResult `bson:"cursor"`
}

// The first batch of a cursor
type FirstBatchCursorResult struct {
	// The first batch of the cursor
	FirstBatch []bson.Raw `bson:"firstBatch"`
	// The namespace to use for iterating the cursor
	NS         string     `bson:"ns"`
	// The cursor id
	ID         int64      `bson:"id"`
}

