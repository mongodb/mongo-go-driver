// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// DeleteOption is for internal use.
type DeleteOption interface {
	DeleteOptioner

	DeleteName() string
	DeleteValue() interface{}
}

type DeleteOptioner interface {
	Optioner
	deleteOption()
}
