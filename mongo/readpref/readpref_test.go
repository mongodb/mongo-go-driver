// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package readpref

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/tag"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func TestPrimary(t *testing.T) {
	require := require.New(t)
	subject := Primary()

	require.Equal(PrimaryMode, subject.Mode())
	_, set := subject.MaxStaleness()
	require.False(set)
	require.Empty(subject.TagSets())
}

func TestPrimaryPreferred(t *testing.T) {
	require := require.New(t)
	subject := PrimaryPreferred()

	require.Equal(PrimaryPreferredMode, subject.Mode())
	_, set := subject.MaxStaleness()
	require.False(set)
	require.Empty(subject.TagSets())
}

func TestPrimaryPreferred_with_options(t *testing.T) {
	require := require.New(t)
	subject := PrimaryPreferred(
		WithMaxStaleness(time.Duration(10)),
		WithTags("a", "1", "b", "2"),
	)

	require.Equal(PrimaryPreferredMode, subject.Mode())
	ms, set := subject.MaxStaleness()
	require.True(set)
	require.Equal(time.Duration(10), ms)
	require.Equal([]tag.Set{{tag.Tag{Name: "a", Value: "1"}, tag.Tag{Name: "b", Value: "2"}}}, subject.TagSets())
}

func TestSecondaryPreferred(t *testing.T) {
	require := require.New(t)
	subject := SecondaryPreferred()

	require.Equal(SecondaryPreferredMode, subject.Mode())
	_, set := subject.MaxStaleness()
	require.False(set)
	require.Empty(subject.TagSets())
}

func TestSecondaryPreferred_with_options(t *testing.T) {
	require := require.New(t)
	subject := SecondaryPreferred(
		WithMaxStaleness(time.Duration(10)),
		WithTags("a", "1", "b", "2"),
	)

	require.Equal(SecondaryPreferredMode, subject.Mode())
	ms, set := subject.MaxStaleness()
	require.True(set)
	require.Equal(time.Duration(10), ms)
	require.Equal([]tag.Set{{tag.Tag{Name: "a", Value: "1"}, tag.Tag{Name: "b", Value: "2"}}}, subject.TagSets())
}

func TestSecondary(t *testing.T) {
	require := require.New(t)
	subject := Secondary()

	require.Equal(SecondaryMode, subject.Mode())
	_, set := subject.MaxStaleness()
	require.False(set)
	require.Empty(subject.TagSets())
}

func TestSecondary_with_options(t *testing.T) {
	require := require.New(t)
	subject := Secondary(
		WithMaxStaleness(time.Duration(10)),
		WithTags("a", "1", "b", "2"),
	)

	require.Equal(SecondaryMode, subject.Mode())
	ms, set := subject.MaxStaleness()
	require.True(set)
	require.Equal(time.Duration(10), ms)
	require.Equal([]tag.Set{{tag.Tag{Name: "a", Value: "1"}, tag.Tag{Name: "b", Value: "2"}}}, subject.TagSets())
}

func TestNearest(t *testing.T) {
	require := require.New(t)
	subject := Nearest()

	require.Equal(NearestMode, subject.Mode())
	_, set := subject.MaxStaleness()
	require.False(set)
	require.Empty(subject.TagSets())
}

func TestNearest_with_options(t *testing.T) {
	require := require.New(t)
	subject := Nearest(
		WithMaxStaleness(time.Duration(10)),
		WithTags("a", "1", "b", "2"),
	)

	require.Equal(NearestMode, subject.Mode())
	ms, set := subject.MaxStaleness()
	require.True(set)
	require.Equal(time.Duration(10), ms)
	require.Equal([]tag.Set{{tag.Tag{Name: "a", Value: "1"}, tag.Tag{Name: "b", Value: "2"}}}, subject.TagSets())
}

func TestHedge(t *testing.T) {
	t.Run("hedge specified with primary mode errors", func(t *testing.T) {
		_, err := New(PrimaryMode, WithHedgeEnabled(true))
		assert.Equal(t, errInvalidReadPreference, err, "expected error %v, got %v", errInvalidReadPreference, err)
	})
	t.Run("valid hedge document and mode succeeds", func(t *testing.T) {
		rp, err := New(SecondaryMode, WithHedgeEnabled(true))
		assert.Nil(t, err, "expected no error, got %v", err)

		transformedDoc := bsoncore.Document(bsoncore.BuildDocumentFromElements(
			nil,
			bsoncore.AppendBooleanElement(nil, "enabled", true),
		))
		assert.True(t, bytes.Equal(transformedDoc, rp.Hedge()), "expected stored hedge document %v, got %v",
			transformedDoc, rp.Hedge())
	})
}
