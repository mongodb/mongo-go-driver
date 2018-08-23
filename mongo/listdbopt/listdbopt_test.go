// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package listdbopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/core/option"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
)

func createNestedListDatabasesBundle1(t *testing.T) *ListDatabasesBundle {
	nestedBundle := BundleListDatabases(NameOnly(false))
	testhelpers.RequireNotNil(t, nestedBundle, "nested bundle was nil")

	outerBundle := BundleListDatabases(NameOnly(true), NameOnly(true), nestedBundle)
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test doubly nested bundle
func createNestedListDatabasesBundle2(t *testing.T) *ListDatabasesBundle {
	b1 := BundleListDatabases(NameOnly(false))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleListDatabases(NameOnly(false), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	outerBundle := BundleListDatabases(NameOnly(true), NameOnly(true), b2)
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

// Test two top level nested bundles
func createNestedListDatabasesBundle3(t *testing.T) *ListDatabasesBundle {
	b1 := BundleListDatabases(NameOnly(false))
	testhelpers.RequireNotNil(t, b1, "nested bundle was nil")

	b2 := BundleListDatabases(NameOnly(false), b1)
	testhelpers.RequireNotNil(t, b2, "nested bundle was nil")

	b3 := BundleListDatabases(NameOnly(true))
	testhelpers.RequireNotNil(t, b3, "nested bundle was nil")

	b4 := BundleListDatabases(NameOnly(false), b3)
	testhelpers.RequireNotNil(t, b4, "nested bundle was nil")

	outerBundle := BundleListDatabases(b4, NameOnly(true), b2)
	testhelpers.RequireNotNil(t, outerBundle, "outer bundle was nil")

	return outerBundle
}

func TestListDatabasesOpt(t *testing.T) {
	var bundle1 *ListDatabasesBundle
	bundle1 = bundle1.NameOnly(true).NameOnly(false)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []option.Optioner{
		NameOnly(true).ConvertListDatabasesOption(),
		NameOnly(false).ConvertListDatabasesOption(),
	}
	bundle1DedupOpts := []option.Optioner{
		NameOnly(false).ConvertListDatabasesOption(),
	}

	bundle2 := BundleListDatabases(NameOnly(true))
	bundle2Opts := []option.Optioner{
		NameOnly(true).ConvertListDatabasesOption(),
	}

	bundle3 := BundleListDatabases().
		NameOnly(false).
		NameOnly(true)

	bundle3Opts := []option.Optioner{
		OptNameOnly(false).ConvertListDatabasesOption(),
		OptNameOnly(true).ConvertListDatabasesOption(),
	}

	bundle3DedupOpts := []option.Optioner{
		OptNameOnly(true).ConvertListDatabasesOption(),
	}

	nilBundle := BundleListDatabases()
	var nilBundleOpts []option.Optioner

	nestedBundle1 := createNestedListDatabasesBundle1(t)
	nestedBundleOpts1 := []option.Optioner{
		OptNameOnly(true).ConvertListDatabasesOption(),
		OptNameOnly(true).ConvertListDatabasesOption(),
		OptNameOnly(false).ConvertListDatabasesOption(),
	}
	nestedBundleDedupOpts1 := []option.Optioner{
		OptNameOnly(false).ConvertListDatabasesOption(),
	}

	nestedBundle2 := createNestedListDatabasesBundle2(t)
	nestedBundleOpts2 := []option.Optioner{
		OptNameOnly(true).ConvertListDatabasesOption(),
		OptNameOnly(true).ConvertListDatabasesOption(),
		OptNameOnly(false).ConvertListDatabasesOption(),
		OptNameOnly(false).ConvertListDatabasesOption(),
	}
	nestedBundleDedupOpts2 := []option.Optioner{
		OptNameOnly(false).ConvertListDatabasesOption(),
	}

	nestedBundle3 := createNestedListDatabasesBundle3(t)
	nestedBundleOpts3 := []option.Optioner{
		OptNameOnly(false).ConvertListDatabasesOption(),
		OptNameOnly(true).ConvertListDatabasesOption(),
		OptNameOnly(true).ConvertListDatabasesOption(),
		OptNameOnly(false).ConvertListDatabasesOption(),
		OptNameOnly(false).ConvertListDatabasesOption(),
	}
	nestedBundleDedupOpts3 := []option.Optioner{
		OptNameOnly(false).ConvertListDatabasesOption(),
	}

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

	t.Run("TestAll", func(t *testing.T) {
		opts := []ListDatabasesOption{
			NameOnly(true),
		}
		params := make([]ListDatabases, len(opts))
		for i := range opts {
			params[i] = opts[i]
		}
		bundle := BundleListDatabases(params...)

		deleteOpts, _, err := bundle.Unbundle(true)
		testhelpers.RequireNil(t, err, "got non-nill error from unbundle: %s", err)

		if len(deleteOpts) != len(opts) {
			t.Errorf("expected unbundled opts len %d. got %d", len(opts), len(deleteOpts))
		}

		for i, opt := range opts {
			if !reflect.DeepEqual(opt.ConvertListDatabasesOption(), deleteOpts[i]) {
				t.Errorf("opt mismatch. expected %#v, got %#v", opt, deleteOpts[i])
			}
		}
	})

	t.Run("Nil Option Bundle", func(t *testing.T) {
		sess := ListDatabasesSessionOpt{}
		opts, _, err := BundleListDatabases(NameOnly(true), BundleListDatabases(nil), sess, nil).unbundle()
		testhelpers.RequireNil(t, err, "got non-nil error from unbundle: %s", err)

		if len(opts) != 1 {
			t.Errorf("expected bundle length 1. got: %d", len(opts))
		}

		opts, _, err = BundleListDatabases(nil, sess, BundleListDatabases(nil), NameOnly(true)).unbundle()
		testhelpers.RequireNil(t, err, "got non-nil error from unbundle: %s", err)

		if len(opts) != 1 {
			t.Errorf("expected bundle length 1. got: %d", len(opts))
		}
	})

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name         string
			dedup        bool
			bundle       *ListDatabasesBundle
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
				options, _, err := tc.bundle.Unbundle(tc.dedup)
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
