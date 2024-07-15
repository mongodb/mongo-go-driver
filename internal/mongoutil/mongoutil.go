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

// Options defines options types that can be merged using the functional
// setters.
type Options interface {
	options.AggregateOptions | options.BucketOptions | options.BulkWriteOptions |
		options.ClientOptions | options.ClientEncryptionOptions | options.CollectionOptions |
		options.CountOptions | options.CreateIndexesOptions |
		options.CreateCollectionOptions | options.CreateSearchIndexesOptions |
		options.CreateViewOptions | options.DataKeyOptions | options.DatabaseOptions |
		options.DefaultIndexOptions | options.DeleteOptions | options.DistinctOptions |
		options.DropCollectionOptions | options.DropIndexesOptions |
		options.DropSearchIndexOptions | options.EncryptOptions |
		options.EstimatedDocumentCountOptions | options.FindOptions |
		options.FindOneOptions | options.FindOneAndDeleteOptions |
		options.FindOneAndReplaceOptions | options.FindOneAndUpdateOptions |
		options.GridFSFindOptions | options.GridFSNameOptions | options.GridFSUploadOptions |
		options.IndexOptions | options.InsertManyOptions | options.InsertOneOptions |
		options.ListCollectionsOptions | options.ListDatabasesOptions |
		options.ListIndexesOptions | options.ListSearchIndexesOptions |
		options.LoggerOptions | options.RangeOptions | options.ReplaceOptions |
		options.RewrapManyDataKeyOptions | options.RunCmdOptions |
		options.SearchIndexesOptions | options.ServerAPIOptions | options.SessionOptions |
		options.TimeSeriesOptions | options.TransactionOptions | options.UpdateOptions |
		options.UpdateSearchIndexOptions | options.ChangeStreamOptions |
		options.AutoEncryptionOptions
}

// OptionsBuilder is an interface that wraps a method to return a list of setter
// functions for merging options for a generic options type.
type OptionsBuilder[T Options] interface {
	OptionsSetters() []func(*T) error
}

// NewOptionsFromBuilder will functionally merge a slice of mongo.Options in a
// "last-one-wins" manner, where nil options are ignored.
func NewOptionsFromBuilder[T Options](opts ...OptionsBuilder[T]) (*T, error) {
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

		for _, setArgs := range opt.OptionsSetters() {
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

// OptionsBuilderWithCallback implements a mongo.OptionsBuilder object for an
// arbitrary options type. The intended use case is to create options from
// options.
type OptionsBuilderWithCallback[T Options] struct {
	Options  *T             // Arguments to set on the option type
	Callback func(*T) error // A callback for further modification
}

// OptionsSetters will re-assign the entire argument option to the Args field
// defined on opts. If a callback exists, that function will be executed to
// further modify the arguments.
func (opts *OptionsBuilderWithCallback[T]) OptionsSetters() []func(*T) error {
	return []func(*T) error{
		func(args *T) error {
			if opts.Options != nil {
				*args = *opts.Options
			}

			if opts.Callback != nil {
				return opts.Callback(args)
			}

			return nil
		},
	}
}

// NewBuilderFromOptions will construct an OptionsBuilder object from the
// provided Options object.
func NewBuilderFromOptions[T Options](args *T) *OptionsBuilderWithCallback[T] {
	return &OptionsBuilderWithCallback[T]{Options: args}
}

// AuthFromURI will create a Credentials object given the provided URI.
func AuthFromURI(uri string) (*options.Credential, error) {
	args, err := NewOptionsFromBuilder[options.ClientOptions](options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	return args.Auth, nil
}

// HostsFromURI will parse the hosts in the URI and return them as a slice of
// strings.
func HostsFromURI(uri string) ([]string, error) {
	args, err := NewOptionsFromBuilder[options.ClientOptions](options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	return args.Hosts, nil
}
