// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"reflect"

	"go.mongodb.org/mongo-driver/mongo/options"
)

// Args defines arguments types that can be merged using the functional setters.
type Args interface {
	options.ChangeStreamArgs | options.ClientEncryptionArgs |
		options.FindArgs | options.FindOneArgs | options.InsertOneArgs |
		options.ListDatabasesArgs | options.SessionArgs | options.BulkWriteArgs |
		options.AggregateArgs | options.ListSearchIndexesArgs
}

// Options is an interface that wraps a method to return a list of setter
// functions that can set a generic arguments type.
type Options[T Args] interface {
	ArgsSetters() []func(*T) error
}

// NewArgsFromOptions will functionally merge a slice of mongo.Options in a "last-one-wins" manner.
func NewArgsFromOptions[T Args](opts ...Options[T]) (*T, error) {
	args := new(T)
	for _, opt := range opts {
		if opt == nil || reflect.ValueOf(opt).IsNil() {
			// Do nothing if the option is nil or if opt is nil but implicitly cast as
			// an Options interface by the NewArgsFromOptions function. The latter
			// case would look something like this:
			//
			// var opt *SomeOptions
			// NewArgsFromOptions(opt)
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
