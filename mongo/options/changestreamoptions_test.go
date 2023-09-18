// Copyright (C) MongoDB, Inc. 2023-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package options

import (
	"testing"

	"go.mongodb.org/mongo-driver/internal/assert"
)

func TestMergeChangeStreamOptions(t *testing.T) {
	t.Parallel()

	fullDocumentP := func(x FullDocument) *FullDocument { return &x }
	int32P := func(x int32) *int32 { return &x }

	testCases := []struct {
		description string
		input       []*ChangeStreamOptions
		want        *ChangeStreamOptions
	}{
		{
			description: "empty",
			input:       []*ChangeStreamOptions{},
			want:        &ChangeStreamOptions{},
		},
		{
			description: "many ChangeStreamOptions with one configuration each",
			input: []*ChangeStreamOptions{
				ChangeStream().SetFullDocumentBeforeChange(Required),
				ChangeStream().SetFullDocument(Required),
				ChangeStream().SetBatchSize(10),
			},
			want: &ChangeStreamOptions{
				FullDocument:             fullDocumentP(Required),
				FullDocumentBeforeChange: fullDocumentP(Required),
				BatchSize:                int32P(10),
			},
		},
		{
			description: "single ChangeStreamOptions with many configurations",
			input: []*ChangeStreamOptions{
				ChangeStream().
					SetFullDocumentBeforeChange(Required).
					SetFullDocument(Required).
					SetBatchSize(10),
			},
			want: &ChangeStreamOptions{
				FullDocument:             fullDocumentP(Required),
				FullDocumentBeforeChange: fullDocumentP(Required),
				BatchSize:                int32P(10),
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			got := MergeChangeStreamOptions(tc.input...)
			assert.Equal(t, tc.want, got, "expected and actual ChangeStreamOptions are different")
		})
	}
}
