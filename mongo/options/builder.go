// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// Builder is an interface that wraps a method to return a list of setter
// functions that can set a generic arguments type.
type Builder[T any] interface {
	OptionsSetters() []func(*T) error
}

func getOptions[T any](mopts Builder[T]) (*T, error) {
	opts := new(T)

	for _, setOptions := range mopts.OptionsSetters() {
		if setOptions == nil {
			continue
		}

		if err := setOptions(opts); err != nil {
			return nil, err
		}
	}

	return opts, nil
}
