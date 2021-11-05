// Copyright (C) MongoDB, Inc. 2021-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package cryptx

import (
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver"
)

// SetCrypt specifies a custom driver.Crypt to be used to encrypt and decrypt documents. The default is no encryption.
//
// This API is unstable and should not be used unless you are perfectly confident in the provided, custom implementation of
// driver.Crypt.
func SetCrypt(co *options.ClientOptions, cr driver.Crypt) *options.ClientOptions {
	co.Crypt = cr
	return co
}
