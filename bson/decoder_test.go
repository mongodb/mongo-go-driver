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

	"github.com/google/go-cmp/cmp"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/bsonrw/bsonrwtest"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/internal/assert"
	"go.mongodb.org/mongo-driver/internal/require"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func TestBasicDecode(t *testing.T) {
	t.Parallel()

	for _, tc := range unmarshalingTestCases() {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := reflect.New(tc.sType).Elem()
			vr := bsonrw.NewBSONDocumentReader(tc.data)
			reg := DefaultRegistry
			decoder, err := reg.LookupDecoder(reflect.TypeOf(got))
			noerr(t, err)
			err = decoder.DecodeValue(bsoncodec.DecodeContext{Registry: reg}, vr, got)
			noerr(t, err)
			assert.Equal(t, tc.want, got.Addr().Interface(), "Results do not match.")
		})
	}
}

func TestDecoderv2(t *testing.T) {
	t.Parallel()

	t.Run("Decode", func(t *testing.T) {
		t.Parallel()

		for _, tc := range unmarshalingTestCases() {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				got := reflect.New(tc.sType).Interface()
				vr := bsonrw.NewBSONDocumentReader(tc.data)
				dec, err := NewDecoderWithContext(bsoncodec.DecodeContext{Registry: DefaultRegistry}, vr)
				noerr(t, err)
				err = dec.Decode(got)
				noerr(t, err)
				assert.Equal(t, tc.want, got, "Results do not match.")
			})
		}
		t.Run("lookup error", func(t *testing.T) {
			t.Parallel()

			type certainlydoesntexistelsewhereihope func(string, string) string
			// Avoid unused code lint error.
			_ = certainlydoesntexistelsewhereihope(func(string, string) string { return "" })

			cdeih := func(string, string) string { return "certainlydoesntexistelsewhereihope" }
			dec, err := NewDecoder(bsonrw.NewBSONDocumentReader([]byte{}))
			noerr(t, err)
			want := bsoncodec.ErrNoDecoder{Type: reflect.TypeOf(cdeih)}
			got := dec.Decode(&cdeih)
			assert.Equal(t, want, got, "Received unexpected error.")
		})
		t.Run("Unmarshaler", func(t *testing.T) {
			t.Parallel()

			testCases := []struct {
				name    string
				err     error
				vr      bsonrw.ValueReader
				invoked bool
			}{
				{
					"error",
					errors.New("Unmarshaler error"),
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.EmbeddedDocument, Err: bsonrw.ErrEOD, ErrAfter: bsonrwtest.ReadElement},
					true,
				},
				{
					"copy error",
					errors.New("copy error"),
					&bsonrwtest.ValueReaderWriter{Err: errors.New("copy error"), ErrAfter: bsonrwtest.ReadDocument},
					false,
				},
				{
					"success",
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.EmbeddedDocument, Err: bsonrw.ErrEOD, ErrAfter: bsonrwtest.ReadElement},
					true,
				},
			}

			for _, tc := range testCases {
				tc := tc

				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()

					unmarshaler := &testUnmarshaler{err: tc.err}
					dec, err := NewDecoder(tc.vr)
					noerr(t, err)
					got := dec.Decode(unmarshaler)
					want := tc.err
					if !compareErrors(got, want) {
						t.Errorf("Did not receive expected error. got %v; want %v", got, want)
					}
					if unmarshaler.invoked != tc.invoked {
						if tc.invoked {
							t.Error("Expected to have UnmarshalBSON invoked, but it wasn't.")
						} else {
							t.Error("Expected UnmarshalBSON to not be invoked, but it was.")
						}
					}
				})
			}

			t.Run("Unmarshaler/success bsonrw.ValueReader", func(t *testing.T) {
				t.Parallel()

				want := bsoncore.BuildDocument(nil, bsoncore.AppendDoubleElement(nil, "pi", 3.14159))
				unmarshaler := &testUnmarshaler{}
				vr := bsonrw.NewBSONDocumentReader(want)
				dec, err := NewDecoder(vr)
				noerr(t, err)
				err = dec.Decode(unmarshaler)
				noerr(t, err)
				got := unmarshaler.data
				if !bytes.Equal(got, want) {
					t.Errorf("Did not unmarshal properly. got %v; want %v", got, want)
				}
			})
		})
	})
	t.Run("NewDecoder", func(t *testing.T) {
		t.Parallel()

		t.Run("error", func(t *testing.T) {
			t.Parallel()

			_, got := NewDecoder(nil)
			want := errors.New("cannot create a new Decoder with a nil ValueReader")
			if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
				t.Errorf("Was expecting error but got different error. got %v; want %v", got, want)
			}
		})
		t.Run("success", func(t *testing.T) {
			t.Parallel()

			got, err := NewDecoder(bsonrw.NewBSONDocumentReader([]byte{}))
			noerr(t, err)
			if got == nil {
				t.Errorf("Was expecting a non-nil Decoder, but got <nil>")
			}
		})
	})
	t.Run("NewDecoderWithContext", func(t *testing.T) {
		t.Parallel()

		t.Run("errors", func(t *testing.T) {
			t.Parallel()

			dc := bsoncodec.DecodeContext{Registry: DefaultRegistry}
			_, got := NewDecoderWithContext(dc, nil)
			want := errors.New("cannot create a new Decoder with a nil ValueReader")
			if !cmp.Equal(got, want, cmp.Comparer(compareErrors)) {
				t.Errorf("Was expecting error but got different error. got %v; want %v", got, want)
			}
		})
		t.Run("success", func(t *testing.T) {
			t.Parallel()

			got, err := NewDecoderWithContext(bsoncodec.DecodeContext{}, bsonrw.NewBSONDocumentReader([]byte{}))
			noerr(t, err)
			if got == nil {
				t.Errorf("Was expecting a non-nil Decoder, but got <nil>")
			}
			dc := bsoncodec.DecodeContext{Registry: DefaultRegistry}
			got, err = NewDecoderWithContext(dc, bsonrw.NewBSONDocumentReader([]byte{}))
			noerr(t, err)
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
		vr := bsonrw.NewBSONDocumentReader(data)
		dec, err := NewDecoder(vr)
		noerr(t, err)
		err = dec.Decode(&got)
		noerr(t, err)
		want := foo{Item: "canvas", Qty: 4, Bonus: 2}
		assert.Equal(t, want, got, "Results do not match.")
	})
	t.Run("Reset", func(t *testing.T) {
		t.Parallel()

		vr1, vr2 := bsonrw.NewBSONDocumentReader([]byte{}), bsonrw.NewBSONDocumentReader([]byte{})
		dc := bsoncodec.DecodeContext{Registry: DefaultRegistry}
		dec, err := NewDecoderWithContext(dc, vr1)
		noerr(t, err)
		if dec.vr != vr1 {
			t.Errorf("Decoder should use the value reader provided. got %v; want %v", dec.vr, vr1)
		}
		err = dec.Reset(vr2)
		noerr(t, err)
		if dec.vr != vr2 {
			t.Errorf("Decoder should use the value reader provided. got %v; want %v", dec.vr, vr2)
		}
	})
	t.Run("SetContext", func(t *testing.T) {
		t.Parallel()

		dc1 := bsoncodec.DecodeContext{Registry: DefaultRegistry}
		dc2 := bsoncodec.DecodeContext{Registry: NewRegistryBuilder().Build()}
		dec, err := NewDecoderWithContext(dc1, bsonrw.NewBSONDocumentReader([]byte{}))
		noerr(t, err)
		if !reflect.DeepEqual(dec.dc, dc1) {
			t.Errorf("Decoder should use the Registry provided. got %v; want %v", dec.dc, dc1)
		}
		err = dec.SetContext(dc2)
		noerr(t, err)
		if !reflect.DeepEqual(dec.dc, dc2) {
			t.Errorf("Decoder should use the Registry provided. got %v; want %v", dec.dc, dc2)
		}
	})
	t.Run("SetRegistry", func(t *testing.T) {
		t.Parallel()

		r1, r2 := DefaultRegistry, NewRegistryBuilder().Build()
		dc1 := bsoncodec.DecodeContext{Registry: r1}
		dc2 := bsoncodec.DecodeContext{Registry: r2}
		dec, err := NewDecoder(bsonrw.NewBSONDocumentReader([]byte{}))
		noerr(t, err)
		if !reflect.DeepEqual(dec.dc, dc1) {
			t.Errorf("Decoder should use the Registry provided. got %v; want %v", dec.dc, dc1)
		}
		err = dec.SetRegistry(r2)
		noerr(t, err)
		if !reflect.DeepEqual(dec.dc, dc2) {
			t.Errorf("Decoder should use the Registry provided. got %v; want %v", dec.dc, dc2)
		}
	})
	t.Run("DecodeToNil", func(t *testing.T) {
		t.Parallel()

		data := docToBytes(D{{"item", "canvas"}, {"qty", 4}})
		vr := bsonrw.NewBSONDocumentReader(data)
		dec, err := NewDecoder(vr)
		noerr(t, err)

		var got *D
		err = dec.Decode(got)
		if !errors.Is(err, ErrDecodeToNil) {
			t.Fatalf("Decode error mismatch; expected %v, got %v", ErrDecodeToNil, err)
		}
	})
}

type testUnmarshaler struct {
	invoked bool
	err     error
	data    []byte
}

func (tu *testUnmarshaler) UnmarshalBSON(d []byte) error {
	tu.invoked = true
	tu.data = d
	return tu.err
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
		decodeInto  func() interface{}
		want        interface{}
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
			decodeInto: func() interface{} { return &truncateDoublesTest{} },
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
				AppendBinary("myBinary", bsontype.BinaryGeneric, []byte{}).
				Build(),
			decodeInto: func() interface{} { return &D{} },
			want:       &D{{Key: "myBinary", Value: []byte{}}},
		},
		// Test that DefaultDocumentD overrides the default "ancestor" logic and always decodes BSON
		// documents into bson.D values, independent of the top-level Go value type.
		{
			description: "DefaultDocumentD nested",
			configure: func(dec *Decoder) {
				dec.DefaultDocumentD()
			},
			input: bsoncore.NewDocumentBuilder().
				AppendDocument("myDocument", bsoncore.NewDocumentBuilder().
					AppendString("myString", "test value").
					Build()).
				Build(),
			decodeInto: func() interface{} { return M{} },
			want: M{
				"myDocument": D{{Key: "myString", Value: "test value"}},
			},
		},
		// Test that DefaultDocumentM overrides the default "ancestor" logic and always decodes BSON
		// documents into bson.M values, independent of the top-level Go value type.
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
			decodeInto: func() interface{} { return &D{} },
			want: &D{
				{Key: "myDocument", Value: M{"myString": "test value"}},
			},
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
			decodeInto: func() interface{} { return &jsonStructTest{} },
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
			decodeInto: func() interface{} { return &localTimeZoneTest{} },
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
			decodeInto: func() interface{} {
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
			decodeInto: func() interface{} {
				return &zeroStructsTest{MyInt: 1}
			},
			want: &zeroStructsTest{MyString: "test value"},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable.

		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			dec, err := NewDecoder(bsonrw.NewBSONDocumentReader(tc.input))
			require.NoError(t, err, "NewDecoder error")

			tc.configure(dec)

			got := tc.decodeInto()
			err = dec.Decode(got)
			require.NoError(t, err, "Decode error")

			assert.Equal(t, tc.want, got, "expected and actual decode results do not match")
		})
	}

	t.Run("DefaultDocumentM top-level", func(t *testing.T) {
		t.Parallel()

		input := bsoncore.NewDocumentBuilder().
			AppendDocument("myDocument", bsoncore.NewDocumentBuilder().
				AppendString("myString", "test value").
				Build()).
			Build()

		dec, err := NewDecoder(bsonrw.NewBSONDocumentReader(input))
		require.NoError(t, err, "NewDecoder error")

		dec.DefaultDocumentM()

		var got interface{}
		err = dec.Decode(&got)
		require.NoError(t, err, "Decode error")

		want := M{
			"myDocument": M{
				"myString": "test value",
			},
		}
		assert.Equal(t, want, got, "expected and actual decode results do not match")
	})
	t.Run("DefaultDocumentD top-level", func(t *testing.T) {
		t.Parallel()

		input := bsoncore.NewDocumentBuilder().
			AppendDocument("myDocument", bsoncore.NewDocumentBuilder().
				AppendString("myString", "test value").
				Build()).
			Build()

		dec, err := NewDecoder(bsonrw.NewBSONDocumentReader(input))
		require.NoError(t, err, "NewDecoder error")

		dec.DefaultDocumentD()

		var got interface{}
		err = dec.Decode(&got)
		require.NoError(t, err, "Decode error")

		want := D{
			{Key: "myDocument", Value: D{
				{Key: "myString", Value: "test value"},
			}},
		}
		assert.Equal(t, want, got, "expected and actual decode results do not match")
	})
}
