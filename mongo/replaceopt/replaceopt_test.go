// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package replaceopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
	"github.com/mongodb/mongo-go-driver/mongo/mongoopt"
)

var collation = &mongoopt.Collation{}

func createNestedReplaceBundle1(t *testing.T) *ReplaceBundle {
	nestedBundle := BundleReplace(Upsert(false))
	testhelpers.RequireNotNil(t, nestedBundle, "nested bundle was nil")

	outerBundle := BundleReplace(Upsert(true), BypassDocumentValidation(true), nestedBundle, BypassDocumentValidation(false))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test doubly nested bundle
func createNestedReplaceBundle2(t *testing.T) *ReplaceBundle {
	b1 := BundleReplace(Upsert(false))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleReplace(Collation(collation), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	outerBundle := BundleReplace(Upsert(true), BypassDocumentValidation(true), b2, BypassDocumentValidation(false))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test two top level nested bundles
func createNestedReplaceBundle3(t *testing.T) *ReplaceBundle {
	b1 := BundleReplace(Upsert(false))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleReplace(Collation(collation), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	b3 := BundleReplace(Upsert(true))
	testhelpers.RequireNotNil(t, b3, "nested bundle was nil")

	b4 := BundleReplace(Collation(collation), b3)
	testhelpers.RequireNotNil(t, b4, "nested bundle was nil")

	outerBundle := BundleReplace(b4, BypassDocumentValidation(true), b2, BypassDocumentValidation(false))
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

func TestReplaceOpt(t *testing.T) {
	var bundle1 *ReplaceBundle
	bundle1 = bundle1.Upsert(true).BypassDocumentValidation(false)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []option.Optioner{
		OptUpsert(true).ConvertReplaceOption(),
		OptBypassDocumentValidation(false).ConvertReplaceOption(),
	}
	bundle1DedupOpts := []option.Optioner{
		OptUpsert(true).ConvertReplaceOption(),
		OptBypassDocumentValidation(false).ConvertReplaceOption(),
	}

	bundle2 := BundleReplace(Collation(collation))
	bundle2Opts := []option.Optioner{
		OptCollation{collation.Convert()}.ConvertReplaceOption(),
	}

	bundle3 := BundleReplace().
		Collation(collation).
		BypassDocumentValidation(true).
		Upsert(false).
		Upsert(true)

	bundle3Opts := []option.Optioner{
		OptCollation{collation.Convert()}.ConvertReplaceOption(),
		OptBypassDocumentValidation(true).ConvertReplaceOption(),
		OptUpsert(false).ConvertReplaceOption(),
		OptUpsert(true).ConvertReplaceOption(),
	}

	bundle3DedupOpts := []option.Optioner{
		OptCollation{collation.Convert()}.ConvertReplaceOption(),
		OptBypassDocumentValidation(true).ConvertReplaceOption(),
		OptUpsert(true).ConvertReplaceOption(),
	}

	nilBundle := BundleReplace()
	var nilBundleOpts []option.Optioner

	nestedBundle1 := createNestedReplaceBundle1(t)
	nestedBundleOpts1 := []option.Optioner{
		OptUpsert(true).ConvertReplaceOption(),
		OptBypassDocumentValidation(true).ConvertReplaceOption(),
		OptUpsert(false).ConvertReplaceOption(),
		OptBypassDocumentValidation(false).ConvertReplaceOption(),
	}
	nestedBundleDedupOpts1 := []option.Optioner{
		OptUpsert(false).ConvertReplaceOption(),
		OptBypassDocumentValidation(false).ConvertReplaceOption(),
	}

	nestedBundle2 := createNestedReplaceBundle2(t)
	nestedBundleOpts2 := []option.Optioner{
		OptUpsert(true).ConvertReplaceOption(),
		OptBypassDocumentValidation(true).ConvertReplaceOption(),
		OptCollation{collation.Convert()}.ConvertReplaceOption(),
		OptUpsert(false).ConvertReplaceOption(),
		OptBypassDocumentValidation(false).ConvertReplaceOption(),
	}
	nestedBundleDedupOpts2 := []option.Optioner{
		OptCollation{collation.Convert()}.ConvertReplaceOption(),
		OptUpsert(false).ConvertReplaceOption(),
		OptBypassDocumentValidation(false).ConvertReplaceOption(),
	}

	nestedBundle3 := createNestedReplaceBundle3(t)
	nestedBundleOpts3 := []option.Optioner{
		OptCollation{collation.Convert()}.ConvertReplaceOption(),
		OptUpsert(true).ConvertReplaceOption(),
		OptBypassDocumentValidation(true).ConvertReplaceOption(),
		OptCollation{collation.Convert()}.ConvertReplaceOption(),
		OptUpsert(false).ConvertReplaceOption(),
		OptBypassDocumentValidation(false).ConvertReplaceOption(),
	}
	nestedBundleDedupOpts3 := []option.Optioner{
		OptCollation{collation.Convert()}.ConvertReplaceOption(),
		OptUpsert(false).ConvertReplaceOption(),
		OptBypassDocumentValidation(false).ConvertReplaceOption(),
	}

	t.Run("TestAll", func(t *testing.T) {
		c := &mongoopt.Collation{
			Locale: "string locale",
		}

		opts := []Replace{
			BypassDocumentValidation(true),
			Collation(c),
			Upsert(false),
		}
		bundle := BundleReplace(opts...)

		deleteOpts, err := bundle.Unbundle(true)
		testhelpers.RequireNil(t, err, "got non-nill error from unbundle: %s", err)

		if len(deleteOpts) != len(opts) {
			t.Errorf("expected unbundled opts len %d. got %d", len(opts), len(deleteOpts))
		}

		for i, opt := range opts {
			if !reflect.DeepEqual(opt.ConvertReplaceOption(), deleteOpts[i]) {
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
			bundle       *ReplaceBundle
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
