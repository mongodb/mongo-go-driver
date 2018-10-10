// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// CursorOptions represents the possible options for cursors
type CursorOptions struct {
	BatchSize      *int32 // The number of documents to return per batch
	MaxAwaitTimeMS *int64 // The maximum amount of time for the server to wait on new documents to satisfy a tailable cursor query
}

func Cursor() *CursorOptions {
	return &CursorOptions{}
}

// SetBatchSize specifies the number of documents to return per batch
func (co *CursorOptions) SetBatchSize(i int32) *CursorOptions {
	co.BatchSize = &i
	return co
}

// SetMaxAwaitTimeMS specifies the maximum amount of time for the server to
// wait on new documents to satisfy a tailable cursor query
func (co *CursorOptions) SetMaxAwaitTimeMS(i int64) *CursorOptions {
	co.MaxAwaitTimeMS = &i
	return co
}

// ToCursorOptions combines the argued CursorOptions into a single CursorOptions in a last-one-wins fashion
func ToCursorOptions(opts ...*CursorOptions) *CursorOptions {
	cursorOpts := Cursor()
	for _, co := range opts {
		if co.BatchSize != nil {
			cursorOpts.BatchSize = co.BatchSize
		}
		if co.MaxAwaitTimeMS != nil {
			cursorOpts.MaxAwaitTimeMS = co.MaxAwaitTimeMS
		}
	}

	return cursorOpts
}
