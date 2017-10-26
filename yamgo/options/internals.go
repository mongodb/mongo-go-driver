// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import "time"

// The types in this file are used internally and should not need to be used directly by users.
// Instead, a number of helper functions in yamgo/options.go are provided to create these options
// more easily.
//
// To see which options are valid for a given operation, see the corresponding file in
// yamgo/options (yamgo/options/count.go, yamgo/options/find.go, etc.)

// OptAllowDiskUse is for internal use.
type OptAllowDiskUse bool

// OptAllowPartialResults is for internal use.
type OptAllowPartialResults bool

// OptArrayFilters is for internal use.
type OptArrayFilters []interface{}

// OptBatchSize is for internal use.
type OptBatchSize int32

// OptBypassDocumentValidation is for internal use.
type OptBypassDocumentValidation bool

// OptCollation is for internal use.
type OptCollation struct{ Collation *CollationOptions }

// OptComment is for internal use.
type OptComment string

// OptCursorType is for internal use.
type OptCursorType CursorType

// OptHint is for internal use.
type OptHint struct{ Hint interface{} }

// OptLimit is for internal use.
type OptLimit int64

// OptMax is for internal use.
type OptMax struct{ Max interface{} }

// OptMaxAwaitTime is for internal use.
type OptMaxAwaitTime time.Duration

// OptMaxScan is for internal use.
type OptMaxScan int64

// OptMaxTime is for internal use.
type OptMaxTime time.Duration

// OptMin is for internal use.
type OptMin struct{ Min interface{} }

// OptNoCursorTimeout is for internal use.
type OptNoCursorTimeout bool

// OptOplogReplay is for internal use.
type OptOplogReplay bool

// OptOrdered is for internal use.
type OptOrdered bool

// OptProjection is for internal use.
type OptProjection struct{ Projection interface{} }

// OptReturnDocument is for internal use.
type OptReturnDocument ReturnDocument

// OptReturnKey is for internal use.
type OptReturnKey bool

// OptShowRecordID is for internal use.
type OptShowRecordID bool

// OptSkip is for internal use.
type OptSkip int64

// OptSnapshot is for internal use.
type OptSnapshot bool

// OptSort is for internal use.
type OptSort struct{ Sort interface{} }

// OptUpsert is for internal use.
type OptUpsert bool
