package ops

import (
	"context"

	"github.com/10gen/mongo-go-driver/conn"
	"github.com/10gen/mongo-go-driver/msg"
	"gopkg.in/mgo.v2/bson"
)

// NewCursor creates a new cursor from the given cursor result.
func NewCursor(cursorResult CursorResult, batchSize int32, conn conn.Connection) (Cursor, error) {
	namespace := cursorResult.Namespace()
	if err := namespace.validate(); err != nil {
		return nil, err
	}

	return &cursorImpl{
		namespace:    cursorResult.Namespace(),
		batchSize:    batchSize,
		current:      0,
		currentBatch: cursorResult.InitialBatch(),
		cursorID:     cursorResult.CursorID(),
		conn:         conn,
	}, nil
}

// Cursor instances iterate a stream of documents. Each document is decoded into the result according to the rules of
// the bson package.  A typical usage of the Cursor interface would be:
//
//      cursor := ...    // get a cursor from some operation
//      var doc bson.D
//      for cursor.Next(&doc) {
//              fmt.Println(doc)
//      err := cursor.Close()
type Cursor interface {
	// Get the next result from the cursor.
	// Returns true if there were no errors and there is a next result.
	Next(context.Context, interface{}) bool

	// Returns the error status of the cursor
	Err() error

	// Close the cursor.  Ordinarily this is a no-op as the server closes the cursor when it is exhausted.
	// Returns the error status of this cursor so that clients do not have to call Err() separately
	Close(context.Context) error
}

type cursorImpl struct {
	namespace    Namespace
	batchSize    int32
	current      int
	currentBatch []bson.Raw
	cursorID     int64
	err          error
	conn         conn.Connection // TODO: missing abstraction.  Shouldn't require a connection here, but just a way to acquire and release one
}

func (c *cursorImpl) Next(ctx context.Context, result interface{}) bool {
	found := c.getNextFromCurrentBatch(result)
	if found {
		return true
	}
	if c.err != nil {
		return false
	}

	c.getMore(ctx)
	if c.err != nil {
		return false
	}

	return c.getNextFromCurrentBatch(result)
}

func (c *cursorImpl) Err() error {
	return c.err
}

func (c *cursorImpl) Close(ctx context.Context) error {
	c.currentBatch = nil

	if c.cursorID == 0 {
		return c.err
	}

	killCursorsCommand := struct {
		Collection string  `bson:"killCursors"`
		Cursors    []int64 `bson:"cursors"`
	}{
		Collection: c.namespace.Collection,
		Cursors:    []int64{c.cursorID},
	}

	killCursorsRequest := msg.NewCommand(
		msg.NextRequestID(),
		c.namespace.DB,
		false,
		killCursorsCommand,
	)

	err := conn.ExecuteCommand(ctx, c.conn, killCursorsRequest, &bson.D{})
	if err == nil {
		c.cursorID = 0
	} else if c.err == nil {
		c.err = err
	}

	return c.err
}

func (c *cursorImpl) getNextFromCurrentBatch(result interface{}) bool {
	if c.current < len(c.currentBatch) {
		err := bson.Unmarshal(c.currentBatch[c.current].Data, result)
		if err != nil {
			c.err = err
			return false
		}
		c.current++
		return true
	}
	return false
}

func (c *cursorImpl) getMore(ctx context.Context) {
	c.currentBatch = nil
	c.current = 0

	if c.cursorID == 0 {
		return
	}

	getMoreCommand := struct {
		CursorID   int64  `bson:"getMore"`
		Collection string `bson:"collection"`
		BatchSize  int32  `bson:"batchSize,omitempty"`
	}{
		CursorID:   c.cursorID,
		Collection: c.namespace.Collection,
	}
	if c.batchSize != 0 {
		getMoreCommand.BatchSize = c.batchSize
	}
	getMoreRequest := msg.NewCommand(
		msg.NextRequestID(),
		c.namespace.DB,
		false,
		getMoreCommand,
	)

	var response struct {
		OK     bool `bson:"ok"`
		Cursor struct {
			NextBatch []bson.Raw `bson:"nextBatch"`
			NS        string     `bson:"ns"`
			ID        int64      `bson:"id"`
		} `bson:"cursor"`
	}

	err := conn.ExecuteCommand(ctx, c.conn, getMoreRequest, &response)
	if err != nil {
		c.err = err
		return
	}

	c.cursorID = response.Cursor.ID
	c.currentBatch = response.Cursor.NextBatch
}
