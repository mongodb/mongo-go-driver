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
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/codecutil"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func TestEnsureID(t *testing.T) {
	t.Parallel()

	oid := primitive.NewObjectID()

	testCases := []struct {
		description string
		// TODO: Registry? DecodeOptions?
		doc    bsoncore.Document
		oid    primitive.ObjectID
		want   bsoncore.Document
		wantID interface{}
	}{
		{
			description: "missing _id should be first element",
			doc: bsoncore.NewDocumentBuilder().
				AppendString("foo", "bar").
				AppendString("baz", "quix").
				AppendString("hello", "world").
				Build(),
			want: bsoncore.NewDocumentBuilder().
				AppendObjectID("_id", oid).
				AppendString("foo", "bar").
				AppendString("baz", "quix").
				AppendString("hello", "world").
				Build(),
			wantID: oid,
		},
		{
			description: "existing ObjectID _id as should remain in place",
			doc: bsoncore.NewDocumentBuilder().
				AppendString("foo", "bar").
				AppendObjectID("_id", oid).
				AppendString("baz", "quix").
				AppendString("hello", "world").
				Build(),
			want: bsoncore.NewDocumentBuilder().
				AppendString("foo", "bar").
				AppendObjectID("_id", oid).
				AppendString("baz", "quix").
				AppendString("hello", "world").
				Build(),
			wantID: oid,
		},
		{
			description: "existing float _id as should remain in place",
			doc: bsoncore.NewDocumentBuilder().
				AppendString("foo", "bar").
				AppendDouble("_id", 3.14159).
				AppendString("baz", "quix").
				AppendString("hello", "world").
				Build(),
			want: bsoncore.NewDocumentBuilder().
				AppendString("foo", "bar").
				AppendDouble("_id", 3.14159).
				AppendString("baz", "quix").
				AppendString("hello", "world").
				Build(),
			wantID: 3.14159,
		},
		{
			description: "existing float _id as first element should remain first element",
			doc: bsoncore.NewDocumentBuilder().
				AppendDouble("_id", 3.14159).
				AppendString("foo", "bar").
				AppendString("baz", "quix").
				AppendString("hello", "world").
				Build(),
			want: bsoncore.NewDocumentBuilder().
				AppendDouble("_id", 3.14159).
				AppendString("foo", "bar").
				AppendString("baz", "quix").
				AppendString("hello", "world").
				Build(),
			wantID: 3.14159,
		},
		{
			description: "existing binary _id as first field should not be overwritten",
			doc: bsoncore.NewDocumentBuilder().
				AppendBinary("bin", 0, []byte{0, 0, 0}).
				AppendString("_id", "LongEnoughIdentifier").
				Build(),
			want: bsoncore.NewDocumentBuilder().
				AppendBinary("bin", 0, []byte{0, 0, 0}).
				AppendString("_id", "LongEnoughIdentifier").
				Build(),
			wantID: "LongEnoughIdentifier",
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			got, gotID, err := ensureID(tc.doc, oid, nil, nil)
			require.NoError(t, err, "ensureID error")

			assert.Equal(t, tc.want, got, "expected and actual documents are different")
			assert.Equal(t, tc.wantID, gotID, "expected and actual IDs are different")

			// Ensure that if the unmarshaled "_id" value is a
			// primitive.ObjectID that it is a deep copy and does not share any
			// memory with the document byte slice.
			if oid, ok := gotID.(primitive.ObjectID); ok {
				assert.DifferentAddressRanges(t, tc.doc, oid[:])
			}
		})
	}
}

func TestEnsureID_NilObjectID(t *testing.T) {
	t.Parallel()

	doc := bsoncore.NewDocumentBuilder().
		AppendString("foo", "bar").
		Build()

	got, gotIDI, err := ensureID(doc, primitive.NilObjectID, nil, nil)
	assert.NoError(t, err)

	gotID, ok := gotIDI.(primitive.ObjectID)

	assert.True(t, ok)
	assert.NotEqual(t, primitive.NilObjectID, gotID)

	want := bsoncore.NewDocumentBuilder().
		AppendObjectID("_id", gotID).
		AppendString("foo", "bar").
		Build()

	assert.Equal(t, want, got)
}

