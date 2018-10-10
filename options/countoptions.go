// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

type CountOptions struct {
	Collation *Collation
	Limit     *int64
	Skip      *int64
	Hint      interface{}
	MaxTimeMS *int32
}

func Count() *CountOptions {
	return &CountOptions{}
}

func (co *CountOptions) SetCollation(c Collation) *CountOptions {
	co.Collation = &c
	return co
}

func (co *CountOptions) SetLimit(i int64) *CountOptions {
	co.Limit = &i
	return co
}

func (co *CountOptions) SetSkip(i int64) *CountOptions {
	co.Skip = &i
	return co
}

func (co *CountOptions) SetHint(hint interface{}) *CountOptions {
	co.Hint = hint
	return co
}

func (co *CountOptions) SetMaxTimeMS(i int32) *CountOptions {
	co.MaxTimeMS = &i
	return co
}
