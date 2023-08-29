// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package unified

import (
	"testing"

	"go.mongodb.org/mongo-driver/internal/require"
)

func newTestLogMessage(t *testing.T, level int, msg string, args ...interface{}) *logMessage {
	t.Helper()

	message, err := newLogMessage(level, msg, args...)
	require.NoError(t, err, "failed to create test log message")

	return message
}
