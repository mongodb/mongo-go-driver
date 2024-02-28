// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongoutil

// Options implements a mongo.Options object for an arbitrary arguments type.
// The intended use case is to create options from arguments.
type Options[T any] struct {
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
