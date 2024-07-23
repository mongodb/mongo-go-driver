// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package readpref

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/internal/assert"
)

func TestReadPref_String(t *testing.T) {
	t.Run("ReadPref.String() with all options", func(t *testing.T) {
		rpOpts := &Options{}

		maxStaleness := 120 * time.Second
		rpOpts.MaxStaleness = &maxStaleness

		rpOpts.TagSets = []TagSet{{{"a", "1"}, {"b", "2"}}, {{"q", "5"}, {"r", "6"}}}

		hedgeEnabled := true
		rpOpts.HedgeEnabled = &hedgeEnabled

		readPref, err := New(NearestMode, rpOpts)
		assert.NoError(t, err)

		expected := "nearest(maxStaleness=2m0s tagSet=a=1,b=2 tagSet=q=5,r=6 hedgeEnabled=true)"
		assert.Equal(t, expected, readPref.String(), "expected %q, got %q", expected, readPref.String())
	})
	t.Run("ReadPref.String() with one option", func(t *testing.T) {
		rpOpts := &Options{}

		tagSet, err := NewTagSet("a", "1", "b", "2")
		assert.NoError(t, err)

		rpOpts.TagSets = []TagSet{tagSet}

		readPref, err := New(SecondaryMode, rpOpts)
		assert.NoError(t, err)

		expected := "secondary(tagSet=a=1,b=2)"
		assert.Equal(t, expected, readPref.String(), "expected %q, got %q", expected, readPref.String())
	})
	t.Run("ReadPref.String() with no options", func(t *testing.T) {
		readPref := Primary()
		expected := "primary"
		assert.Equal(t, expected, readPref.String(), "expected %q, got %q", expected, readPref.String())
	})
}
