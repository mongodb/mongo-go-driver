// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package transactionopt

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

func createNestedBundle1(t *testing.T) *TransactionBundle {
	nested := BundleTransaction(ReadPreference(rpSecondary), ReadConcern(rcLocal), WriteConcern(wc1))
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	outer := BundleTransaction(ReadPreference(rpPrimary), WriteConcern(wc2), nested)
	testhelpers.RequireNotNil(t, nested, "nested bundle was nil")

	return outer
}

func createdNestedBundle2(t *testing.T) *TransactionBundle {
	b1 := BundleTransaction(WriteConcern(wc1))
	testhelpers.RequireNotNil(t, b1, "b1 was nil")

	b2 := BundleTransaction(ReadPreference(rpPrimary), b1, ReadConcern(rcMajority))
	testhelpers.RequireNotNil(t, b2, "b2 was nil")

	outer := BundleTransaction(ReadConcern(rcLocal), b2, WriteConcern(wc2))
	testhelpers.RequireNotNil(t, outer, "outer was nil")

	return outer
}

func TestTransactionOpt(t *testing.T) {
	nilBundle := BundleTransaction()
	var nilOpts []session.ClientOptioner

	var bundle1 *TransactionBundle
	bundle1 = bundle1.WriteConcern(wc1).ReadConcern(rcLocal).WriteConcern(wc2)
	testhelpers.RequireNotNil(t, bundle1, "created bundle was nil")
	bundle1Opts := []session.ClientOptioner{
		WriteConcern(wc1).ConvertTransactionOption(),
		ReadConcern(rcLocal).ConvertTransactionOption(),
		WriteConcern(wc2).ConvertTransactionOption(),
	}

	bundle1DedupOpts := []session.ClientOptioner{
		ReadConcern(rcLocal).ConvertTransactionOption(),
		WriteConcern(wc2).ConvertTransactionOption(),
	}

	bundle2 := BundleTransaction(ReadPreference(rpPrimary), ReadConcern(rcMajority), WriteConcern(wc2), ReadPreference(rpSecondary))
	bundle2Opts := []session.ClientOptioner{
		ReadPreference(rpPrimary).ConvertTransactionOption(),
		ReadConcern(rcMajority).ConvertTransactionOption(),
		WriteConcern(wc2).ConvertTransactionOption(),
		ReadPreference(rpSecondary).ConvertTransactionOption(),
	}
	bundle2DedupOpts := []session.ClientOptioner{
		ReadConcern(rcMajority).ConvertTransactionOption(),
		WriteConcern(wc2).ConvertTransactionOption(),
		ReadPreference(rpSecondary).ConvertTransactionOption(),
	}

	nested1 := createNestedBundle1(t)
	nested1Opts := []session.ClientOptioner{
		ReadPreference(rpPrimary).ConvertTransactionOption(),
		WriteConcern(wc2).ConvertTransactionOption(),
		ReadPreference(rpSecondary).ConvertTransactionOption(),
		ReadConcern(rcLocal).ConvertTransactionOption(),
		WriteConcern(wc1).ConvertTransactionOption(),
	}
	nested1DedupOpts := []session.ClientOptioner{
		ReadPreference(rpSecondary).ConvertTransactionOption(),
		ReadConcern(rcLocal).ConvertTransactionOption(),
		WriteConcern(wc1).ConvertTransactionOption(),
	}

	nested2 := createdNestedBundle2(t)
	nested2Opts := []session.ClientOptioner{
		ReadConcern(rcLocal).ConvertTransactionOption(),
		ReadPreference(rpPrimary).ConvertTransactionOption(),
		WriteConcern(wc1).ConvertTransactionOption(),
		ReadConcern(rcMajority).ConvertTransactionOption(),
		WriteConcern(wc2).ConvertTransactionOption(),
	}
	nested2DedupOpts := []session.ClientOptioner{
		ReadPreference(rpPrimary).ConvertTransactionOption(),
		ReadConcern(rcMajority).ConvertTransactionOption(),
		WriteConcern(wc2).ConvertTransactionOption(),
	}

	t.Run("Unbundle", func(t *testing.T) {
		var cases = []struct {
			name         string
			bundle       *TransactionBundle
			expectedOpts []session.ClientOptioner
			dedup        bool
		}{
			{"NilBundle", nilBundle, nilOpts, false},
			{"NilBundle", nilBundle, nilOpts, true},
			{"Bundle1", bundle1, bundle1Opts, false},
			{"Bundle1Dedup", bundle1, bundle1DedupOpts, true},
			{"Bundle2", bundle2, bundle2Opts, false},
			{"Bundle2Dedup", bundle2, bundle2DedupOpts, true},
			{"Nested1", nested1, nested1Opts, false},
			{"Nested1Dedup", nested1, nested1DedupOpts, true},
			{"Nested2", nested2, nested2Opts, false},
			{"Nested2Dedup", nested2, nested2DedupOpts, true},
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
