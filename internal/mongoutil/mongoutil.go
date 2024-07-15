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
	options.AggregateOptions | options.BucketArgs | options.BulkWriteOptions |
		options.ClientOptions | options.ClientEncryptionOptions | options.CollectionArgs |
		options.CountArgs | options.CreateIndexesArgs |
		options.CreateCollectionArgs | options.CreateSearchIndexesArgs |
		options.CreateViewArgs | options.DataKeyArgs | options.DatabaseArgs |
		options.DefaultIndexArgs | options.DeleteArgs | options.DistinctArgs |
		options.DropCollectionArgs | options.DropIndexesArgs |
		options.DropSearchIndexArgs | options.EncryptArgs |
		options.EstimatedDocumentCountArgs | options.FindArgs |
		options.FindOneArgs | options.FindOneAndDeleteArgs |
		options.FindOneAndReplaceArgs | options.FindOneAndUpdateArgs |
		options.GridFSFindArgs | options.GridFSNameArgs | options.GridFSUploadArgs |
		options.IndexArgs | options.InsertManyArgs | options.InsertOneArgs |
		options.ListCollectionsArgs | options.ListDatabasesArgs |
		options.ListIndexesArgs | options.ListSearchIndexesArgs |
		options.LoggerArgs | options.RangeArgs | options.ReplaceArgs |
		options.RewrapManyDataKeyArgs | options.RunCmdArgs |
		options.SearchIndexesArgs | options.ServerAPIArgs | options.SessionArgs |
		options.TimeSeriesArgs | options.TransactionArgs | options.UpdateArgs |
		options.UpdateSearchIndexArgs | options.ChangeStreamOptions |
		options.AutoEncryptionOptions
}

// MongoOptions is an interface that wraps a method to return a list of setter
// functions for merging options for a generic argument type.
type MongoOptions[T Args] interface {
	ArgsSetters() []func(*T) error
}

// NewArgsFromOptions will functionally merge a slice of mongo.Options in a
// "last-one-wins" manner, where nil options are ignored.
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

// ArgOptions implements a mongo.ArgOptions object for an arbitrary arguments type.
// The intended use case is to create options from arguments.
type ArgOptions[T Args] struct {
	Args     *T             // Arguments to set on the option type
	Callback func(*T) error // A callback for further modification
}

// ArgsSetters will re-assign the entire argument option to the Args field
// defined on opts. If a callback exists, that function will be executed to
// further modify the arguments.
func (opts *ArgOptions[T]) ArgsSetters() []func(*T) error {
	return []func(*T) error{
		func(args *T) error {
			if opts.Args != nil {
				*args = *opts.Args
			}

			if opts.Callback != nil {
				return opts.Callback(args)
			}

			return nil
		},
	}
}

// NewOptionsFromArgs will construct an Options object from the provided
// arguments object.
func NewOptionsFromArgs[T Args](args *T) *ArgOptions[T] {
	return &ArgOptions[T]{Args: args}
}

// AuthFromURI will create a Credentials object given the provided URI.
func AuthFromURI(uri string) (*options.Credential, error) {
	args, err := NewArgsFromOptions[options.ClientOptions](options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	return args.Auth, nil
}

// HostsFromURI will parse the hosts in the URI and return them as a slice of
// strings.
func HostsFromURI(uri string) ([]string, error) {
	args, err := NewArgsFromOptions[options.ClientOptions](options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	return args.Hosts, nil
}
