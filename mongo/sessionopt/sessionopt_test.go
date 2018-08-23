// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package sessionopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/core/readconcern"
	"github.com/mongodb/mongo-go-driver/core/readpref"
	"github.com/mongodb/mongo-go-driver/core/session"
	"github.com/mongodb/mongo-go-driver/core/writeconcern"
	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
)

var wc1 = writeconcern.New(writeconcern.WMajority())
var wc2 = writeconcern.New(writeconcern.W(5))

var rpPrimary = readpref.Primary()
var rpSecondary = readpref.Secondary()

var rcMajority = readconcern.Majority()
var rcLocal = readconcern.Local()

func createNestedBundle1(t *testing.T) *SessionBundle {
	nested := BundleSession(DefaultReadPreference(rpSecondary), DefaultReadConcern(rcLocal))
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	outer := BundleSession(CausalConsistency(false), CausalConsistency(true), DefaultReadPreference(rpPrimary), nested)
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	return outer
}

func createdNestedBundle2(t *testing.T) *SessionBundle {
	b1 := BundleSession(DefaultWriteConcern(wc1))
	testhelpers.RequireNotNil(t, b1, "b1 was nil")

	b2 := BundleSession(b1, DefaultReadConcern(rcMajority))
	testhelpers.RequireNotNil(t, b2, "b2 was nil")

	outer := BundleSession(CausalConsistency(true), CausalConsistency(false), b2, DefaultWriteConcern(wc2))
	testhelpers.RequireNotNil(t, outer, "outer was nil")

	return outer
}

func TestSessionOpt(t *testing.T) {
	nilBundle := BundleSession()
	var nilOpts []session.ClientOptioner

	var bundle1 *SessionBundle
	bundle1 = bundle1.CausalConsistency(true).CausalConsistency(false).DefaultWriteConcern(wc1)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []session.ClientOptioner{
		CausalConsistency(true).ConvertSessionOption(),
		CausalConsistency(false).ConvertSessionOption(),
		DefaultWriteConcern(wc1).ConvertSessionOption(),
	}

	bundle1DedupOpts := []session.ClientOptioner{
		CausalConsistency(false).ConvertSessionOption(),
		DefaultWriteConcern(wc1).ConvertSessionOption(),
	}

	bundle2 := BundleSession(CausalConsistency(true), DefaultReadPreference(rpPrimary), DefaultReadConcern(rcMajority))
	bundle2Opts := []session.ClientOptioner{
		CausalConsistency(true).ConvertSessionOption(),
		DefaultReadPreference(rpPrimary).ConvertSessionOption(),
		DefaultReadConcern(rcMajority).ConvertSessionOption(),
	}

	nested1 := createNestedBundle1(t)
	nested1Opts := []session.ClientOptioner{
		CausalConsistency(false).ConvertSessionOption(),
		CausalConsistency(true).ConvertSessionOption(),
		DefaultReadPreference(rpPrimary).ConvertSessionOption(),
		DefaultReadPreference(rpSecondary).ConvertSessionOption(),
		DefaultReadConcern(rcLocal).ConvertSessionOption(),
	}
	nested1DedupOpts := []session.ClientOptioner{
		CausalConsistency(true).ConvertSessionOption(),
		DefaultReadPreference(rpSecondary).ConvertSessionOption(),
		DefaultReadConcern(rcLocal).ConvertSessionOption(),
	}

	nested2 := createdNestedBundle2(t)
	nested2Opts := []session.ClientOptioner{
		CausalConsistency(true).ConvertSessionOption(),
		CausalConsistency(false).ConvertSessionOption(),
		DefaultWriteConcern(wc1).ConvertSessionOption(),
		DefaultReadConcern(rcMajority).ConvertSessionOption(),
		DefaultWriteConcern(wc2).ConvertSessionOption(),
	}
	nested2DedupOpts := []session.ClientOptioner{
		CausalConsistency(false).ConvertSessionOption(),
		DefaultReadConcern(rcMajority).ConvertSessionOption(),
		DefaultWriteConcern(wc2).ConvertSessionOption(),
	}

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name         string
			bundle       *SessionBundle
			expectedOpts []session.ClientOptioner
			dedup        bool
		}{
			{"NilBundle", nilBundle, nilOpts, false},
			{"NilBundle", nilBundle, nilOpts, true},
			{"Bundle1", bundle1, bundle1Opts, false},
			{"Bundle1Dedup", bundle1, bundle1DedupOpts, true},
			{"Bundle2", bundle2, bundle2Opts, false},
			{"Bundle2", bundle2, bundle2Opts, true},
			{"Nested1", nested1, nested1Opts, false},
			{"Nested1", nested1, nested1DedupOpts, true},
			{"Nested2", nested2, nested2Opts, false},
			{"Nested2", nested2, nested2DedupOpts, true},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				opts, err := tc.bundle.Unbundle(tc.dedup)
				testhelpers.RequireNil(t, err, "error unbundling: %s", err)

				if len(opts) != len(tc.expectedOpts) {
					t.Fatalf("options length mismatch; expected %d got %d", len(tc.expectedOpts), len(opts))
				}

				for i, opt := range opts {
					if !reflect.DeepEqual(opt, tc.expectedOpts[i]) {
						t.Fatalf("expected opt %s got %s", tc.expectedOpts[i], opt)
					}
				}
			})
		}
	})
}
