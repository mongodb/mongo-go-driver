// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"go.mongodb.org/mongo-driver/internal/mongoutil"
)

// Options is an interface that wraps a method to return a list of setter
// functions that can set a generic arguments type.
type Options[T mongoutil.Options] interface {
	OptionsSetters() []func(*T) error
}

// newOptionsFromBuilder wraps the given mongo-level options in the internal
// mongoutil options, merging a slice of options in a last-one-wins algorithm.

func newOptionsFromBuilder[T mongoutil.Options](opts ...Options[T]) (*T, error) {
	mongoOpts := make([]mongoutil.OptionsBuilder[T], len(opts))
	for idx, opt := range opts {
		mongoOpts[idx] = opt
	}

	return mongoutil.NewOptionsFromBuilder(mongoOpts...)
}
