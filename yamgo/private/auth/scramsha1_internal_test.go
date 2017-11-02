// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// In order to test that the SCRAM-SHA client key is cached on successful authentication, we need
// to be able to access the clientKey field on ScramSHA1Authenticator. In order to forgo any
// user-facing API, the IsClientKeyNil() method is defined in a *_test.go file so that it only
// builds with `go test` (and not `go build`).

package auth

func (a *ScramSHA1Authenticator) IsClientKeyNil() bool {
	return a.clientKey == nil
}
