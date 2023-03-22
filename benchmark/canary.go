// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package benchmark

import (
	"context"
)

// CanaryIncCase is a no-op.
//
// Deprecated: CanaryIncCase has no observable effect, so recent versions of the Go compiler may
// bypass calls to it in the compiled binary. It should not be used in benchmarks.
func CanaryIncCase(context.Context, TimerManager, int) error {
	return nil
}

// GlobalCanaryIncCase is a no-op.
//
// Deprecated: GlobalCanaryIncCase has no observable effect, so recent versions of the Go compiler
// may bypass calls to it in the compiled binary. It should not be used in benchmarks.
func GlobalCanaryIncCase(context.Context, TimerManager, int) error {
	return nil
}
