// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"context"
)

// The cursor interface defines the methods that must be implemented by iterable types that can be stored in an
// EntityMap. This type exists so that we can store multiple concrete types returned by functions from the mongo package
// under a single interface (e.g. mongo.Cursor and mongo.ChangeStream).
type cursor interface {
	Close(context.Context) error
	Decode(interface{}) error
	Err() error
	Next(context.Context) bool
	TryNext(context.Context) bool
}