func TestMarshalAggregatePipeline(t *testing.T) {
	// []byte of [{{"$limit", 12345}}]
	index, arr := bsoncore.AppendArrayStart(nil)
	dindex, arr := bsoncore.AppendDocumentElementStart(arr, "0")
	arr = bsoncore.AppendInt32Element(arr, "$limit", 12345)
	arr, _ = bsoncore.AppendDocumentEnd(arr, dindex)
	arr, _ = bsoncore.AppendArrayEnd(arr, index)

	// []byte of {{"x", 1}}
	index, doc := bsoncore.AppendDocumentStart(nil)
	doc = bsoncore.AppendInt32Element(doc, "x", 1)
	doc, _ = bsoncore.AppendDocumentEnd(doc, index)

	// bsoncore.Array of [{{"$merge", {}}}]
	mergeStage := bsoncore.NewDocumentBuilder().
		StartDocument("$merge").
		FinishDocument().
		Build()
	arrMergeStage := bsoncore.NewArrayBuilder().AppendDocument(mergeStage).Build()

	fooStage := bsoncore.NewDocumentBuilder().AppendString("foo", "bar").Build()
	bazStage := bsoncore.NewDocumentBuilder().AppendString("baz", "qux").Build()
	outStage := bsoncore.NewDocumentBuilder().AppendString("$out", "myColl").Build()

	// bsoncore.Array of [{{"foo", "bar"}}, {{"baz", "qux"}}, {{"$out", "myColl"}}]
	arrOutStage := bsoncore.NewArrayBuilder().
		AppendDocument(fooStage).
		AppendDocument(bazStage).
		AppendDocument(outStage).
		Build()

	// bsoncore.Array of [{{"foo", "bar"}}, {{"$out", "myColl"}}, {{"baz", "qux"}}]
	arrMiddleOutStage := bsoncore.NewArrayBuilder().
		AppendDocument(fooStage).
		AppendDocument(outStage).
		AppendDocument(bazStage).
		Build()

	testCases := []struct {
		name           string
		pipeline       interface{}
		arr            bson.A
		hasOutputStage bool
		err            error
	}{
		{
			"Pipeline/error",
			Pipeline{{{"hello", func() {}}}},
			nil,
			false,
			MarshalError{Value: primitive.D{}, Err: errors.New("no encoder found for func()")},
		},
		{
			"Pipeline/success",
			Pipeline{{{"hello", "world"}}, {{"pi", 3.14159}}},
			bson.A{
				bson.D{{"hello", "world"}},
				bson.D{{"pi", 3.14159}},
			},
			false,
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
			false,
			nil,
		},
		{
			"[]bson.D",
			[]bson.D{{{"$limit", 12345}}},
			bson.A{
				bson.D{{"$limit", 12345}},
			},
			false,
			nil,
		},
		{
			"primitive.A/error",
			primitive.A{"5"},
			nil,
			false,
			MarshalError{Value: "", Err: errors.New("WriteString can only write while positioned on a Element or Value but is positioned on a TopLevel")},
		},
		{
			"primitive.A/success",
			primitive.A{bson.D{{"$limit", int32(12345)}}, map[string]interface{}{"$count": "foobar"}},
			bson.A{
				bson.D{{"$limit", int(12345)}},
				bson.D{{"$count", "foobar"}},
			},
			false,
			nil,
		},
		{
			"bson.A/error",
			bson.A{"5"},
			nil,
			false,
			MarshalError{Value: "", Err: errors.New("WriteString can only write while positioned on a Element or Value but is positioned on a TopLevel")},
		},
		{
			"bson.A/success",
			bson.A{bson.D{{"$limit", int32(12345)}}, map[string]interface{}{"$count": "foobar"}},
			bson.A{
				bson.D{{"$limit", int32(12345)}},
				bson.D{{"$count", "foobar"}},
			},
			false,
			nil,
		},
		{
			"[]interface{}/error",
			[]interface{}{"5"},
			nil,
			false,
			MarshalError{Value: "", Err: errors.New("WriteString can only write while positioned on a Element or Value but is positioned on a TopLevel")},
		},
		{
			"[]interface{}/success",
			[]interface{}{bson.D{{"$limit", int32(12345)}}, map[string]interface{}{"$count": "foobar"}},
			bson.A{
				bson.D{{"$limit", int32(12345)}},
				bson.D{{"$count", "foobar"}},
			},
			false,
			nil,
		},
		{
			"bsoncodec.ValueMarshaler/MarshalBSONValue error",
			bvMarsh{err: errors.New("MarshalBSONValue error")},
			nil,
			false,
			errors.New("MarshalBSONValue error"),
		},
		{
			"bsoncodec.ValueMarshaler/not array",
			bvMarsh{t: bsontype.String},
			nil,
			false,
			fmt.Errorf("ValueMarshaler returned a %v, but was expecting %v", bsontype.String, bsontype.Array),
		},
		{
			"bsoncodec.ValueMarshaler/UnmarshalBSONValue error",
			bvMarsh{err: errors.New("UnmarshalBSONValue error")},
			nil,
			false,
			errors.New("UnmarshalBSONValue error"),
		},
		{
			"bsoncodec.ValueMarshaler/success",
			bvMarsh{t: bsontype.Array, data: arr},
			bson.A{
				bson.D{{"$limit", int32(12345)}},
			},
			false,
			nil,
		},
		{
			"bsoncodec.ValueMarshaler/success nil",
			bvMarsh{t: bsontype.Array},
			nil,
			false,
			nil,
		},
		{
			"nil",
			nil,
			nil,
			false,
			errors.New("can only marshal slices and arrays into aggregation pipelines, but got invalid"),
		},
		{
			"not array or slice",
			int64(42),
			nil,
			false,
			errors.New("can only marshal slices and arrays into aggregation pipelines, but got int64"),
		},
		{
			"array/error",
			[1]interface{}{int64(42)},
			nil,
			false,
			MarshalError{Value: int64(0), Err: errors.New("WriteInt64 can only write while positioned on a Element or Value but is positioned on a TopLevel")},
		},
		{
			"array/success",
			[1]interface{}{primitive.D{{"$limit", int64(12345)}}},
			bson.A{
				bson.D{{"$limit", int64(12345)}},
			},
			false,
			nil,
		},
		{
			"slice/error",
			[]interface{}{int64(42)},
			nil,
			false,
			MarshalError{Value: int64(0), Err: errors.New("WriteInt64 can only write while positioned on a Element or Value but is positioned on a TopLevel")},
		},
		{
			"slice/success",
			[]interface{}{primitive.D{{"$limit", int64(12345)}}},
			bson.A{
				bson.D{{"$limit", int64(12345)}},
			},
			false,
			nil,
		},
		{
			"hasOutputStage/out",
			bson.A{
				bson.D{{"$out", bson.D{
					{"db", "output-db"},
					{"coll", "output-collection"},
				}}},
			},
			bson.A{
				bson.D{{"$out", bson.D{
					{"db", "output-db"},
					{"coll", "output-collection"},
				}}},
			},
			true,
			nil,
		},
		{
			"hasOutputStage/merge",
			bson.A{
				bson.D{{"$merge", bson.D{
					{"into", bson.D{
						{"db", "output-db"},
						{"coll", "output-collection"},
					}},
				}}},
			},
			bson.A{
				bson.D{{"$merge", bson.D{
					{"into", bson.D{
						{"db", "output-db"},
						{"coll", "output-collection"},
					}},
				}}},
			},
			true,
			nil,
		},
		{
			"semantic single document/bson.D",
			bson.D{{"x", 1}},
			nil,
			false,
			errors.New("primitive.D is not an allowed pipeline type as it represents a single document. Use bson.A or mongo.Pipeline instead"),
		},
		{
			"semantic single document/bson.Raw",
			bson.Raw(doc),
			nil,
			false,
			errors.New("bson.Raw is not an allowed pipeline type as it represents a single document. Use bson.A or mongo.Pipeline instead"),
		},
		{
			"semantic single document/bsoncore.Document",
			bsoncore.Document(doc),
			nil,
			false,
			errors.New("bsoncore.Document is not an allowed pipeline type as it represents a single document. Use bson.A or mongo.Pipeline instead"),
		},
		{
			"semantic single document/empty bson.D",
			bson.D{},
			bson.A{},
			false,
			nil,
		},
		{
			"semantic single document/empty bson.Raw",
			bson.Raw{},
			bson.A{},
			false,
			nil,
		},
		{
			"semantic single document/empty bsoncore.Document",
			bsoncore.Document{},
			bson.A{},
			false,
			nil,
		},
		{
			"bsoncore.Array/success",
			bsoncore.Array(arr),
			bson.A{
				bson.D{{"$limit", int32(12345)}},
			},
			false,
			nil,
		},
		{
			"bsoncore.Array/mergeStage",
			arrMergeStage,
			bson.A{
				bson.D{{"$merge", bson.D{}}},
			},
			true,
			nil,
		},
		{
			"bsoncore.Array/outStage",
			arrOutStage,
			bson.A{
				bson.D{{"foo", "bar"}},
				bson.D{{"baz", "qux"}},
				bson.D{{"$out", "myColl"}},
			},
			true,
			nil,
		},
		{
			"bsoncore.Array/middleOutStage",
			arrMiddleOutStage,
			bson.A{
				bson.D{{"foo", "bar"}},
				bson.D{{"$out", "myColl"}},
				bson.D{{"baz", "qux"}},
			},
			false,
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			arr, hasOutputStage, err := marshalAggregatePipeline(tc.pipeline, nil, nil)
			assert.Equal(t, tc.hasOutputStage, hasOutputStage, "expected hasOutputStage %v, got %v",
				tc.hasOutputStage, hasOutputStage)
			if tc.err != nil {
				assert.NotNil(t, err)
				assert.EqualError(t, err, tc.err.Error())
			} else {
				assert.Nil(t, err)
			}

			var expected bsoncore.Document
			if tc.arr != nil {
				_, expectedBSON, err := bson.MarshalValue(tc.arr)
				assert.Nil(t, err, "MarshalValue error: %v", err)
				expected = bsoncore.Document(expectedBSON)
			}
			assert.Equal(t, expected, arr, "expected array %v, got %v", expected, arr)
		})
	}
}

