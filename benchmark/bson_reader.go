// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package benchmark

import (
	"context"
	"errors"
)

func bsonReaderDecoding(ctx context.Context, tm TimerManager, iters, numKeys int, source string) error {
	doc, err := loadSourceReader(getProjectRoot(), perfDataDir, bsonDataDir, source)
	if err != nil {
		return err
	}

	tm.ResetTimer()

	for i := 0; i < iters; i++ {
		keys, err := doc.Keys(true)
		if err != nil {
			return err
		}
		if len(keys) != numKeys {
			return errors.New("reader parsing error")
		}
	}

	return nil
}

func BSONFlatReaderDecoding(ctx context.Context, tm TimerManager, iters int) error {
	return bsonReaderDecoding(ctx, tm, iters, 145, flatBSONData)
}

func BSONDeepReaderDecoding(ctx context.Context, tm TimerManager, iters int) error {
	return bsonReaderDecoding(ctx, tm, iters, 126, deepBSONData)
}

func BSONFullReaderDecoding(ctx context.Context, tm TimerManager, iters int) error {
	return bsonReaderDecoding(ctx, tm, iters, 145, fullBSONData)
}
