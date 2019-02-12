// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"testing"

	"go.mongodb.org/mongo-driver/x/bsonx"
)

func TestEndSessions(t *testing.T) {
	t.Run("TestSplitBatches", func(t *testing.T) {
		ids := []bsonx.Doc{}
		for i := 0; i < 2*BatchSize; i++ {
			ids = append(ids, bsonx.Doc{{"x", bsonx.Int32(int32(i))}})
		}

		es := &EndSessions{
			SessionIDs: ids,
		}

		batches := es.split()
		if len(batches) != 2 {
			t.Fatalf("incorrect number of batches. expected 2 got %d", len(batches))
		}

		for i, batch := range batches {
			if len(batch) != BatchSize {
				t.Fatalf("incorrect batch size for batch %d. expected %d got %d", i, BatchSize, len(batch))
			}
		}
	})
}
