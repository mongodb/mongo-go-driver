// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

type CountOptions struct {
	Collation *Collation // Specifies a collation
	Hint      *Hint       // The index to use
	Limit     *int64     // The maximum number of documents to count
	MaxTimeMS *int32     // The maximum amount of time to allow the operation to run
	Skip      *int64     // The number of documents to skip before counting
}

func Count() *CountOptions {
	return &CountOptions{}
}

func (co *CountOptions) SetCollation(c Collation) *CountOptions {
	co.Collation = &c
	return co
}

func (co *CountOptions) SetHint(hint *Hint) *CountOptions {
	co.Hint = hint
	return co
}

func (co *CountOptions) SetLimit(i int64) *CountOptions {
	co.Limit = &i
	return co
}

func (co *CountOptions) SetMaxTimeMS(i int32) *CountOptions {
	co.MaxTimeMS = &i
	return co
}

func (co *CountOptions) SetSkip(i int64) *CountOptions {
	co.Skip = &i
	return co
}

func ToCountOptions(opts ...*CountOptions) *CountOptions {
	var countOpts *CountOptions
	if len(opts) >= 1 {
		countOpts = opts[len(opts)-1]
	} else {
		countOpts = Count()
	}

	return countOpts
}
