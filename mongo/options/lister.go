// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// Lister is an interface that wraps a List method to return a
// slice of option setters.
type Lister[T any] interface {
	List() []func(*T) error
}
