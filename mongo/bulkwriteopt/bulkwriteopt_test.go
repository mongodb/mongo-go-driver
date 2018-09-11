// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bulkwriteopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var collation = &mongoopt.Collation{}

func createNestedBulkWriteBundle1(t *testing.T) *BulkWriteBundle {
	nestedBundle := BundleBulkWrite(Ordered(false))
	testhelpers.RequireNotNil(t, nestedBundle, "nested bundle was nil")

	outerBundle := BundleBulkWrite(Ordered(true), BypassDocumentValidation(true), nestedBundle, BypassDocumentValidation(false))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test doubly nested bundle
func createNestedBulkWriteBundle2(t *testing.T) *BulkWriteBundle {
	b1 := BundleBulkWrite(Ordered(false))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleBulkWrite(Ordered(true), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	outerBundle := BundleBulkWrite(Ordered(true), BypassDocumentValidation(true), b2, BypassDocumentValidation(false))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test two top level nested bundles
func createNestedBulkWriteBundle3(t *testing.T) *BulkWriteBundle {
	b1 := BundleBulkWrite(Ordered(false))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleBulkWrite(BypassDocumentValidation(true), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	b3 := BundleBulkWrite(Ordered(true))
	testhelpers.RequireNotNil(t, b3, "nested bundle was nil")

	b4 := BundleBulkWrite(BypassDocumentValidation(true), b3)
	testhelpers.RequireNotNil(t, b4, "nested bundle was nil")

	outerBundle := BundleBulkWrite(b4, BypassDocumentValidation(true), b2, BypassDocumentValidation(false))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

func TestBulkWriteOpt(t *testing.T) {
	var bundle1 *BulkWriteBundle
	bundle1 = bundle1.Ordered(true).BypassDocumentValidation(false)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := &BulkWriteOptions{
		Ordered:                     true,
		OrderedSet:                  true,
		BypassDocumentValidation:    false,
		BypassDocumentValidationSet: true,
	}

	bundle2 := BundleBulkWrite().
		BypassDocumentValidation(true).
		Ordered(false).
		Ordered(true)

	bundle2Opts := &BulkWriteOptions{
		Ordered:                     true,
		OrderedSet:                  true,
		BypassDocumentValidation:    true,
		BypassDocumentValidationSet: true,
	}

	nilBundle := BundleBulkWrite()
	nilBundleOpts := &BulkWriteOptions{
		Ordered: true,
	}

	nestedBundle1 := createNestedBulkWriteBundle1(t)
	nestedBundleOpts1 := &BulkWriteOptions{
		Ordered:                     false,
		OrderedSet:                  true,
		BypassDocumentValidation:    false,
		BypassDocumentValidationSet: true,
	}

	nestedBundle2 := createNestedBulkWriteBundle2(t)
	nestedBundleOpts2 := &BulkWriteOptions{
		Ordered:                     false,
		OrderedSet:                  true,
		BypassDocumentValidation:    false,
		BypassDocumentValidationSet: true,
	}

	nestedBundle3 := createNestedBulkWriteBundle3(t)
	nestedBundleOpts3 := &BulkWriteOptions{
		Ordered:                     false,
		OrderedSet:                  true,
		BypassDocumentValidation:    false,
		BypassDocumentValidationSet: true,
	}

	t.Run("TestAll", func(t *testing.T) {
		opts := []BulkWrite{
			BypassDocumentValidation(true),
			Ordered(false),
		}
		expected := &BulkWriteOptions{
			Ordered:                     false,
			OrderedSet:                  true,
			BypassDocumentValidation:    true,
			BypassDocumentValidationSet: true,
		}
		params := make([]BulkWrite, len(opts))
		for i := range opts {
			params[i] = opts[i]
		}
		bundle := BundleBulkWrite(params...)

		bulkWriteOpts, _, err := bundle.Unbundle()
		testhelpers.RequireNil(t, err, "got non-nill error from unbundle: %s", err)

		if !reflect.DeepEqual(bulkWriteOpts, expected) {
			t.Errorf("BulkWrite Options not equal\nexpected %v\nactual %v", expected, bulkWriteOpts)
		}
	})

	t.Run("Nil Option Bundle", func(t *testing.T) {
		sess := BulkWriteSessionOpt{}
		opts, _, err := BundleBulkWrite(Ordered(true), BundleBulkWrite(nil), sess, nil).Unbundle()
		testhelpers.RequireNil(t, err, "got non-nil error from unbundle: %s", err)
		expected := &BulkWriteOptions{
			Ordered:                     true,
			OrderedSet:                  true,
			BypassDocumentValidationSet: false,
		}
		if !reflect.DeepEqual(opts, expected) {
			t.Errorf("BulkWrite Options not equal\nexpected %v\nactual %v", expected, opts)
		}

		opts, _, err = BundleBulkWrite(nil, sess, BundleBulkWrite(nil), Ordered(true)).Unbundle()
		testhelpers.RequireNil(t, err, "got non-nil error from unbundle: %s", err)

		if !reflect.DeepEqual(opts, expected) {
			t.Errorf("BulkWrite Options not equal\nexpected %v\nactual %v", expected, opts)
		}
	})

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name         string
			bundle       *BulkWriteBundle
			expectedOpts *BulkWriteOptions
		}{
			{"NilBundle", nilBundle, nilBundleOpts},
			{"Bundle1", bundle1, bundle1Opts},
			{"Bundle2", bundle2, bundle2Opts},
			{"NestedBundle1", nestedBundle1, nestedBundleOpts1},
			{"NestedBundle2", nestedBundle2, nestedBundleOpts2},
			{"NestedBundle3", nestedBundle3, nestedBundleOpts3},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				options, _, err := tc.bundle.Unbundle()
				testhelpers.RequireNil(t, err, "got non-nill error from unbundle: %s", err)

				if !reflect.DeepEqual(options, tc.expectedOpts) {
					t.Errorf("options does not match expected options. \nreceived: %v\nexpected %v", options, tc.expectedOpts)
				}
			})
		}
	})
}
