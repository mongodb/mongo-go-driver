// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package version defines the Go Driver version.
package version // import "go.mongodb.org/mongo-driver/version"

// Driver is the current version of the driver.
var Driver = "v1.11.0-prerelease"

// MinSupportedWire is the minimum wire version supported by the driver.
var MinSupportedWire int32 = 6

// MinSupportedWire is the maximum wire version supported by the driver.
var MaxSupportedWire int32 = 17

// MinSupportedMongoDB is the version string for the lowest MongoDB version supported by the driver.
var MinSupportedMongoDB = "3.6"
