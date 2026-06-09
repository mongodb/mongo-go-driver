// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/internal/assert"
	"go.mongodb.org/mongo-driver/v2/internal/require"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

func TestDecodeValue(t *testing.T) {
	t.Parallel()

	for _, tc := range unmarshalingTestCases() {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := reflect.New(tc.sType).Elem()
			vr := NewDocumentReader(bytes.NewReader(tc.data))
			reg := defaultRegistry
			decoder, err := reg.LookupDecoder(reflect.TypeOf(got))
			noerr(t, err)
			err = decoder.DecodeValue(DecodeContext{Registry: reg}, vr, got)
			noerr(t, err)
			assert.Equal(t, tc.want, got.Addr().Interface(), "Results do not match.")
		})
	}
}

func TestDecodingInterfaces(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		stub func() ([]byte, any, func(*testing.T))
	}
	testCases := []testCase{
		{
			name: "struct with interface containing a concrete value",
			stub: func() ([]byte, any, func(*testing.T)) {
				type testStruct struct {
					Value any
				}
				var value string

				data := docToBytes(struct {
					Value string
				}{
					Value: "foo",
				})

				receiver := testStruct{&value}

				check := func(t *testing.T) {
					t.Helper()
					assert.Equal(t, "foo", value)
				}

				return data, &receiver, check
			},
		},
		{
			name: "struct with interface containing a struct",
			stub: func() ([]byte, any, func(*testing.T)) {
				type demo struct {
					Data string
				}

				type testStruct struct {
					Value any
				}
				var value demo

				data := docToBytes(struct {
					Value demo
				}{
					Value: demo{"foo"},
				})

				receiver := testStruct{&value}

				check := func(t *testing.T) {
					t.Helper()
					assert.Equal(t, "foo", value.Data)
				}

				return data, &receiver, check
			},
		},
		{
			name: "struct with interface containing a slice",
			stub: func() ([]byte, any, func(*testing.T)) {
				type testStruct struct {
					Values any
				}
				var values []string

				data := docToBytes(struct {
					Values []string
				}{
					Values: []string{"foo", "bar"},
				})

				receiver := testStruct{&values}

				check := func(t *testing.T) {
					t.Helper()
					assert.Equal(t, []string{"foo", "bar"}, values)
				}

				return data, &receiver, check
			},
		},
		{
			name: "struct with interface containing an array",
			stub: func() ([]byte, any, func(*testing.T)) {
				type testStruct struct {
					Values any
				}
				var values [2]string

				data := docToBytes(struct {
					Values []string
				}{
					Values: []string{"foo", "bar"},
				})

				receiver := testStruct{&values}

				check := func(t *testing.T) {
					t.Helper()
					assert.Equal(t, [2]string{"foo", "bar"}, values)
				}

				return data, &receiver, check
			},
		},
		{
			name: "struct with interface array containing concrete values",
			stub: func() ([]byte, any, func(*testing.T)) {
				type testStruct struct {
					Values [3]any
				}
				var str string
				var i, j int

				data := docToBytes(struct {
					Values []any
				}{
					Values: []any{"foo", 42, nil},
				})

				receiver := testStruct{[3]any{&str, &i, &j}}

				check := func(t *testing.T) {
					t.Helper()
					assert.Equal(t, "foo", str)
					assert.Equal(t, 42, i)
					assert.Equal(t, 0, j)
					assert.Equal(t, testStruct{[3]any{&str, &i, nil}}, receiver)
				}

				return data, &receiver, check
			},
		},
		{
			name: "overwriting prepopulated slice",
			stub: func() ([]byte, any, func(*testing.T)) {
				type testStruct struct {
					Values []any
				}

				data := docToBytes(struct {
					Values []any
				}{
					Values: []any{1, 2, 3},
				})

				receiver := testStruct{[]any{7, 8}}

				check := func(t *testing.T) {
					t.Helper()
					assert.Equal(t, testStruct{[]any{1, 2, int32(3)}}, receiver)
				}

				return data, &receiver, check
			},
		},
	}
	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data, receiver, check := tc.stub()
			got := reflect.ValueOf(receiver).Elem()
			vr := NewDocumentReader(bytes.NewReader(data))
			reg := defaultRegistry
			decoder, err := reg.LookupDecoder(got.Type())
			noerr(t, err)
			err = decoder.DecodeValue(DecodeContext{Registry: reg}, vr, got)
			noerr(t, err)
			check(t)
		})
	}
}

