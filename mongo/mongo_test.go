// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package mongo

import (
	"errors"
	"fmt"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/internal/testutil/assert"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func TestMongoHelpers(t *testing.T) {
	t.Run("transform document", func(t *testing.T) {
		testCases := []struct {
			name     string
			document interface{}
			want     bsonx.Doc
			err      error
		}{
			{
				"bson.Marshaler",
				bMarsh{bsonx.Doc{{"foo", bsonx.String("bar")}}},
				bsonx.Doc{{"foo", bsonx.String("bar")}},
				nil,
			},
			{
				"reflection",
				reflectStruct{Foo: "bar"},
				bsonx.Doc{{"foo", bsonx.String("bar")}},
				nil,
			},
			{
				"reflection pointer",
				&reflectStruct{Foo: "bar"},
				bsonx.Doc{{"foo", bsonx.String("bar")}},
				nil,
			},
			{
				"unsupported type",
				[]string{"foo", "bar"},
				nil,
				MarshalError{
					Value: []string{"foo", "bar"},
					Err:   errors.New("WriteArray can only write a Array while positioned on a Element or Value but is positioned on a TopLevel")},
			},
			{
				"nil",
				nil,
				nil,
				ErrNilDocument,
			},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := transformDocument(bson.NewRegistryBuilder().Build(), tc.document)
				assert.Equal(t, tc.err, err, "expected error %v, got %v", tc.err, err)
				assert.Equal(t, tc.want, got, "expected document %v, got %v", tc.want, got)
			})
		}
	})
	t.Run("transform and ensure ID", func(t *testing.T) {
		t.Run("newly added _id should be first element", func(t *testing.T) {
			doc := bson.D{{"foo", "bar"}, {"baz", "qux"}, {"hello", "world"}}
			got, id, err := transformAndEnsureID(bson.DefaultRegistry, doc)
			assert.Nil(t, err, "transformAndEnsureID error: %v", err)
			oid, ok := id.(primitive.ObjectID)
			assert.True(t, ok, "expected returned id type %T, got %T", primitive.ObjectID{}, id)
			wantDoc := bson.D{
				{"_id", oid}, {"foo", "bar"},
				{"baz", "qux"}, {"hello", "world"},
			}
			_, wantBSON, _ := bson.MarshalValue(wantDoc)
			want := bsoncore.Document(wantBSON)
			assert.Equal(t, want, got, "expected document %v, got %v", want, got)
		})
		t.Run("existing _id as first element should remain first element", func(t *testing.T) {
			doc := bson.D{{"_id", 3.14159}, {"foo", "bar"}, {"baz", "qux"}, {"hello", "world"}}
			got, id, err := transformAndEnsureID(bson.DefaultRegistry, doc)
			assert.Nil(t, err, "transformAndEnsureID error: %v", err)
			_, ok := id.(float64)
			assert.True(t, ok, "expected returned id type %T, got %T", float64(0), id)
			wantDoc := bson.D{
				{"_id", 3.14159}, {"foo", "bar"},
				{"baz", "qux"}, {"hello", "world"},
			}
			_, wantBSON, _ := bson.MarshalValue(wantDoc)
			want := bsoncore.Document(wantBSON)
			assert.Equal(t, want, got, "expected document %v, got %v", want, got)
		})
		t.Run("existing _id should not overwrite a first binary field", func(t *testing.T) {
			doc := bson.D{{"bin", []byte{0, 0, 0}}, {"_id", "LongEnoughIdentifier"}}
			got, id, err := transformAndEnsureID(bson.DefaultRegistry, doc)
			assert.Nil(t, err, "transformAndEnsureID error: %v", err)
			_, ok := id.(string)
			assert.True(t, ok, "expected returned id type string, got %T", id)
			wantDoc := bson.D{
				{"bin", []byte{0x00, 0x00, 0x00}},
				{"_id", "LongEnoughIdentifier"},
			}
			_, wantBSON, _ := bson.MarshalValue(wantDoc)
			want := bsoncore.Document(wantBSON)
			assert.Equal(t, want, got, "expected document %v, got %v", want, got)
		})
	})
	t.Run("transform aggregate pipeline", func(t *testing.T) {
		index, arr := bsoncore.AppendArrayStart(nil)
		dindex, arr := bsoncore.AppendDocumentElementStart(arr, "0")
		arr = bsoncore.AppendInt32Element(arr, "$limit", 12345)
		arr, _ = bsoncore.AppendDocumentEnd(arr, dindex)
		arr, _ = bsoncore.AppendArrayEnd(arr, index)

		testCases := []struct {
			name     string
			pipeline interface{}
			arr      bson.A
			err      error
		}{
			{
				"Pipeline/error",
				Pipeline{{{"hello", func() {}}}},
				nil,
				MarshalError{Value: primitive.D{}, Err: errors.New("no encoder found for func()")},
			},
			{
				"Pipeline/success",
				Pipeline{{{"hello", "world"}}, {{"pi", 3.14159}}},
				bson.A{
					bson.D{{"hello", "world"}},
					bson.D{{"pi", 3.14159}},
				},
				nil,
			},
			{
				"bson.A",
				bson.A{
					bson.D{{"$limit", 12345}},
				},
				bson.A{
					bson.D{{"$limit", 12345}},
				},
				nil,
			},
			{
				"[]bson.D",
				[]bson.D{{{"$limit", 12345}}},
				bson.A{
					bson.D{{"$limit", 12345}},
				},
				nil,
			},
			{
				"primitive.A/error",
				primitive.A{"5"},
				nil,
				MarshalError{Value: string(""), Err: errors.New("WriteString can only write while positioned on a Element or Value but is positioned on a TopLevel")},
			},
			{
				"primitive.A/success",
				primitive.A{bson.D{{"$limit", int32(12345)}}, map[string]interface{}{"$count": "foobar"}},
				bson.A{
					bson.D{{"$limit", int(12345)}},
					bson.D{{"$count", "foobar"}},
				},
				nil,
			},
			{
				"bson.A/error",
				bson.A{"5"},
				nil,
				MarshalError{Value: string(""), Err: errors.New("WriteString can only write while positioned on a Element or Value but is positioned on a TopLevel")},
			},
			{
				"bson.A/success",
				bson.A{bson.D{{"$limit", int32(12345)}}, map[string]interface{}{"$count": "foobar"}},
				bson.A{
					bson.D{{"$limit", int32(12345)}},
					bson.D{{"$count", "foobar"}},
				},
				nil,
			},
			{
				"[]interface{}/error",
				[]interface{}{"5"},
				nil,
				MarshalError{Value: string(""), Err: errors.New("WriteString can only write while positioned on a Element or Value but is positioned on a TopLevel")},
			},
			{
				"[]interface{}/success",
				[]interface{}{bson.D{{"$limit", int32(12345)}}, map[string]interface{}{"$count": "foobar"}},
				bson.A{
					bson.D{{"$limit", int32(12345)}},
					bson.D{{"$count", "foobar"}},
				},
				nil,
			},
			{
				"bsoncodec.ValueMarshaler/MarshalBSONValue error",
				bvMarsh{err: errors.New("MarshalBSONValue error")},
				nil,
				errors.New("MarshalBSONValue error"),
			},
			{
				"bsoncodec.ValueMarshaler/not array",
				bvMarsh{t: bsontype.String},
				nil,
				fmt.Errorf("ValueMarshaler returned a %v, but was expecting %v", bsontype.String, bsontype.Array),
			},
			{
				"bsoncodec.ValueMarshaler/UnmarshalBSONValue error",
				bvMarsh{t: bsontype.Array, err: bsoncore.NewInsufficientBytesError(nil, nil)},
				nil,
				bsoncore.NewInsufficientBytesError(nil, nil),
			},
			{
				"bsoncodec.ValueMarshaler/success",
				bvMarsh{t: bsontype.Array, data: arr},
				bson.A{
					bson.D{{"$limit", int32(12345)}},
				},
				nil,
			},
			{
				"nil",
				nil,
				nil,
				errors.New("can only transform slices and arrays into aggregation pipelines, but got invalid"),
			},
			{
				"not array or slice",
				int64(42),
				nil,
				errors.New("can only transform slices and arrays into aggregation pipelines, but got int64"),
			},
			{
				"array/error",
				[1]interface{}{int64(42)},
				nil,
				MarshalError{Value: int64(0), Err: errors.New("WriteInt64 can only write while positioned on a Element or Value but is positioned on a TopLevel")},
			},
			{
				"array/success",
				[1]interface{}{primitive.D{{"$limit", int64(12345)}}},
				bson.A{
					bson.D{{"$limit", int64(12345)}},
				},
				nil,
			},
			{
				"slice/error",
				[]interface{}{int64(42)},
				nil,
				MarshalError{Value: int64(0), Err: errors.New("WriteInt64 can only write while positioned on a Element or Value but is positioned on a TopLevel")},
			},
			{
				"slice/success",
				[]interface{}{primitive.D{{"$limit", int64(12345)}}},
				bson.A{
					bson.D{{"$limit", int64(12345)}},
				},
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				arr, _, err := transformAggregatePipeline(bson.NewRegistryBuilder().Build(), tc.pipeline)
				assert.Equal(t, tc.err, err, "expected error %v, got %v", tc.err, err)

				var expected bsoncore.Document
				if tc.arr != nil {
					_, expectedBSON, _ := bson.MarshalValue(tc.arr)
					expected = bsoncore.Document(expectedBSON)
				}
				assert.Equal(t, expected, arr, "expected array %v, got %v", expected, arr)
			})
		}
	})
	t.Run("transform value", func(t *testing.T) {
		valueMarshaler := bvMarsh{
			t:    bsontype.String,
			data: bsoncore.AppendString(nil, "foo"),
		}
		doc := bson.D{{"x", 1}}
		docBytes, _ := bson.Marshal(doc)

		testCases := []struct {
			name      string
			value     interface{}
			err       error
			bsonType  bsontype.Type
			bsonValue []byte
		}{
			{"nil document", nil, ErrNilValue, 0, nil},
			{"value marshaler", valueMarshaler, nil, valueMarshaler.t, valueMarshaler.data},
			{"document", doc, nil, bsontype.EmbeddedDocument, docBytes},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				res, err := transformValue(nil, tc.value, true, "")
				if tc.err != nil {
					assert.Equal(t, tc.err, err, "expected error %v, got %v", tc.err, err)
					return
				}

				assert.Equal(t, tc.bsonType, res.Type, "expected BSON type %s, got %s", tc.bsonType, res.Type)
				assert.Equal(t, tc.bsonValue, res.Data, "expected BSON data %v, got %v", tc.bsonValue, res.Data)
			})
		}
	})
}

var _ bson.Marshaler = bMarsh{}

type bMarsh struct {
	bsonx.Doc
}

func (b bMarsh) MarshalBSON() ([]byte, error) {
	return b.Doc.MarshalBSON()
}

type reflectStruct struct {
	Foo string
}

var _ bsoncodec.ValueMarshaler = bvMarsh{}

type bvMarsh struct {
	t    bsontype.Type
	data []byte
	err  error
}

func (b bvMarsh) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return b.t, b.data, b.err
}
