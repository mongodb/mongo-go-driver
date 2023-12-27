// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// Args defines arguments types that can be merged using the functional setters.
type Args interface {
	FindArgs | FindOneArgs
}

// Options is an interface that wraps a method to return a list of setter
// functions that can set a generic arguments type.
type Options[T Args] interface {
	ArgsSetters() []func(*T) error
}

// Merge will functionally merge a slice of Options in a "last-one-wins" manner.
func Merge[T Args](args *T, opts ...Options[T]) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		for _, setArgs := range opt.ArgsSetters() {
			if setArgs == nil {
				continue
			}
			if err := setArgs(args); err != nil {
				return err
			}
		}
	}
	return nil
}