func TestDecoder(t *testing.T) {
	t.Parallel()

	t.Run("Decode", func(t *testing.T) {
		t.Parallel()

		t.Run("basic", func(t *testing.T) {
			t.Parallel()

			for _, tc := range unmarshalingTestCases() {
				tc := tc

				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()

					got := reflect.New(tc.sType).Interface()
					vr := NewDocumentReader(bytes.NewReader(tc.data))
					dec := NewDecoder(vr)
					err := dec.Decode(got)
					noerr(t, err)
					assert.Equal(t, tc.want, got, "Results do not match.")
				})
			}
		})
		t.Run("stream", func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			vr := NewDocumentReader(&buf)
			dec := NewDecoder(vr)
			for _, tc := range unmarshalingTestCases() {
				tc := tc

				t.Run(tc.name, func(t *testing.T) {
					buf.Write(tc.data)
					got := reflect.New(tc.sType).Interface()
					err := dec.Decode(got)
					noerr(t, err)
					assert.Equal(t, tc.want, got, "Results do not match.")
				})
			}
		})
		t.Run("lookup error", func(t *testing.T) {
			t.Parallel()

			type certainlydoesntexistelsewhereihope func(string, string) string
			// Avoid unused code lint error.
			_ = certainlydoesntexistelsewhereihope(func(string, string) string { return "" })

			cdeih := func(string, string) string { return "certainlydoesntexistelsewhereihope" }
			dec := NewDecoder(NewDocumentReader(bytes.NewReader([]byte{})))
			want := errNoDecoder{Type: reflect.TypeOf(cdeih)}
			got := dec.Decode(&cdeih)
			assert.Equal(t, want, got, "Received unexpected error.")
		})
		t.Run("Unmarshaler", func(t *testing.T) {
			t.Parallel()

			testCases := []struct {
				name    string
				err     error
				vr      ValueReader
				invoked bool
			}{
				{
					"error",
					errors.New("Unmarshaler error"),
					&valueReaderWriter{BSONType: TypeEmbeddedDocument, Err: ErrEOD, ErrAfter: readElement},
					true,
				},
				{
					"copy error",
					errors.New("copy error"),
					&valueReaderWriter{Err: errors.New("copy error"), ErrAfter: readDocument},
					false,
				},
				{
					"success",
					nil,
					&valueReaderWriter{BSONType: TypeEmbeddedDocument, Err: ErrEOD, ErrAfter: readElement},
					true,
				},
			}

			for _, tc := range testCases {
				tc := tc

				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()

					unmarshaler := &testUnmarshaler{Err: tc.err}
					dec := NewDecoder(tc.vr)
					got := dec.Decode(unmarshaler)
					want := tc.err
					if !assert.CompareErrors(got, want) {
						t.Errorf("Did not receive expected error. got %v; want %v", got, want)
					}
					if unmarshaler.Invoked != tc.invoked {
						if tc.invoked {
							t.Error("Expected to have UnmarshalBSON invoked, but it wasn't.")
						} else {
							t.Error("Expected UnmarshalBSON to not be invoked, but it was.")
						}
					}
				})
			}

			t.Run("Unmarshaler/success ValueReader", func(t *testing.T) {
				t.Parallel()

				want := bsoncore.BuildDocument(nil, bsoncore.AppendDoubleElement(nil, "pi", 3.14159))
				unmarshaler := &testUnmarshaler{}
				vr := NewDocumentReader(bytes.NewReader(want))
				dec := NewDecoder(vr)
				err := dec.Decode(unmarshaler)
				noerr(t, err)
				got := unmarshaler.Val
				if !bytes.Equal(got, want) {
					t.Errorf("Did not unmarshal properly. got %v; want %v", got, want)
				}
			})
		})
	})
	t.Run("NewDecoder", func(t *testing.T) {
		t.Parallel()

		t.Run("success", func(t *testing.T) {
			t.Parallel()

			got := NewDecoder(NewDocumentReader(bytes.NewReader([]byte{})))
			if got == nil {
				t.Errorf("Was expecting a non-nil Decoder, but got <nil>")
			}
		})
	})
	t.Run("NewDecoderWithContext", func(t *testing.T) {
		t.Parallel()

		t.Run("success", func(t *testing.T) {
			t.Parallel()

			got := NewDecoder(NewDocumentReader(bytes.NewReader([]byte{})))
			if got == nil {
				t.Errorf("Was expecting a non-nil Decoder, but got <nil>")
			}
		})
	})
	t.Run("Decode doesn't zero struct", func(t *testing.T) {
		t.Parallel()

		type foo struct {
			Item  string
			Qty   int
			Bonus int
		}
		var got foo
		got.Item = "apple"
		got.Bonus = 2
		data := docToBytes(D{{"item", "canvas"}, {"qty", 4}})
		vr := NewDocumentReader(bytes.NewReader(data))
		dec := NewDecoder(vr)
		err := dec.Decode(&got)
		noerr(t, err)
		want := foo{Item: "canvas", Qty: 4, Bonus: 2}
		assert.Equal(t, want, got, "Results do not match.")
	})
	t.Run("Reset", func(t *testing.T) {
		t.Parallel()

		vr1, vr2 := NewDocumentReader(bytes.NewReader([]byte{})), NewDocumentReader(bytes.NewReader([]byte{}))
		dec := NewDecoder(vr1)
		if dec.vr != vr1 {
			t.Errorf("Decoder should use the value reader provided. got %v; want %v", dec.vr, vr1)
		}
		dec.Reset(vr2)
		if dec.vr != vr2 {
			t.Errorf("Decoder should use the value reader provided. got %v; want %v", dec.vr, vr2)
		}
	})
	t.Run("SetRegistry", func(t *testing.T) {
		t.Parallel()

		r1, r2 := defaultRegistry, NewRegistry()
		dc1 := DecodeContext{Registry: r1}
		dc2 := DecodeContext{Registry: r2}
		dec := NewDecoder(NewDocumentReader(bytes.NewReader([]byte{})))
		if !reflect.DeepEqual(dec.dc, dc1) {
			t.Errorf("Decoder should use the Registry provided. got %v; want %v", dec.dc, dc1)
		}
		dec.SetRegistry(r2)
		if !reflect.DeepEqual(dec.dc, dc2) {
			t.Errorf("Decoder should use the Registry provided. got %v; want %v", dec.dc, dc2)
		}
	})
	t.Run("DecodeToNil", func(t *testing.T) {
		t.Parallel()

		data := docToBytes(D{{"item", "canvas"}, {"qty", 4}})
		vr := NewDocumentReader(bytes.NewReader(data))
		dec := NewDecoder(vr)

		var got *D
		err := dec.Decode(got)
		if !errors.Is(err, ErrDecodeToNil) {
			t.Fatalf("Decode error mismatch; expected %v, got %v", ErrDecodeToNil, err)
		}
	})
}

