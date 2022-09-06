// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package integration

// This file exists to allow the build scripts (and standard Go builds for some early Go versions)
// to succeed. Without it, the build may encounter an error like:
//
//   go build go.mongodb.org/mongo-driver/mongo/integration: build constraints exclude all Go files in ./go.mongodb.org/mongo-driver/mongo/integration
//
