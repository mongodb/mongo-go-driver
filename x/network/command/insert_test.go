// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/network/description"
)

func TestInsertCommandSplitting(t *testing.T) {
	const (
		megabyte = 10 * 10 * 10 * 10 * 10 * 10
		kilobyte = 10 * 10 * 10
	)

	ss := description.SelectedServer{}
	t.Run("split_smoke_test", func(t *testing.T) {
		i := &Insert{}
		for n := 0; n < 100; n++ {
			i.Docs = append(i.Docs, bsonx.Doc{{"a", bsonx.Int32(int32(n))}})
		}

		batches, err := splitBatches(i.Docs, 10, kilobyte) // 1kb
		assert.NoError(t, err)
		assert.Len(t, batches, 10)
		for _, b := range batches {
			assert.Len(t, b, 10)
			cmd, err := i.encodeBatch(b, ss)
			assert.NoError(t, err)

			wm, err := cmd.Encode(ss)
			assert.NoError(t, err)

			assert.True(t, wm.Len() < 16*megabyte)
		}
	})
	t.Run("split_with_small_target_Size", func(t *testing.T) {
		i := &Insert{}
		for n := 0; n < 100; n++ {
			i.Docs = append(i.Docs, bsonx.Doc{{"a", bsonx.Int32(int32(n))}})
		}

		batches, err := splitBatches(i.Docs, 100, 32) // 32 bytes?
		assert.NoError(t, err)
		assert.Len(t, batches, 50)
		for _, b := range batches {
			assert.Len(t, b, 2)
			cmd, err := i.encodeBatch(b, ss)
			assert.NoError(t, err)

			wm, err := cmd.Encode(ss)
			assert.NoError(t, err)

			assert.True(t, wm.Len() < 16*megabyte)
		}
	})
	t.Run("invalid_max_counts", func(t *testing.T) {
		i := &Insert{}
		for n := 0; n < 100; n++ {
			i.Docs = append(i.Docs, bsonx.Doc{{"a", bsonx.Int32(int32(n))}})
		}

		for _, ct := range []int{-1, 0, -1000} {
			batches, err := splitBatches(i.Docs, ct, 100*megabyte)
			assert.NoError(t, err)
			assert.Len(t, batches, 100)
			for _, b := range batches {
				assert.Len(t, b, 1)
				cmd, err := i.encodeBatch(b, ss)
				assert.NoError(t, err)

				wm, err := cmd.Encode(ss)
				assert.NoError(t, err)

				assert.True(t, wm.Len() < 16*megabyte)
			}
		}

	})
	t.Run("document_larger_than_max_size", func(t *testing.T) {
		i := &Insert{}
		i.Docs = append(i.Docs, bsonx.Doc{{"a", bsonx.String("bcdefghijklmnopqrstuvwxyz")}})
		_, err := splitBatches(i.Docs, 100, 5)
		if err != ErrDocumentTooLarge {
			t.Errorf("Expected a too large error. got %v; want %v", err, ErrDocumentTooLarge)
		}
	})
}
