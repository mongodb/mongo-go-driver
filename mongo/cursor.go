// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import "github.com/10gen/mongo-go-driver/mongo/private/ops"

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
