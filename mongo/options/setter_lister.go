// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// Lister is an interface that wraps a method to return a list of setter
// functions that can set a generic arguments type.

// SetterLister is an interface that wraps a ListSetters method to return a
// slice of option setters.
type SetterLister[T any] interface {
	ListSetters() []func(*T) error
}

func getOptions[T any](mopts SetterLister[T]) (*T, error) {
	opts := new(T)

	for _, setterFn := range mopts.ListSetters() {
		if setterFn == nil {
			continue
		}

		if err := setterFn(opts); err != nil {
			return nil, err
		}
	}

	return opts, nil
}
