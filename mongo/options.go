// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import "go.mongodb.org/mongo-driver/mongo/options"

// Args defines arguments types that can be merged using the functional setters.
type Args interface {
	options.FindArgs | options.FindOneArgs | options.InsertOneArgs
}

// Options is an interface that wraps a method to return a list of setter
// functions that can set a generic arguments type.
type Options[T Args] interface {
	ArgsSetters() []func(*T) error
}

var _ Options[options.FindArgs] = (*options.FindOptions)(nil)
var _ Options[options.FindOneArgs] = (*options.FindOneOptions)(nil)
var _ Options[options.InsertOneArgs] = (*options.InsertOneOptions)(nil)

// NewArgsFromOptions will functionally merge a slice of mongo.Options in a "last-one-wins" manner.
func NewArgsFromOptions[T Args](opts ...Options[T]) (*T, error) {
	args := new(T)
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		for _, setArgs := range opt.ArgsSetters() {
			if setArgs == nil {
				continue
			}
			if err := setArgs(args); err != nil {
				return nil, err
			}
		}
	}
	return args, nil
}
