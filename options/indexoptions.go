// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import "time"

// CreateIndexesOptions represents all possible options for the create() function.
type CreateIndexesOptions struct {
	MaxTime *time.Duration // The maximum amount of time to allow the query to run.
}

// CreateIndexes creates a new CreateIndexesOptions instance.
func CreateIndexes() *CreateIndexesOptions {
	return &CreateIndexesOptions{}
}

// SetMaxTime specifies the maximum amount of time to allow the query to run.
func (c *CreateIndexesOptions) SetMaxTime(d time.Duration) *CreateIndexesOptions {
	c.MaxTime = &d
	return c
}

// MergeCreateIndexesOptions combines the given *CreateIndexesOptions into a single *CreateIndexesOptions in a last one
// wins fashion.
func MergeCreateIndexesOptions(opts ...*CreateIndexesOptions) *CreateIndexesOptions {
	c := CreateIndexes()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.MaxTime != nil {
			c.MaxTime = opt.MaxTime
		}
	}

	return c
}

// DropIndexesOptions represents all possible options for the create() function.
type DropIndexesOptions struct {
	MaxTime *time.Duration
}

// DropIndexes creates a new DropIndexesOptions instance.
func DropIndexes() *DropIndexesOptions {
	return &DropIndexesOptions{}
}

// SetMaxTime specifies the maximum amount of time to allow the query to run.
func (d *DropIndexesOptions) SetMaxTime(duration time.Duration) *DropIndexesOptions {
	d.MaxTime = &duration
	return d
}

// MergeDropIndexesOptions combines the given *DropIndexesOptions into a single *DropIndexesOptions in a last one
// wins fashion.
func MergeDropIndexesOptions(opts ...*DropIndexesOptions) *DropIndexesOptions {
	c := DropIndexes()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.MaxTime != nil {
			c.MaxTime = opt.MaxTime
		}
	}

	return c
}

// ListIndexesOptions represents all possible options for the create() function.
type ListIndexesOptions struct {
	BatchSize *int32
	MaxTime   *time.Duration
}

// ListIndexes creates a new ListIndexesOptions instance.
func ListIndexes() *ListIndexesOptions {
	return &ListIndexesOptions{}
}

// BatchSize specifies the number of documents to return in every batch.
func (l *ListIndexesOptions) SetBatchSize(i int32) *ListIndexesOptions {
	l.BatchSize = &i
	return l
}

// SetMaxTime specifies the maximum amount of time to allow the query to run.
func (l *ListIndexesOptions) SetMaxTime(d time.Duration) *ListIndexesOptions {
	l.MaxTime = &d
	return l
}

// MergeListIndexesOptions combines the given *ListIndexesOptions into a single *ListIndexesOptions in a last one
// wins fashion.
func MergeListIndexesOptions(opts ...*ListIndexesOptions) *ListIndexesOptions {
	c := ListIndexes()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if opt.MaxTime != nil {
			c.MaxTime = opt.MaxTime
		}
	}

	return c
}
