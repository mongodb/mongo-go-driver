// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongoutil

import (
	"reflect"

	"go.mongodb.org/mongo-driver/mongo/options"
)

// Args defines arguments types that can be merged using the functional setters.
type Args interface {
	options.ChangeStreamArgs | options.ClientArgs | options.ClientEncryptionArgs |
		options.CollectionArgs | options.CountArgs | options.CreateCollectionArgs |
		options.DefaultIndexArgs | options.FindArgs | options.FindOneArgs |
		options.InsertOneArgs | options.ListDatabasesArgs | options.SessionArgs |
		options.BulkWriteArgs | options.AggregateArgs |
		options.ListSearchIndexesArgs | options.TimeSeriesArgs |
		options.CreateViewArgs | options.DataKeyArgs | options.DatabaseArgs |
		options.DeleteArgs | options.DistinctArgs | options.DropCollectionArgs |
		options.EncryptArgs | options.RangeArgs |
		options.EstimatedDocumentCountArgs | options.FindOneAndReplaceArgs |
		options.FindOneAndUpdateArgs | options.FindOneAndDeleteArgs |
		options.BucketArgs | options.GridFSUploadArgs | options.GridFSNameArgs |
		options.GridFSFindArgs
}

// MongoOptions is an interface that wraps a method to return a list of setter
// functions that can set a generic arguments type.
type MongoOptions[T Args] interface {
	ArgsSetters() []func(*T) error
}

// NewArgsFromOptions will functionally merge a slice of mongo.Options in a
// "last-one-wins" manner.
func NewArgsFromOptions[T Args](opts ...MongoOptions[T]) (*T, error) {
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

// Options implements a mongo.Options object for an arbitrary arguments type.
// The intended use case is to create options from arguments.
type Options[T Args] struct {
	Args *T             // Arguments to set on the option type
	Clbk func(*T) error // A callback for further modification
}

// ArgsSetters will re-assign the entire argument option to the Args field
// defined on opts. If a callback exists, that function will be executed to
// further modify the arguments.
func (opts *Options[T]) ArgsSetters() []func(*T) error {
	return []func(*T) error{
		func(args *T) error {
			if opts.Args != nil {
				*args = *opts.Args
			}

			if opts.Clbk != nil {
				return opts.Clbk(args)
			}

			return nil
		},
	}
}

func NewOptionsFromArgs[T Args](args *T, clbk func(*T) error) *Options[T] {
	return &Options[T]{Args: args, Clbk: clbk}
}

func AuthFromURI(uri string) (*options.Credential, error) {
	args, err := NewArgsFromOptions[options.ClientArgs](options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	return args.Auth, nil
}

func HostsFromURI(uri string) ([]string, error) {
	args, err := NewArgsFromOptions[options.ClientArgs](options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	return args.Hosts, nil
}
