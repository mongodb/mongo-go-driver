// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package updateopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var collation = &mongoopt.Collation{}

func createNestedUpdateBundle1(t *testing.T) *UpdateBundle {
	nestedBundle := BundleUpdate(Upsert(false))
	testhelpers.RequireNotNil(t, nestedBundle, "nested bundle was nil")

	outerBundle := BundleUpdate(Upsert(true), BypassDocumentValidation(true), nestedBundle, BypassDocumentValidation(false))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test doubly nested bundle
func createNestedUpdateBundle2(t *testing.T) *UpdateBundle {
	b1 := BundleUpdate(Upsert(false))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleUpdate(Collation(collation), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	outerBundle := BundleUpdate(Upsert(true), BypassDocumentValidation(true), b2, BypassDocumentValidation(false))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test two top level nested bundles
func createNestedUpdateBundle3(t *testing.T) *UpdateBundle {
	b1 := BundleUpdate(Upsert(false))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleUpdate(Collation(collation), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	b3 := BundleUpdate(Upsert(true))
	testhelpers.RequireNotNil(t, b3, "nested bundle was nil")

	b4 := BundleUpdate(Collation(collation), b3)
	testhelpers.RequireNotNil(t, b4, "nested bundle was nil")

	outerBundle := BundleUpdate(b4, BypassDocumentValidation(true), b2, BypassDocumentValidation(false))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

func TestUpdateOpt(t *testing.T) {
	var bundle1 *UpdateBundle
	bundle1 = bundle1.Upsert(true).BypassDocumentValidation(false)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []option.Optioner{
		OptUpsert(true).ConvertUpdateOption(),
		OptBypassDocumentValidation(false).ConvertUpdateOption(),
	}
	bundle1DedupOpts := []option.Optioner{
		OptUpsert(true).ConvertUpdateOption(),
		OptBypassDocumentValidation(false).ConvertUpdateOption(),
	}

	bundle2 := BundleUpdate(Collation(collation))
	bundle2Opts := []option.Optioner{
		OptCollation{collation.Convert()}.ConvertUpdateOption(),
	}

	bundle3 := BundleUpdate().
		Collation(collation).
		BypassDocumentValidation(true).
		Upsert(false).
		Upsert(true)

	bundle3Opts := []option.Optioner{
		OptCollation{collation.Convert()}.ConvertUpdateOption(),
		OptBypassDocumentValidation(true).ConvertUpdateOption(),
		OptUpsert(false).ConvertUpdateOption(),
		OptUpsert(true).ConvertUpdateOption(),
	}

	bundle3DedupOpts := []option.Optioner{
		OptCollation{collation.Convert()}.ConvertUpdateOption(),
		OptBypassDocumentValidation(true).ConvertUpdateOption(),
		OptUpsert(true).ConvertUpdateOption(),
	}

	nilBundle := BundleUpdate()
	var nilBundleOpts []option.Optioner

	nestedBundle1 := createNestedUpdateBundle1(t)
	nestedBundleOpts1 := []option.Optioner{
		OptUpsert(true).ConvertUpdateOption(),
		OptBypassDocumentValidation(true).ConvertUpdateOption(),
		OptUpsert(false).ConvertUpdateOption(),
		OptBypassDocumentValidation(false).ConvertUpdateOption(),
	}
	nestedBundleDedupOpts1 := []option.Optioner{
		OptUpsert(false).ConvertUpdateOption(),
		OptBypassDocumentValidation(false).ConvertUpdateOption(),
	}

	nestedBundle2 := createNestedUpdateBundle2(t)
	nestedBundleOpts2 := []option.Optioner{
		OptUpsert(true).ConvertUpdateOption(),
		OptBypassDocumentValidation(true).ConvertUpdateOption(),
		OptCollation{collation.Convert()}.ConvertUpdateOption(),
		OptUpsert(false).ConvertUpdateOption(),
		OptBypassDocumentValidation(false).ConvertUpdateOption(),
	}
	nestedBundleDedupOpts2 := []option.Optioner{
		OptCollation{collation.Convert()}.ConvertUpdateOption(),
		OptUpsert(false).ConvertUpdateOption(),
		OptBypassDocumentValidation(false).ConvertUpdateOption(),
	}

	nestedBundle3 := createNestedUpdateBundle3(t)
	nestedBundleOpts3 := []option.Optioner{
		OptCollation{collation.Convert()}.ConvertUpdateOption(),
		OptUpsert(true).ConvertUpdateOption(),
		OptBypassDocumentValidation(true).ConvertUpdateOption(),
		OptCollation{collation.Convert()}.ConvertUpdateOption(),
		OptUpsert(false).ConvertUpdateOption(),
		OptBypassDocumentValidation(false).ConvertUpdateOption(),
	}
	nestedBundleDedupOpts3 := []option.Optioner{
		OptCollation{collation.Convert()}.ConvertUpdateOption(),
		OptUpsert(false).ConvertUpdateOption(),
		OptBypassDocumentValidation(false).ConvertUpdateOption(),
	}

	t.Run("TestAll", func(t *testing.T) {
		filters := []interface{}{"filter1", "filter2"}
		c := &mongoopt.Collation{
			Locale: "string locale",
		}

		opts := []Update{
			ArrayFilters(filters),
			BypassDocumentValidation(true),
			Collation(c),
			Upsert(false),
		}

		bundle := BundleUpdate(opts...)

		deleteOpts, err := bundle.Unbundle(true)
		testhelpers.RequireNil(t, err, "got non-nill error from unbundle: %s", err)

		if len(deleteOpts) != len(opts) {
			t.Errorf("expected unbundled opts len %d. got %d", len(opts), len(deleteOpts))
		}

		for i, opt := range opts {
			if !reflect.DeepEqual(opt.ConvertUpdateOption(), deleteOpts[i]) {
				t.Errorf("opt mismatch. expected %#v, got %#v", opt, deleteOpts[i])
			}
		}
	})

	t.Run("MakeOptions", func(t *testing.T) {
		head := bundle1

		bundleLen := 0
		for head != nil && head.option != nil {
			bundleLen++
			head = head.next
		}

		if bundleLen != len(bundle1Opts) {
			t.Errorf("expected bundle length %d. got: %d", len(bundle1Opts), bundleLen)
		}
	})

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name         string
			dedup        bool
			bundle       *UpdateBundle
			expectedOpts []option.Optioner
		}{
			{"NilBundle", false, nilBundle, nilBundleOpts},
			{"Bundle1", false, bundle1, bundle1Opts},
			{"Bundle1Dedup", true, bundle1, bundle1DedupOpts},
			{"Bundle2", false, bundle2, bundle2Opts},
			{"Bundle2Dedup", true, bundle2, bundle2Opts},
			{"Bundle3", false, bundle3, bundle3Opts},
			{"Bundle3Dedup", true, bundle3, bundle3DedupOpts},
			{"NestedBundle1_DedupFalse", false, nestedBundle1, nestedBundleOpts1},
			{"NestedBundle1_DedupTrue", true, nestedBundle1, nestedBundleDedupOpts1},
			{"NestedBundle2_DedupFalse", false, nestedBundle2, nestedBundleOpts2},
			{"NestedBundle2_DedupTrue", true, nestedBundle2, nestedBundleDedupOpts2},
			{"NestedBundle3_DedupFalse", false, nestedBundle3, nestedBundleOpts3},
			{"NestedBundle3_DedupTrue", true, nestedBundle3, nestedBundleDedupOpts3},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				options, err := tc.bundle.Unbundle(tc.dedup)
				testhelpers.RequireNil(t, err, "got non-nill error from unbundle: %s", err)

				if len(options) != len(tc.expectedOpts) {
					t.Errorf("options length does not match expected length. got %d expected %d", len(options),
						len(tc.expectedOpts))
				} else {
					for i, opt := range options {
						if !reflect.DeepEqual(opt, tc.expectedOpts[i]) {
							t.Errorf("expected: %s\nreceived: %s", opt, tc.expectedOpts[i])
						}
					}
				}
			})
		}
	})
}
