// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"reflect"

	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
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
		docToBytes(bsonx.Doc{{"foo", bsonx.Boolean(true)}}),
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
		docToBytes(bsonx.Doc{{"foo", bsonx.Document(bsonx.Doc{{"bar", bsonx.Boolean(true)}})}}),
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
		docToBytes(bsonx.Doc{{"foo", bsonx.Array(bsonx.Arr{bsonx.Boolean(true)})}}),
	},
}
