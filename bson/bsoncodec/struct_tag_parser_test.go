// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncodec

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestStructTagParsers(t *testing.T) {
	testCases := []struct {
		name   string
		sf     reflect.StructField
		want   StructTags
		parser StructTagParserFunc
	}{
		{
			"default no bson tag",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag("bar")},
			StructTags{Name: "bar"},
			DefaultStructTagParser,
		},
		{
			"default empty",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag("")},
			StructTags{Name: "foo"},
			DefaultStructTagParser,
		},
		{
			"default tag only dash",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag("-")},
			StructTags{Skip: true},
			DefaultStructTagParser,
		},
		{
			"default bson tag only dash",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag(`bson:"-"`)},
			StructTags{Skip: true},
			DefaultStructTagParser,
		},
		{
			"default all options",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag(`bar,omitempty,minsize,truncate,inline`)},
			StructTags{Name: "bar", OmitEmpty: true, MinSize: true, Truncate: true, Inline: true},
			DefaultStructTagParser,
		},
		{
			"default all options default name",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag(`,omitempty,minsize,truncate,inline`)},
			StructTags{Name: "foo", OmitEmpty: true, MinSize: true, Truncate: true, Inline: true},
			DefaultStructTagParser,
		},
		{
			"default bson tag all options",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag(`bson:"bar,omitempty,minsize,truncate,inline"`)},
			StructTags{Name: "bar", OmitEmpty: true, MinSize: true, Truncate: true, Inline: true},
			DefaultStructTagParser,
		},
		{
			"default bson tag all options default name",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag(`bson:",omitempty,minsize,truncate,inline"`)},
			StructTags{Name: "foo", OmitEmpty: true, MinSize: true, Truncate: true, Inline: true},
			DefaultStructTagParser,
		},
		{
			"default ignore xml",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag(`xml:"bar"`)},
			StructTags{Name: "foo"},
			DefaultStructTagParser,
		},
		{
			"JSONFallback no bson tag",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag("bar")},
			StructTags{Name: "bar"},
			JSONFallbackStructTagParser,
		},
		{
			"JSONFallback empty",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag("")},
			StructTags{Name: "foo"},
			JSONFallbackStructTagParser,
		},
		{
			"JSONFallback tag only dash",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag("-")},
			StructTags{Skip: true},
			JSONFallbackStructTagParser,
		},
		{
			"JSONFallback bson tag only dash",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag(`bson:"-"`)},
			StructTags{Skip: true},
			JSONFallbackStructTagParser,
		},
		{
			"JSONFallback all options",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag(`bar,omitempty,minsize,truncate,inline`)},
			StructTags{Name: "bar", OmitEmpty: true, MinSize: true, Truncate: true, Inline: true},
			JSONFallbackStructTagParser,
		},
		{
			"JSONFallback all options default name",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag(`,omitempty,minsize,truncate,inline`)},
			StructTags{Name: "foo", OmitEmpty: true, MinSize: true, Truncate: true, Inline: true},
			JSONFallbackStructTagParser,
		},
		{
			"JSONFallback bson tag all options",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag(`bson:"bar,omitempty,minsize,truncate,inline"`)},
			StructTags{Name: "bar", OmitEmpty: true, MinSize: true, Truncate: true, Inline: true},
			JSONFallbackStructTagParser,
		},
		{
			"JSONFallback bson tag all options default name",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag(`bson:",omitempty,minsize,truncate,inline"`)},
			StructTags{Name: "foo", OmitEmpty: true, MinSize: true, Truncate: true, Inline: true},
			JSONFallbackStructTagParser,
		},
		{
			"JSONFallback json tag all options",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag(`json:"bar,omitempty,minsize,truncate,inline"`)},
			StructTags{Name: "bar", OmitEmpty: true, MinSize: true, Truncate: true, Inline: true},
			JSONFallbackStructTagParser,
		},
		{
			"JSONFallback json tag all options default name",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag(`json:",omitempty,minsize,truncate,inline"`)},
			StructTags{Name: "foo", OmitEmpty: true, MinSize: true, Truncate: true, Inline: true},
			JSONFallbackStructTagParser,
		},
		{
			"JSONFallback bson tag overrides other tags",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag(`bson:"bar" json:"qux,truncate"`)},
			StructTags{Name: "bar"},
			JSONFallbackStructTagParser,
		},
		{
			"JSONFallback ignore xml",
			reflect.StructField{Name: "foo", Tag: reflect.StructTag(`xml:"bar"`)},
			StructTags{Name: "foo"},
			JSONFallbackStructTagParser,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.parser(tc.sf)
			noerr(t, err)
			if !cmp.Equal(got, tc.want) {
				t.Errorf("Returned struct tags do not match. got %#v; want %#v", got, tc.want)
			}
		})
	}
}
