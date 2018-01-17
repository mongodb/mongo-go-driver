// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

// CountOption is for internal use.
type CountOption interface {
	CountOptioner

	CountName() string
	CountValue() interface{}
}

type CountOptioner interface {
	Optioner
	countOption()
}
