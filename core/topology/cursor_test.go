// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package topology

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/stretchr/testify/assert"
)

func TestCursorNextDoesNotPanicIfContextisNil(t *testing.T) {
	// all collection/cursor iterators should take contexts, but
	// permit passing nils for contexts, which should not
	// panic.
	//
	// While more through testing might be ideal this check
	// prevents a regression of GODRIVER-298

	c := cursor{batch: bson.NewArray(bson.VC.String("a"), bson.VC.String("b"))}

	var iterNext bool
	assert.NotPanics(t, func() {
		iterNext = c.Next(nil)
	})
	assert.True(t, iterNext)
}
