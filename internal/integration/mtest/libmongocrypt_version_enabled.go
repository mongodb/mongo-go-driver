// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

//go:build cse

package mtest

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/mongocrypt"
)

// verifyLibmongocryptVersionConstraints returns an error if the loaded
// libmongocrypt version is not in the range [min, max]. Bounds are only
// checked when non-empty.
func verifyLibmongocryptVersionConstraints(minVersion, maxVersion string) error {
	if minVersion == "" && maxVersion == "" {
		return nil
	}
	version := mongocrypt.Version()
	if minVersion != "" && CompareServerVersions(version, minVersion) < 0 {
		return fmt.Errorf("libmongocrypt version %q is lower than min required version %q", version, minVersion)
	}
	if maxVersion != "" && CompareServerVersions(version, maxVersion) > 0 {
		return fmt.Errorf("libmongocrypt version %q is higher than max version %q", version, maxVersion)
	}
	return nil
}
