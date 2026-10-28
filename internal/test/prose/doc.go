// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package prose runs prose tests that need drivers test secrets. The secrets
// are exported by an AWS SSO login that runs in a container built from a
// Dockerfile kept in the private 10gen/go-driver-tools repo.
package prose
