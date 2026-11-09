// Copyright (C) MongoDB, Inc. 2025-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package optionsutil

// Options stores internal options.
//
// values is held behind a pointer so that Options, and the options structs that
// embed it, stay comparable.
//
// Equality is therefore by pointer identity rather than by contents: two
// Options holding the same values are not equal, and two Options sharing a map
// through a copy stay equal after WithValue mutates one of them. Setting an
// internal option on an options struct changes == on that struct to compare
// pointer identity. Use Equal for value comparison.
type Options struct {
	values *map[string]any
}

// WithValue sets an option value with the associated key.
func WithValue(opts Options, key string, option any) Options {
	if opts.values == nil {
		m := make(map[string]any)
		opts.values = &m
	}
	(*opts.values)[key] = option
	return opts
}

// Value returns the value associated with the options for key.
func Value(opts Options, key string) any {
	if opts.values == nil {
		return nil
	}
	return (*opts.values)[key]
}

// Equal compares two Options instances for equality.
func Equal(opts1, opts2 Options) bool {
	var m1, m2 map[string]any
	if opts1.values != nil {
		m1 = *opts1.values
	}
	if opts2.values != nil {
		m2 = *opts2.values
	}

	if len(m1) != len(m2) {
		return false
	}
	for key, val1 := range m1 {
		if val2, ok := m2[key]; !ok || val1 != val2 {
			return false
		}
	}
	return true
}