func TestMarshalValue(t *testing.T) {
	t.Parallel()

	valueMarshaler := bvMarsh{
		t:    bson.TypeString,
		data: bsoncore.AppendString(nil, "foo"),
	}

	testCases := []struct {
		name     string
		value    interface{}
		bsonOpts *options.BSONOptions
		registry *bsoncodec.Registry
		want     bsoncore.Value
		wantErr  error
	}{
		{
			name:    "nil document",
			value:   nil,
			wantErr: codecutil.ErrNilValue,
		},
		{
			name:  "value marshaler",
			value: valueMarshaler,
			want: bsoncore.Value{
				Type: valueMarshaler.t,
				Data: valueMarshaler.data,
			},
		},
		{
			name:  "document",
			value: bson.D{{Key: "x", Value: int64(1)}},
			want: bsoncore.Value{
				Type: bson.TypeEmbeddedDocument,
				Data: bsoncore.NewDocumentBuilder().
					AppendInt64("x", 1).
					Build(),
			},
		},
		{
			name: "custom encode options",
			value: struct {
				Int         int64
				NilBytes    []byte
				NilMap      map[string]interface{}
				NilStrings  []string
				ZeroStruct  struct{ X int } `bson:"_,omitempty"`
				StringerMap map[*bson.RawValue]bool
				BSONField   string `json:"jsonField"`
			}{
				Int:         1,
				NilBytes:    nil,
				NilMap:      nil,
				NilStrings:  nil,
				StringerMap: map[*bson.RawValue]bool{{}: true},
			},
			bsonOpts: &options.BSONOptions{
				IntMinSize:              true,
				NilByteSliceAsEmpty:     true,
				NilMapAsEmpty:           true,
				NilSliceAsEmpty:         true,
				OmitZeroStruct:          true,
				StringifyMapKeysWithFmt: true,
				UseJSONStructTags:       true,
			},
			want: bsoncore.Value{
				Type: bson.TypeEmbeddedDocument,
				Data: bsoncore.NewDocumentBuilder().
					AppendInt32("int", 1).
					AppendBinary("nilbytes", 0, []byte{}).
					AppendDocument("nilmap", bsoncore.NewDocumentBuilder().Build()).
					AppendArray("nilstrings", bsoncore.NewArrayBuilder().Build()).
					AppendDocument("stringermap", bsoncore.NewDocumentBuilder().
						AppendBoolean("", true).
						Build()).
					AppendString("jsonField", "").
					Build(),
			},
		},
	}
	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := marshalValue(tc.value, tc.bsonOpts, tc.registry)
			assert.EqualBSON(t, tc.want, got)
			assert.Equal(t, tc.wantErr, err, "expected and actual error do not match")
		})
	}
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
