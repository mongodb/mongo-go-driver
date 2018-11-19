// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsonrw

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
)

func TestExtJSONReader(t *testing.T) {
	t.Run("ReadDocument", func(t *testing.T) {
		t.Run("EmbeddedDocument", func(t *testing.T) {
			ejvr := &extJSONValueReader{
				stack: []ejvrState{
					{mode: mTopLevel},
					{mode: mElement, vType: bsontype.Boolean},
				},
				frame: 1,
			}

			ejvr.stack[1].mode = mArray
			wanterr := ejvr.invalidTransitionErr(mDocument, "ReadDocument", []mode{mTopLevel, mElement, mValue})
			_, err := ejvr.ReadDocument()
			if err == nil || err.Error() != wanterr.Error() {
				t.Errorf("Incorrect returned error. got %v; want %v", err, wanterr)
			}

		})
	})

	t.Run("invalid transition", func(t *testing.T) {
		t.Run("Skip", func(t *testing.T) {
			ejvr := &extJSONValueReader{stack: []ejvrState{{mode: mTopLevel}}}
			wanterr := (&extJSONValueReader{stack: []ejvrState{{mode: mTopLevel}}}).invalidTransitionErr(0, "Skip", []mode{mElement, mValue})
			goterr := ejvr.Skip()
			if !cmp.Equal(goterr, wanterr, cmp.Comparer(compareErrors)) {
				t.Errorf("Expected correct invalid transition error. got %v; want %v", goterr, wanterr)
			}
		})
	})
}
