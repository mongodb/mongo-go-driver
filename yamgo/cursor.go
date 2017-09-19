package yamgo

import "github.com/10gen/mongo-go-driver/yamgo/private/ops"

// Cursor instances iterate a stream of documents. Each document is
// decoded into the result according to the rules of the bson package.
//
// A typical usage of the Cursor interface would be:
//
//		cursor := ...    // get a cursor from some operation
//		ctx := ...       // create a context for the operation
//		var doc bson.D
//		for cursor.Next(ctx, &doc) {
//			...
//		}
//		err := cursor.Close(ctx)
type Cursor ops.Cursor
