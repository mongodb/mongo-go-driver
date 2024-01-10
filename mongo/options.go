// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import "go.mongodb.org/mongo-driver/mongo/options"

// Args defines arguments types that can be merged using the functional setters.
type Args interface {
	options.ChangeStreamArgs | options.ClientEncryptionArgs |
		options.FindArgs | options.FindOneArgs | options.InsertOneArgs |
		options.ListDatabasesArgs | options.SessionArgs
}

// Options is an interface that wraps a method to return a list of setter
// functions that can set a generic arguments type.
type Options[T Args] interface {
	ArgsSetters() []func(*T) error
}

var _ Options[options.ChangeStreamArgs] = (*options.ChangeStreamOptions)(nil)
var _ Options[options.ClientEncryptionArgs] = (*options.ClientEncryptionOptions)(nil)
var _ Options[options.FindArgs] = (*options.FindOptions)(nil)
var _ Options[options.FindOneArgs] = (*options.FindOneOptions)(nil)
var _ Options[options.InsertOneArgs] = (*options.InsertOneOptions)(nil)
var _ Options[options.ListDatabasesArgs] = (*options.ListDatabasesOptions)(nil)
var _ Options[options.SessionArgs] = (*options.SessionOptions)(nil)

func isNil[T Args](opt Options[T]) bool {
	if opt == nil {
		return true
	}
	switch any(opt).(type) {
	case *options.ChangeStreamOptions:
		return any(opt).(*options.ChangeStreamOptions) == nil
	case *options.ClientEncryptionOptions:
		return any(opt).(*options.ClientEncryptionOptions) == nil
	case *options.FindOptions:
		return any(opt).(*options.FindOptions) == nil
	case *options.FindOneOptions:
		return any(opt).(*options.FindOneOptions) == nil
	case *options.InsertOneOptions:
		return any(opt).(*options.InsertOneOptions) == nil
	case *options.ListDatabasesOptions:
		return any(opt).(*options.ListDatabasesOptions) == nil
	case *options.SessionOptions:
		return any(opt).(*options.SessionOptions) == nil
	}
	return false
}

// NewArgsFromOptions will functionally merge a slice of mongo.Options in a "last-one-wins" manner.
func NewArgsFromOptions[T Args](opts ...Options[T]) (*T, error) {
	args := new(T)
	for _, opt := range opts {
		if isNil(opt) {
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