type testUnmarshaler struct {
	Invoked bool
	Val     []byte
	Err     error
}

func (tu *testUnmarshaler) UnmarshalBSON(d []byte) error {
	tu.Invoked = true
	tu.Val = d
	return tu.Err
}

func TestDecoderConfiguration(t *testing.T) {
	type truncateDoublesTest struct {
		MyInt    int
		MyInt8   int8
		MyInt16  int16
		MyInt32  int32
		MyInt64  int64
		MyUint   uint
		MyUint8  uint8
		MyUint16 uint16
		MyUint32 uint32
		MyUint64 uint64
	}

	type objectIDTest struct {
		ID string
	}

	type jsonStructTest struct {
		StructFieldName string `json:"jsonFieldName"`
	}

	type localTimeZoneTest struct {
		MyTime time.Time
	}

	type zeroMapsTest struct {
		MyMap map[string]string
	}

	type zeroStructsTest struct {
		MyString string
		MyInt    int
	}

	testCases := []struct {
		description string
		configure   func(*Decoder)
		input       []byte
		decodeInto  func() any
		want        any
	}{
		// Test that AllowTruncatingDoubles causes the Decoder to unmarshal BSON doubles with
		// fractional parts into Go integer types by truncating the fractional part.
		{
			description: "AllowTruncatingDoubles",
			configure: func(dec *Decoder) {
				dec.AllowTruncatingDoubles()
			},
			input: bsoncore.NewDocumentBuilder().
				AppendDouble("myInt", 1.999).
				AppendDouble("myInt8", 1.999).
				AppendDouble("myInt16", 1.999).
				AppendDouble("myInt32", 1.999).
				AppendDouble("myInt64", 1.999).
				AppendDouble("myUint", 1.999).
				AppendDouble("myUint8", 1.999).
				AppendDouble("myUint16", 1.999).
				AppendDouble("myUint32", 1.999).
				AppendDouble("myUint64", 1.999).
				Build(),
			decodeInto: func() any { return &truncateDoublesTest{} },
			want: &truncateDoublesTest{
				MyInt:    1,
				MyInt8:   1,
				MyInt16:  1,
				MyInt32:  1,
				MyInt64:  1,
				MyUint:   1,
				MyUint8:  1,
				MyUint16: 1,
				MyUint32: 1,
				MyUint64: 1,
			},
		},
		// Test that BinaryAsSlice causes the Decoder to unmarshal BSON binary fields into Go byte
		// slices when there is no type information (e.g when unmarshaling into a bson.D).
		{
			description: "BinaryAsSlice",
			configure: func(dec *Decoder) {
				dec.BinaryAsSlice()
			},
			input: bsoncore.NewDocumentBuilder().
				AppendBinary("myBinary", TypeBinaryGeneric, []byte{}).
				Build(),
			decodeInto: func() any { return &D{} },
			want:       &D{{Key: "myBinary", Value: []byte{}}},
		},
		// Test that the default decoder always decodes BSON documents into bson.D values,
		// independent of the top-level Go value type.
		{
			description: "DocumentD nested by default",
			configure:   func(_ *Decoder) {},
			input: bsoncore.NewDocumentBuilder().
				AppendDocument("myDocument", bsoncore.NewDocumentBuilder().
					AppendString("myString", "test value").
					Build()).
				Build(),
			decodeInto: func() any { return M{} },
			want: M{
				"myDocument": D{{Key: "myString", Value: "test value"}},
			},
		},
		// Test that DefaultDocumentM always decodes BSON documents into bson.M values,
		// independent of the top-level Go value type.
		{
			description: "DefaultDocumentM nested",
			configure: func(dec *Decoder) {
				dec.DefaultDocumentM()
			},
			input: bsoncore.NewDocumentBuilder().
				AppendDocument("myDocument", bsoncore.NewDocumentBuilder().
					AppendString("myString", "test value").
					Build()).
				Build(),
			decodeInto: func() any { return &D{} },
			want: &D{
				{Key: "myDocument", Value: M{"myString": "test value"}},
			},
		},
		// Test that DefaultDocumentMap always decodes BSON documents into
		// map[string]any values, independent of the top-level Go value type.
		{
			description: "DefaultDocumentMap nested",
			configure: func(dec *Decoder) {
				dec.DefaultDocumentMap()
			},
			input: bsoncore.NewDocumentBuilder().
				AppendDocument("myDocument", bsoncore.NewDocumentBuilder().
					AppendString("myString", "test value").
					Build()).
				Build(),
			decodeInto: func() any { return &D{} },
			want: &D{
				{Key: "myDocument", Value: map[string]any{"myString": "test value"}},
			},
		},
		// Test that ObjectIDAsHexString causes the Decoder to decode object ID to hex.
		{
			description: "ObjectIDAsHexString",
			configure: func(dec *Decoder) {
				dec.ObjectIDAsHexString()
			},
			input: bsoncore.NewDocumentBuilder().
				AppendObjectID("id", func() ObjectID {
					id, _ := ObjectIDFromHex("5ef7fdd91c19e3222b41b839")
					return id
				}()).
				Build(),
			decodeInto: func() any { return &objectIDTest{} },
			want:       &objectIDTest{ID: "5ef7fdd91c19e3222b41b839"},
		},
		// Test that UseJSONStructTags causes the Decoder to fall back to "json" struct tags if
		// "bson" struct tags are not available.
		{
			description: "UseJSONStructTags",
			configure: func(dec *Decoder) {
				dec.UseJSONStructTags()
			},
			input: bsoncore.NewDocumentBuilder().
				AppendString("jsonFieldName", "test value").
				Build(),
			decodeInto: func() any { return &jsonStructTest{} },
			want:       &jsonStructTest{StructFieldName: "test value"},
		},
		// Test that UseLocalTimeZone causes the Decoder to use the local time zone for decoded
		// time.Time values instead of UTC.
		{
			description: "UseLocalTimeZone",
			configure: func(dec *Decoder) {
				dec.UseLocalTimeZone()
			},
			input: bsoncore.NewDocumentBuilder().
				AppendDateTime("myTime", 1684349179939).
				Build(),
			decodeInto: func() any { return &localTimeZoneTest{} },
			want:       &localTimeZoneTest{MyTime: time.UnixMilli(1684349179939)},
		},
		// Test that ZeroMaps causes the Decoder to empty any Go map values before decoding BSON
		// documents into them.
		{
			description: "ZeroMaps",
			configure: func(dec *Decoder) {
				dec.ZeroMaps()
			},
			input: bsoncore.NewDocumentBuilder().
				AppendDocument("myMap", bsoncore.NewDocumentBuilder().
					AppendString("myString", "test value").
					Build()).
				Build(),
			decodeInto: func() any {
				return &zeroMapsTest{MyMap: map[string]string{"myExtraValue": "extra value"}}
			},
			want: &zeroMapsTest{MyMap: map[string]string{"myString": "test value"}},
		},
		// Test that ZeroStructs causes the Decoder to empty any Go struct values before decoding
		// BSON documents into them.
		{
			description: "ZeroStructs",
			configure: func(dec *Decoder) {
				dec.ZeroStructs()
			},
			input: bsoncore.NewDocumentBuilder().
				AppendString("myString", "test value").
				Build(),
			decodeInto: func() any {
				return &zeroStructsTest{MyInt: 1}
			},
			want: &zeroStructsTest{MyString: "test value"},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			dec := NewDecoder(NewDocumentReader(bytes.NewReader(tc.input)))

			tc.configure(dec)

			got := tc.decodeInto()
			err := dec.Decode(got)
			require.NoError(t, err, "Decode error")

			assert.Equal(t, tc.want, got, "expected and actual decode results do not match")
		})
	}

	t.Run("Decoding an object ID to string", func(t *testing.T) {
		t.Parallel()

		type objectIDTest struct {
			ID string
		}

		doc := bsoncore.NewDocumentBuilder().
			AppendObjectID("id", func() ObjectID {
				id, _ := ObjectIDFromHex("5ef7fdd91c19e3222b41b839")
				return id
			}()).
			Build()

		dec := NewDecoder(NewDocumentReader(bytes.NewReader(doc)))

		var got objectIDTest
		err := dec.Decode(&got)
		const want = "error decoding key id: decoding an object ID into a string is not supported by default (set Decoder.ObjectIDAsHexString to enable decoding as a hexadecimal string)"
		assert.EqualError(t, err, want)
	})
	t.Run("DefaultDocumentM top-level", func(t *testing.T) {
		t.Parallel()

		input := bsoncore.NewDocumentBuilder().
			AppendDocument("myDocument", bsoncore.NewDocumentBuilder().
				AppendString("myString", "test value").
				Build()).
			Build()

		dec := NewDecoder(NewDocumentReader(bytes.NewReader(input)))

		dec.DefaultDocumentM()

		var got any
		err := dec.Decode(&got)
		require.NoError(t, err, "Decode error")

		want := M{
			"myDocument": M{
				"myString": "test value",
			},
		}
		assert.Equal(t, want, got, "expected and actual decode results do not match")
	})
	t.Run("DefaultDocumentMap top-level", func(t *testing.T) {
		t.Parallel()

		input := bsoncore.NewDocumentBuilder().
			AppendDocument("myDocument", bsoncore.NewDocumentBuilder().
				AppendString("myString", "test value").
				Build()).
			Build()

		dec := NewDecoder(NewDocumentReader(bytes.NewReader(input)))

		dec.DefaultDocumentMap()

		var got any
		err := dec.Decode(&got)
		require.NoError(t, err, "Decode error")

		want := map[string]any{
			"myDocument": map[string]any{
				"myString": "test value",
			},
		}
		assert.Equal(t, want, got, "expected and actual decode results do not match")
	})
	t.Run("Default decodes DocumentD for top-level", func(t *testing.T) {
		t.Parallel()

		input := bsoncore.NewDocumentBuilder().
			AppendDocument("myDocument", bsoncore.NewDocumentBuilder().
				AppendString("myString", "test value").
				Build()).
			Build()

		dec := NewDecoder(NewDocumentReader(bytes.NewReader(input)))

		var got any
		err := dec.Decode(&got)
		require.NoError(t, err, "Decode error")

		want := D{
			{Key: "myDocument", Value: D{
				{Key: "myString", Value: "test value"},
			}},
		}
		assert.Equal(t, want, got, "expected and actual decode results do not match")
	})
}
