// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"fmt"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

var (
	supportedSchemaVersions = map[int]string{
		1: "1.9",
	}
)

// checkSchemaVersion determines if the provided schema version is supported and returns an error if it is not.
func checkSchemaVersion(version string) error {
	// First get the major version number from the schema. The schema version string should be in the format
	// "major.minor.patch", "major.minor", or "major".

	parts := strings.Split(version, ".")
	if len(parts) == 0 {
		return fmt.Errorf("error splitting schema version %q into parts", version)
	}

	majorVersion, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("error converting major version component %q to an integer: %v", parts[0], err)
	}

	// Find the latest supported version for the major version and use that to determine if the provided version is
	// supported.
	supportedVersion, ok := supportedSchemaVersions[majorVersion]
	if !ok {
		return fmt.Errorf("major version %d not supported", majorVersion)
	}
	if mtest.CompareServerVersions(supportedVersion, version) < 0 {
		return fmt.Errorf(
			"latest version supported for major version %d is %q, which is incompatible with specified version %q",
			majorVersion, supportedVersion, version,
		)
	}
	return nil
}
