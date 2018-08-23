// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package dropcollopt

import (
	"testing"

	"reflect"

	"github.com/mongodb/mongo-go-driver/internal/testutil/helpers"
)

func TestDropCollOpt(t *testing.T) {

	t.Run("TestAll", func(t *testing.T) {
		opts := []DropCollOption{}
		params := make([]DropColl, len(opts))
		for i := range opts {
			params[i] = opts[i]
		}
		bundle := BundleDropColl(params...)

		deleteOpts, _, err := bundle.Unbundle(true)
		testhelpers.RequireNil(t, err, "got non-nill error from unbundle: %s", err)

		if len(deleteOpts) != len(opts) {
			t.Errorf("expected unbundled opts len %d. got %d", len(opts), len(deleteOpts))
		}

		for i, opt := range opts {
			if !reflect.DeepEqual(opt.ConvertDropCollOption(), deleteOpts[i]) {
				t.Errorf("opt mismatch. expected %#v, got %#v", opt, deleteOpts[i])
			}
		}
	})
}
