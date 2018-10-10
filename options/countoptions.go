// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// CountOptions represents all possible options to the count() function
type CountOptions struct {
	Collation *Collation // Specifies a collation
	Hint      *Hint      // The index to use
	Limit     *int64     // The maximum number of documents to count
	MaxTimeMS *int32     // The maximum amount of time to allow the operation to run
	Skip      *int64     // The number of documents to skip before counting
}

// Count returns a pointer to a new CountOptions
func Count() *CountOptions {
	return &CountOptions{}
}

// SetCollation specifies a collation
func (co *CountOptions) SetCollation(c Collation) *CountOptions {
	co.Collation = &c
	return co
}

// SetHint specifies the index to use
func (co *CountOptions) SetHint(hint *Hint) *CountOptions {
	co.Hint = hint
	return co
}

// SetLimit specifies the maximum number of documents to count
func (co *CountOptions) SetLimit(i int64) *CountOptions {
	co.Limit = &i
	return co
}

// SetMaxTimeMS specifies the maximum amount of time to allow the operation to run
func (co *CountOptions) SetMaxTimeMS(i int32) *CountOptions {
	co.MaxTimeMS = &i
	return co
}

// SetSkip specifies the number of documents to skip before counting
func (co *CountOptions) SetSkip(i int64) *CountOptions {
	co.Skip = &i
	return co
}

// ToCountOptions combines the argued CountOptions into a single CountOptions in a last-one-wins fashion
func ToCountOptions(opts ...*CountOptions) *CountOptions {
	countOpts := Count()
	for _, co := range opts {
		if co.Collation != nil {
			countOpts.Collation = co.Collation
		}
		if co.Hint != nil {
			countOpts.Hint = co.Hint
		}
		if co.Limit != nil {
			countOpts.Limit = co.Limit
		}
		if co.MaxTimeMS != nil {
			countOpts.MaxTimeMS = co.MaxTimeMS
		}
		if co.Skip != nil {
			countOpts.Skip = co.Skip
		}
	}

	return countOpts
}
