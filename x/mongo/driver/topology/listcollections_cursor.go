package topology

import (
	"context"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/x/bsonx/bsoncore"
	"github.com/mongodb/mongo-go-driver/x/network/command"
	"strings"
)

type listCollCursor struct {
	*cursor
}

// NewListCollectionsCursor creates a new command.Cursor. The command.Cursor passed in to be wrapped must be of type
// *cursor
func NewListCollectionsCursor(c command.Cursor) command.Cursor {
	return &listCollCursor{
		c.(*cursor),
	}
}

func (c *listCollCursor) ID() int64 {
	return c.cursor.ID()
}

func (c *listCollCursor) Next(ctx context.Context) bool {
	return c.cursor.Next(ctx)
}

func (c *listCollCursor) Decode(v interface{}) error {
	br, err := c.DecodeBytes()
	if err != nil {
		return err
	}

	return bson.UnmarshalWithRegistry(c.cursor.registry, br, v)
}

func (c *listCollCursor) DecodeBytes() (bson.Raw, error) {
	doc, err := c.cursor.DecodeBytes()
	if err != nil {
		return nil, err
	}

	return projectNameElement(doc)
}

func (c *listCollCursor) Err() error {
	return c.cursor.Err()
}

func (c *listCollCursor) Close(ctx context.Context) error {
	return c.cursor.Close(ctx)
}

// project out the database name for a legacy server
func projectNameElement(rawDoc bson.Raw) (bson.Raw, error) {
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
