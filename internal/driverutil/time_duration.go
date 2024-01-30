// Copyright (C) MongoDB, Inc. 2024-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package driverutil

import "time"

// TimeDuration is used to seal the WTimeout field on the exported
// writeconcern.WriteConcern object.
type TimeDuration time.Duration
