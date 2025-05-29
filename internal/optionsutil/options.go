// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package optionsutil

// Options stores internal options.
type Options struct {
	values map[string]any
}

// WithValue sets an option value with the associated key.
func WithValue(opts Options, key string, option any) Options {
	if opts.values == nil {
		opts.values = make(map[string]any)
	}
	opts.values[key] = option
	return opts
}

// Value returns the value associated with the options for key.
func Value(opts Options, key string) any {
	if opts.values == nil {
		return nil
	}
	if val, ok := opts.values[key]; ok {
		return val
	}
	return nil
}

// Equal compares two Options instances for equality.
func Equal(opts1, opts2 Options) bool {
	if len(opts1.values) != len(opts2.values) {
		return false
	}
	for key, val1 := range opts1.values {
		if val2, ok := opts2.values[key]; !ok || val1 != val2 {
			return false
		}
	}
	return true
}
