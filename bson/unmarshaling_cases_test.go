// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"reflect"

	"go.mongodb.org/mongo-driver/bson/bsoncodec"
)

type unmarshalingTestCase struct {
	name  string
	reg   *bsoncodec.Registry
	sType reflect.Type
	want  interface{}
	data  []byte
}

var unmarshalingTestCases = []unmarshalingTestCase{
	{
		"small struct",
		nil,
		reflect.TypeOf(struct {
			Foo bool
		}{}),
		&struct {
			Foo bool
		}{Foo: true},
		docToBytes(D{{"foo", true}}),
	},
	{
		"nested document",
		nil,
		reflect.TypeOf(struct {
			Foo struct {
				Bar bool
			}
		}{}),
		&struct {
			Foo struct {
				Bar bool
			}
		}{
			Foo: struct {
				Bar bool
			}{Bar: true},
		},
		docToBytes(D{{"foo", D{{"bar", true}}}}),
	},
	{
		"simple array",
		nil,
		reflect.TypeOf(struct {
			Foo []bool
		}{}),
		&struct {
			Foo []bool
		}{
			Foo: []bool{true},
		},
		docToBytes(D{{"foo", A{true}}}),
	},
	{
		"struct with mixed case fields",
		nil,
		reflect.TypeOf(struct {
			FooBar int32
		}{}),
		&struct {
			FooBar int32
		}{
			FooBar: 10,
		},
		docToBytes(D{{"fooBar", int32(10)}}),
	},
}

var unmarshalingExtTestCases = []unmarshalingTestCase{
	{
		name: "Small struct",
		reg:  DefaultRegistry,
		sType: reflect.TypeOf(struct {
			Foo int
		}{}),
		data: []byte(`{"foo":1}`),
		want: &struct {
			Foo int
		}{Foo: 1},
	},
	{
		name: "Valid surrogate pair",
		reg:  DefaultRegistry,
		sType: reflect.TypeOf(struct {
			Foo string
		}{}),
		data: []byte(`{"foo":"\uD834\uDd1e"}`),
		want: &struct {
			Foo string
		}{Foo: "𝄞"},
	},
	{
		name: "Valid surrogate pair with other values",
		reg:  DefaultRegistry,
		sType: reflect.TypeOf(struct {
			Foo string
		}{}),
		data: []byte(`{"foo":"abc \uD834\uDd1e 123"}`),
		want: &struct {
			Foo string
		}{Foo: "abc 𝄞 123"},
	},
	{
		name: "High surrogate value with no following low surrogate value",
		reg:  DefaultRegistry,
		sType: reflect.TypeOf(struct {
			Foo string
		}{}),
		data: []byte(`{"foo":"abc \uD834 123"}`),
		want: &struct {
			Foo string
		}{Foo: "abc � 123"},
	},
	{
		name: "High surrogate value at end of string",
		reg:  DefaultRegistry,
		sType: reflect.TypeOf(struct {
			Foo string
		}{}),
		data: []byte(`{"foo":"\uD834"}`),
		want: &struct {
			Foo string
		}{Foo: "�"},
	},
	{
		name: "Low surrogate value with no preceeding high surrogate value",
		reg:  DefaultRegistry,
		sType: reflect.TypeOf(struct {
			Foo string
		}{}),
		data: []byte(`{"foo":"abc \uDd1e 123"}`),
		want: &struct {
			Foo string
		}{Foo: "abc � 123"},
	},
	{
		name: "Low surrogate value at end of string",
		reg:  DefaultRegistry,
		sType: reflect.TypeOf(struct {
			Foo string
		}{}),
		data: []byte(`{"foo":"\uDd1e"}`),
		want: &struct {
			Foo string
		}{Foo: "�"},
	},
	{
		name: "High surrogate value with non-surrogate unicode value",
		reg:  DefaultRegistry,
		sType: reflect.TypeOf(struct {
			Foo string
		}{}),
		data: []byte(`{"foo":"\uD834\u00BF"}`),
		want: &struct {
			Foo string
		}{Foo: "�¿"},
	},
}
