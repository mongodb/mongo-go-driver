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
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/tag"
)

func TestPrimary(t *testing.T) {
	subject := Primary()

	require.Equal(t, PrimaryMode, subject.Mode())
	_, set := subject.MaxStaleness()
	require.False(t, set)
	require.Len(t, subject.TagSets(), 0)
}

func TestPrimaryPreferred(t *testing.T) {
	subject := PrimaryPreferred()

	require.Equal(t, PrimaryPreferredMode, subject.Mode())
	_, set := subject.MaxStaleness()
	require.False(t, set)
	require.Len(t, subject.TagSets(), 0)
}

func TestPrimaryPreferred_with_options(t *testing.T) {
	subject := PrimaryPreferred(
		WithMaxStaleness(time.Duration(10)),
		WithTags("a", "1", "b", "2"),
	)

	require.Equal(t, PrimaryPreferredMode, subject.Mode())
	ms, set := subject.MaxStaleness()
	require.True(t, set)
	require.Equal(t, time.Duration(10), ms)
	require.Equal(t, []tag.Set{{tag.Tag{Name: "a", Value: "1"}, tag.Tag{Name: "b", Value: "2"}}}, subject.TagSets())
}

func TestSecondaryPreferred(t *testing.T) {
	subject := SecondaryPreferred()

	require.Equal(t, SecondaryPreferredMode, subject.Mode())
	_, set := subject.MaxStaleness()
	require.False(t, set)
	require.Len(t, subject.TagSets(), 0)
}

func TestSecondaryPreferred_with_options(t *testing.T) {
	subject := SecondaryPreferred(
		WithMaxStaleness(time.Duration(10)),
		WithTags("a", "1", "b", "2"),
	)

	require.Equal(t, SecondaryPreferredMode, subject.Mode())
	ms, set := subject.MaxStaleness()
	require.True(t, set)
	require.Equal(t, time.Duration(10), ms)
	require.Equal(t, []tag.Set{{tag.Tag{Name: "a", Value: "1"}, tag.Tag{Name: "b", Value: "2"}}}, subject.TagSets())
}

func TestSecondary(t *testing.T) {
	subject := Secondary()

	require.Equal(t, SecondaryMode, subject.Mode())
	_, set := subject.MaxStaleness()
	require.False(t, set)
	require.Len(t, subject.TagSets(), 0)
}

func TestSecondary_with_options(t *testing.T) {
	subject := Secondary(
		WithMaxStaleness(time.Duration(10)),
		WithTags("a", "1", "b", "2"),
	)

	require.Equal(t, SecondaryMode, subject.Mode())
	ms, set := subject.MaxStaleness()
	require.True(t, set)
	require.Equal(t, time.Duration(10), ms)
	require.Equal(t, []tag.Set{{tag.Tag{Name: "a", Value: "1"}, tag.Tag{Name: "b", Value: "2"}}}, subject.TagSets())
}

func TestNearest(t *testing.T) {
	subject := Nearest()

	require.Equal(t, NearestMode, subject.Mode())
	_, set := subject.MaxStaleness()
	require.False(t, set)
	require.Len(t, subject.TagSets(), 0)
}

func TestNearest_with_options(t *testing.T) {
	subject := Nearest(
		WithMaxStaleness(time.Duration(10)),
		WithTags("a", "1", "b", "2"),
	)

	require.Equal(t, NearestMode, subject.Mode())
	ms, set := subject.MaxStaleness()
	require.True(t, set)
	require.Equal(t, time.Duration(10), ms)
	require.Equal(t, []tag.Set{{tag.Tag{Name: "a", Value: "1"}, tag.Tag{Name: "b", Value: "2"}}}, subject.TagSets())
}

func TestHedge(t *testing.T) {
	t.Run("hedge specified with primary mode errors", func(t *testing.T) {
		_, err := New(PrimaryMode, WithHedgeEnabled(true))
		assert.Equal(t, errInvalidReadPreference, err, "expected error %v, got %v", errInvalidReadPreference, err)
	})
	t.Run("valid hedge document and mode succeeds", func(t *testing.T) {
		rp, err := New(SecondaryMode, WithHedgeEnabled(true))
		assert.Nil(t, err, "expected no error, got %v", err)
		enabled := rp.HedgeEnabled()
		assert.NotNil(t, enabled, "expected HedgeEnabled to return a non-nil value, got nil")
		assert.True(t, *enabled, "expected HedgeEnabled to return true, got false")
	})
}

func TestReadPref_String(t *testing.T) {
	t.Run("ReadPref.String() with all options", func(t *testing.T) {
		readPref := Nearest(
			WithMaxStaleness(120*time.Second),
			WithTagSets(tag.Set{{"a", "1"}, {"b", "2"}}, tag.Set{{"q", "5"}, {"r", "6"}}),
			WithHedgeEnabled(true),
		)
		expected := "nearest(maxStaleness=2m0s tagSet=a=1,b=2 tagSet=q=5,r=6 hedgeEnabled=true)"
		assert.Equal(t, expected, readPref.String(), "expected %q, got %q", expected, readPref.String())
	})
	t.Run("ReadPref.String() with one option", func(t *testing.T) {
		readPref := Secondary(WithTags("a", "1", "b", "2"))
		expected := "secondary(tagSet=a=1,b=2)"
		assert.Equal(t, expected, readPref.String(), "expected %q, got %q", expected, readPref.String())
	})
	t.Run("ReadPref.String() with no options", func(t *testing.T) {
		readPref := Primary()
		expected := "primary"
		assert.Equal(t, expected, readPref.String(), "expected %q, got %q", expected, readPref.String())
	})
}
