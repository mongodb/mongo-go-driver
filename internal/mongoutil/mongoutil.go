// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongoutil

import (
	"reflect"

	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// NewOptions will functionally merge a slice of mongo.Options in a
// "last-one-wins" manner, where nil options are ignored.
func NewOptions[T any](opts ...options.Lister[T]) (*T, error) {
	args := new(T)
	for _, opt := range opts {
		if opt == nil || reflect.ValueOf(opt).IsNil() {
			// Do nothing if the option is nil or if opt is nil but implicitly cast as
			// an Options interface by the NewArgsFromOptions function. The latter
			// case would look something like this:
			continue
		}

		for _, setArgs := range opt.List() {
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

// OptionsLister implements an options.SetterLister object for an arbitrary
// options type.
type OptionsLister[T any] struct {
	Options  *T             // Arguments to set on the option type
	Callback func(*T) error // A callback for further modification
}

// List will re-assign the entire argument option to the Args field
// defined on opts. If a callback exists, that function will be executed to
// further modify the arguments.
func (opts *OptionsLister[T]) List() []func(*T) error {
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

// NewOptionsLister will construct a SetterLister from the provided Options
// object.
func NewOptionsLister[T any](args *T, callback func(*T) error) *OptionsLister[T] {
	return &OptionsLister[T]{Options: args, Callback: callback}
}

// AuthFromURI will create a Credentials object given the provided URI.
func AuthFromURI(uri string) (*options.Credential, error) {
	opts := options.Client().ApplyURI(uri)

	return opts.Auth, nil
}

// HostsFromURI will parse the hosts in the URI and return them as a slice of
// strings.
func HostsFromURI(uri string) ([]string, error) {
	opts := options.Client().ApplyURI(uri)

	return opts.Hosts, nil
}
