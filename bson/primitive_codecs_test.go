package bson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/bson/bsoncore"
	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
	"github.com/mongodb/mongo-go-driver/bson/bsonrw/bsonrwtest"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

func bytesFromDoc(doc *Document) []byte {
	b, err := doc.MarshalBSON()
	if err != nil {
		panic(fmt.Errorf("Couldn't marshal BSON document: %v", err))
	}
	return b
}

func compareValues(v1, v2 *Value) bool     { return v1.Equal(v2) }
func compareElements(e1, e2 *Element) bool { return e1.Equal(e2) }

func compareDecimal128(d1, d2 decimal.Decimal128) bool {
	d1H, d1L := d1.GetBytes()
	d2H, d2L := d2.GetBytes()

	if d1H != d2H {
		return false
	}

	if d1L != d2L {
		return false
	}

	return true
}

func compareErrors(err1, err2 error) bool {
	if err1 == nil && err2 == nil {
		return true
	}

	if err1 == nil || err2 == nil {
		return false
	}

	if err1.Error() != err2.Error() {
		return false
	}

	return true
}

func TestDefaultValueEncoders(t *testing.T) {
	var pc PrimitiveCodecs

	var pjs = new(JavaScriptCode)
	*pjs = JavaScriptCode("var hello = 'world';")
	var psymbol = new(Symbol)
	*psymbol = Symbol("foobarbaz")

	var wrong = func(string, string) string { return "wrong" }

	pdatetime := new(DateTime)
	*pdatetime = DateTime(1234567890)

	type subtest struct {
		name   string
		val    interface{}
		ectx   *bsoncodec.EncodeContext
		llvrw  *bsonrwtest.ValueReaderWriter
		invoke bsonrwtest.Invoked
		err    error
	}

	testCases := []struct {
		name     string
		ve       bsoncodec.ValueEncoder
		subtests []subtest
	}{
		{
			"JavaScriptEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.JavaScriptEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{
						Name:     "JavaScriptEncodeValue",
						Types:    []interface{}{JavaScriptCode(""), (*JavaScriptCode)(nil)},
						Received: wrong,
					},
				},
				{"JavaScript", JavaScriptCode("foobar"), nil, nil, bsonrwtest.WriteJavascript, nil},
				{"*JavaScript", pjs, nil, nil, bsonrwtest.WriteJavascript, nil},
			},
		},
		{
			"SymbolEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.SymbolEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{
						Name:     "SymbolEncodeValue",
						Types:    []interface{}{Symbol(""), (*Symbol)(nil)},
						Received: wrong,
					},
				},
				{"Symbol", Symbol("foobar"), nil, nil, bsonrwtest.WriteJavascript, nil},
				{"*Symbol", psymbol, nil, nil, bsonrwtest.WriteJavascript, nil},
			},
		},
		{
			"BinaryEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.BinaryEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{
						Name:     "BinaryEncodeValue",
						Types:    []interface{}{Binary{}, (*Binary)(nil)},
						Received: wrong,
					},
				},
				{"Binary/success", Binary{Data: []byte{0x01, 0x02}, Subtype: 0xFF}, nil, nil, bsonrwtest.WriteBinaryWithSubtype, nil},
				{"*Binary/success", &Binary{Data: []byte{0x01, 0x02}, Subtype: 0xFF}, nil, nil, bsonrwtest.WriteBinaryWithSubtype, nil},
			},
		},
		{
			"UndefinedEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.UndefinedEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{
						Name:     "UndefinedEncodeValue",
						Types:    []interface{}{Undefinedv2{}, (*Undefinedv2)(nil)},
						Received: wrong,
					},
				},
				{"Undefined/success", Undefinedv2{}, nil, nil, bsonrwtest.WriteUndefined, nil},
				{"*Undefined/success", &Undefinedv2{}, nil, nil, bsonrwtest.WriteUndefined, nil},
			},
		},
		{
			"DateTimeEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.DateTimeEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{
						Name:     "DateTimeEncodeValue",
						Types:    []interface{}{DateTime(0), (*DateTime)(nil)},
						Received: wrong,
					},
				},
				{"DateTime/success", DateTime(1234567890), nil, nil, bsonrwtest.WriteDateTime, nil},
				{"*DateTime/success", pdatetime, nil, nil, bsonrwtest.WriteDateTime, nil},
			},
		},
		{
			"NullEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.NullEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{
						Name:     "NullEncodeValue",
						Types:    []interface{}{Nullv2{}, (*Nullv2)(nil)},
						Received: wrong,
					},
				},
				{"Null/success", Nullv2{}, nil, nil, bsonrwtest.WriteNull, nil},
				{"*Null/success", &Nullv2{}, nil, nil, bsonrwtest.WriteNull, nil},
			},
		},
		{
			"RegexEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.RegexEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{
						Name:     "RegexEncodeValue",
						Types:    []interface{}{Regex{}, (*Regex)(nil)},
						Received: wrong,
					},
				},
				{"Regex/success", Regex{Pattern: "foo", Options: "bar"}, nil, nil, bsonrwtest.WriteRegex, nil},
				{"*Regex/success", &Regex{Pattern: "foo", Options: "bar"}, nil, nil, bsonrwtest.WriteRegex, nil},
			},
		},
		{
			"DBPointerEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.DBPointerEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{
						Name:     "DBPointerEncodeValue",
						Types:    []interface{}{DBPointer{}, (*DBPointer)(nil)},
						Received: wrong,
					},
				},
				{
					"DBPointer/success",
					DBPointer{
						DB:      "foobar",
						Pointer: objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					},
					nil, nil, bsonrwtest.WriteDBPointer, nil,
				},
				{
					"*DBPointer/success",
					&DBPointer{
						DB:      "foobar",
						Pointer: objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					},
					nil, nil, bsonrwtest.WriteDBPointer, nil,
				},
			},
		},
		{
			"CodeWithScopeEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.CodeWithScopeEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{
						Name:     "CodeWithScopeEncodeValue",
						Types:    []interface{}{CodeWithScope{}, (*CodeWithScope)(nil)},
						Received: wrong,
					},
				},
				{
					"WriteCodeWithScope error",
					CodeWithScope{},
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("wcws error"), ErrAfter: bsonrwtest.WriteCodeWithScope},
					bsonrwtest.WriteCodeWithScope,
					errors.New("wcws error"),
				},
				{
					"CodeWithScope/success",
					CodeWithScope{
						Code:  "var hello = 'world';",
						Scope: NewDocument(),
					},
					nil, nil, bsonrwtest.WriteDocumentEnd, nil,
				},
				{
					"*CodeWithScope/success",
					&CodeWithScope{
						Code:  "var hello = 'world';",
						Scope: NewDocument(),
					},
					nil, nil, bsonrwtest.WriteDocumentEnd, nil,
				},
			},
		},
		{
			"TimestampEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.TimestampEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{
						Name:     "TimestampEncodeValue",
						Types:    []interface{}{Timestamp{}, (*Timestamp)(nil)},
						Received: wrong,
					},
				},
				{"Timestamp/success", Timestamp{T: 12345, I: 67890}, nil, nil, bsonrwtest.WriteTimestamp, nil},
				{"*Timestamp/success", &Timestamp{T: 12345, I: 67890}, nil, nil, bsonrwtest.WriteTimestamp, nil},
			},
		},
		{
			"MinKeyEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.MinKeyEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{
						Name:     "MinKeyEncodeValue",
						Types:    []interface{}{MinKeyv2{}, (*MinKeyv2)(nil)},
						Received: wrong,
					},
				},
				{"MinKey/success", MinKeyv2{}, nil, nil, bsonrwtest.WriteMinKey, nil},
				{"*MinKey/success", &MinKeyv2{}, nil, nil, bsonrwtest.WriteMinKey, nil},
			},
		},
		{
			"MaxKeyEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.MaxKeyEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{
						Name:     "MaxKeyEncodeValue",
						Types:    []interface{}{MaxKeyv2{}, (*MaxKeyv2)(nil)},
						Received: wrong,
					},
				},
				{"MaxKey/success", MaxKeyv2{}, nil, nil, bsonrwtest.WriteMaxKey, nil},
				{"*MaxKey/success", &MaxKeyv2{}, nil, nil, bsonrwtest.WriteMaxKey, nil},
			},
		},
		{
			"ValueEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.ValueEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{Name: "ValueEncodeValue", Types: []interface{}{(*Value)(nil)}, Received: wrong},
				},
				{"invalid value", &Value{}, nil, nil, bsonrwtest.Nothing, ErrUninitializedElement},
				{
					"success",
					VC.Null(),
					&bsoncodec.EncodeContext{Registry: DefaultRegistry},
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.WriteNull,
					nil,
				},
			},
		},
		{
			"ReaderEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.ReaderEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{Name: "ReaderEncodeValue", Types: []interface{}{Reader{}}, Received: wrong},
				},
				{
					"WriteDocument Error",
					Reader{},
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("wd error"), ErrAfter: bsonrwtest.WriteDocument},
					bsonrwtest.WriteDocument,
					errors.New("wd error"),
				},
				{
					"Reader.Iterator Error",
					Reader{0xFF, 0x00, 0x00, 0x00, 0x00},
					nil,
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.WriteDocument,
					errors.New("length read exceeds number of bytes available. length=5 bytes=255"),
				},
				{
					"WriteDocumentElement Error",
					Reader(bytesFromDoc(NewDocument(EC.Null("foo")))),
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("wde error"), ErrAfter: bsonrwtest.WriteDocumentElement},
					bsonrwtest.WriteDocumentElement,
					errors.New("wde error"),
				},
				{
					"encodeValue error",
					Reader(bytesFromDoc(NewDocument(EC.Null("foo")))),
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("ev error"), ErrAfter: bsonrwtest.WriteNull},
					bsonrwtest.WriteNull,
					errors.New("ev error"),
				},
				{
					"iterator error",
					Reader{0x0C, 0x00, 0x00, 0x00, 0x01, 'f', 'o', 'o', 0x00, 0x01, 0x02, 0x03},
					nil,
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.WriteDocumentElement,
					errors.New("not enough bytes available to read type. bytes=3 type=double"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, subtest := range tc.subtests {
				t.Run(subtest.name, func(t *testing.T) {
					var ec bsoncodec.EncodeContext
					if subtest.ectx != nil {
						ec = *subtest.ectx
					}
					llvrw := new(bsonrwtest.ValueReaderWriter)
					if subtest.llvrw != nil {
						llvrw = subtest.llvrw
					}
					llvrw.T = t
					err := tc.ve.EncodeValue(ec, llvrw, subtest.val)
					if !compareErrors(err, subtest.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, subtest.err)
					}
					invoked := llvrw.Invoked
					if !cmp.Equal(invoked, subtest.invoke) {
						t.Errorf("Incorrect method invoked. got %v; want %v", invoked, subtest.invoke)
					}
				})
			}
		})
	}

	t.Run("DocumentEncodeValue", func(t *testing.T) {
		t.Run("ValueEncoderError", func(t *testing.T) {
			val := bool(true)
			want := bsoncodec.ValueEncoderError{Name: "DocumentEncodeValue", Types: []interface{}{(*Document)(nil), (**Document)(nil)}, Received: val}
			got := (PrimitiveCodecs{}).DocumentEncodeValue(bsoncodec.EncodeContext{}, nil, val)
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("WriteDocument Error", func(t *testing.T) {
			want := errors.New("WriteDocument Error")
			llvrw := &bsonrwtest.ValueReaderWriter{
				T:        t,
				Err:      want,
				ErrAfter: bsonrwtest.WriteDocument,
			}
			got := (PrimitiveCodecs{}).DocumentEncodeValue(bsoncodec.EncodeContext{}, llvrw, NewDocument())
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("encodeDocument errors", func(t *testing.T) {
			ec := bsoncodec.EncodeContext{}
			err := errors.New("encodeDocument error")
			oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			badelem := &Element{}
			testCases := []struct {
				name  string
				ec    bsoncodec.EncodeContext
				llvrw *bsonrwtest.ValueReaderWriter
				doc   *Document
				err   error
			}{
				{
					"WriteDocumentElement",
					ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: errors.New("wde error"), ErrAfter: bsonrwtest.WriteDocumentElement},
					NewDocument(EC.Null("foo")),
					errors.New("wde error"),
				},
				{
					"WriteDouble", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDouble},
					NewDocument(EC.Double("foo", 3.14159)), err,
				},
				{
					"WriteString", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteString},
					NewDocument(EC.String("foo", "bar")), err,
				},
				{
					"WriteDocument (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t},
					NewDocument(EC.SubDocument("foo", NewDocument(EC.Null("bar")))),
					bsoncodec.ErrNoEncoder{Type: tDocument},
				},
				{
					"WriteArray (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t},
					NewDocument(EC.Array("foo", NewArray(VC.Null()))),
					bsoncodec.ErrNoEncoder{Type: tArray},
				},
				{
					"WriteBinary", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteBinaryWithSubtype},
					NewDocument(EC.BinaryWithSubtype("foo", []byte{0x01, 0x02, 0x03}, 0xFF)), err,
				},
				{
					"WriteUndefined", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteUndefined},
					NewDocument(EC.Undefined("foo")), err,
				},
				{
					"WriteObjectID", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteObjectID},
					NewDocument(EC.ObjectID("foo", oid)), err,
				},
				{
					"WriteBoolean", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteBoolean},
					NewDocument(EC.Boolean("foo", true)), err,
				},
				{
					"WriteDateTime", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDateTime},
					NewDocument(EC.DateTime("foo", 1234567890)), err,
				},
				{
					"WriteNull", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteNull},
					NewDocument(EC.Null("foo")), err,
				},
				{
					"WriteRegex", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteRegex},
					NewDocument(EC.Regex("foo", "bar", "baz")), err,
				},
				{
					"WriteDBPointer", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDBPointer},
					NewDocument(EC.DBPointer("foo", "bar", oid)), err,
				},
				{
					"WriteJavascript", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteJavascript},
					NewDocument(EC.JavaScript("foo", "var hello = 'world';")), err,
				},
				{
					"WriteSymbol", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteSymbol},
					NewDocument(EC.Symbol("foo", "symbolbaz")), err,
				},
				{
					"WriteCodeWithScope (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteCodeWithScope},
					NewDocument(EC.CodeWithScope("foo", "var hello = 'world';", NewDocument(EC.Null("bar")))),
					err,
				},
				{
					"WriteInt32", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteInt32},
					NewDocument(EC.Int32("foo", 12345)), err,
				},
				{
					"WriteInt64", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteInt64},
					NewDocument(EC.Int64("foo", 1234567890)), err,
				},
				{
					"WriteTimestamp", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteTimestamp},
					NewDocument(EC.Timestamp("foo", 10, 20)), err,
				},
				{
					"WriteDecimal128", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDecimal128},
					NewDocument(EC.Decimal128("foo", decimal.NewDecimal128(10, 20))), err,
				},
				{
					"WriteMinKey", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteMinKey},
					NewDocument(EC.MinKey("foo")), err,
				},
				{
					"WriteMaxKey", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteMaxKey},
					NewDocument(EC.MaxKey("foo")), err,
				},
				{
					"Invalid Type", ec,
					&bsonrwtest.ValueReaderWriter{T: t, BSONType: bsontype.Type(0)},
					NewDocument(badelem),
					ErrUninitializedElement,
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					err := (PrimitiveCodecs{}).DocumentEncodeValue(tc.ec, tc.llvrw, tc.doc)
					if !compareErrors(err, tc.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, tc.err)
					}
				})
			}
		})

		t.Run("success", func(t *testing.T) {
			oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			d128 := decimal.NewDecimal128(10, 20)
			want := NewDocument(
				EC.Double("a", 3.14159), EC.String("b", "foo"), EC.SubDocumentFromElements("c", EC.Null("aa")),
				EC.ArrayFromElements("d", VC.Null()),
				EC.BinaryWithSubtype("e", []byte{0x01, 0x02, 0x03}, 0xFF), EC.Undefined("f"),
				EC.ObjectID("g", oid), EC.Boolean("h", true), EC.DateTime("i", 1234567890), EC.Null("j"), EC.Regex("k", "foo", "abr"),
				EC.DBPointer("l", "foobar", oid), EC.JavaScript("m", "var hello = 'world';"), EC.Symbol("n", "bazqux"),
				EC.CodeWithScope("o", "var hello = 'world';", NewDocument(EC.Null("ab"))), EC.Int32("p", 12345),
				EC.Timestamp("q", 10, 20), EC.Int64("r", 1234567890), EC.Decimal128("s", d128), EC.MinKey("t"), EC.MaxKey("u"),
			)
			got := NewDocument()
			slc := make(bsonrw.SliceWriter, 0, 128)
			vw, err := bsonrw.NewBSONValueWriter(&slc)
			noerr(t, err)

			ec := bsoncodec.EncodeContext{Registry: DefaultRegistry}
			err = (PrimitiveCodecs{}).DocumentEncodeValue(ec, vw, want)
			noerr(t, err)
			got, err = ReadDocument(slc)
			noerr(t, err)
			if !got.Equal(want) {
				t.Error("Documents do not match")
				t.Errorf("\ngot :%v\nwant:%v", got, want)
			}
		})
	})

	t.Run("ArrayEncodeValue", func(t *testing.T) {
		t.Run("CodecEncodeError", func(t *testing.T) {
			val := bool(true)
			want := bsoncodec.ValueEncoderError{Name: "ArrayEncodeValue", Types: []interface{}{(*Array)(nil)}, Received: val}
			got := (PrimitiveCodecs{}).ArrayEncodeValue(bsoncodec.EncodeContext{}, nil, val)
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("WriteArray Error", func(t *testing.T) {
			want := errors.New("WriteArray Error")
			llvrw := &bsonrwtest.ValueReaderWriter{
				T:        t,
				Err:      want,
				ErrAfter: bsonrwtest.WriteArray,
			}
			got := (PrimitiveCodecs{}).ArrayEncodeValue(bsoncodec.EncodeContext{}, llvrw, NewArray())
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("encode array errors", func(t *testing.T) {
			ec := bsoncodec.EncodeContext{}
			err := errors.New("encode array error")
			oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			badval := &Value{}
			testCases := []struct {
				name  string
				ec    bsoncodec.EncodeContext
				llvrw *bsonrwtest.ValueReaderWriter
				arr   *Array
				err   error
			}{
				{
					"WriteDocumentElement",
					ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: errors.New("wde error"), ErrAfter: bsonrwtest.WriteArrayElement},
					NewArray(VC.Null()),
					errors.New("wde error"),
				},
				{
					"WriteDouble", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDouble},
					NewArray(VC.Double(3.14159)), err,
				},
				{
					"WriteString", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteString},
					NewArray(VC.String("bar")), err,
				},
				{
					"WriteDocument (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t},
					NewArray(VC.Document(NewDocument(EC.Null("bar")))),
					bsoncodec.ErrNoEncoder{Type: tDocument},
				},
				{
					"WriteArray (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t},
					NewArray(VC.Array(NewArray(VC.Null()))),
					bsoncodec.ErrNoEncoder{Type: tArray},
				},
				{
					"WriteBinary", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteBinaryWithSubtype},
					NewArray(VC.BinaryWithSubtype([]byte{0x01, 0x02, 0x03}, 0xFF)), err,
				},
				{
					"WriteUndefined", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteUndefined},
					NewArray(VC.Undefined()), err,
				},
				{
					"WriteObjectID", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteObjectID},
					NewArray(VC.ObjectID(oid)), err,
				},
				{
					"WriteBoolean", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteBoolean},
					NewArray(VC.Boolean(true)), err,
				},
				{
					"WriteDateTime", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDateTime},
					NewArray(VC.DateTime(1234567890)), err,
				},
				{
					"WriteNull", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteNull},
					NewArray(VC.Null()), err,
				},
				{
					"WriteRegex", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteRegex},
					NewArray(VC.Regex("bar", "baz")), err,
				},
				{
					"WriteDBPointer", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDBPointer},
					NewArray(VC.DBPointer("bar", oid)), err,
				},
				{
					"WriteJavascript", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteJavascript},
					NewArray(VC.JavaScript("var hello = 'world';")), err,
				},
				{
					"WriteSymbol", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteSymbol},
					NewArray(VC.Symbol("symbolbaz")), err,
				},
				{
					"WriteCodeWithScope (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteCodeWithScope},
					NewArray(VC.CodeWithScope("var hello = 'world';", NewDocument(EC.Null("bar")))),
					err,
				},
				{
					"WriteInt32", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteInt32},
					NewArray(VC.Int32(12345)), err,
				},
				{
					"WriteInt64", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteInt64},
					NewArray(VC.Int64(1234567890)), err,
				},
				{
					"WriteTimestamp", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteTimestamp},
					NewArray(VC.Timestamp(10, 20)), err,
				},
				{
					"WriteDecimal128", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDecimal128},
					NewArray(VC.Decimal128(decimal.NewDecimal128(10, 20))), err,
				},
				{
					"WriteMinKey", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteMinKey},
					NewArray(VC.MinKey()), err,
				},
				{
					"WriteMaxKey", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteMaxKey},
					NewArray(VC.MaxKey()), err,
				},
				{
					"Invalid Type", ec,
					&bsonrwtest.ValueReaderWriter{T: t, BSONType: bsontype.Type(0)},
					NewArray(badval),
					ErrUninitializedElement,
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					err := (PrimitiveCodecs{}).ArrayEncodeValue(tc.ec, tc.llvrw, tc.arr)
					if !compareErrors(err, tc.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, tc.err)
					}
				})
			}
		})

		t.Run("success", func(t *testing.T) {
			oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			d128 := decimal.NewDecimal128(10, 20)
			want := NewArray(
				VC.Double(3.14159), VC.String("foo"), VC.DocumentFromElements(EC.Null("aa")),
				VC.ArrayFromValues(VC.Null()),
				VC.BinaryWithSubtype([]byte{0x01, 0x02, 0x03}, 0xFF), VC.Undefined(),
				VC.ObjectID(oid), VC.Boolean(true), VC.DateTime(1234567890), VC.Null(), VC.Regex("foo", "abr"),
				VC.DBPointer("foobar", oid), VC.JavaScript("var hello = 'world';"), VC.Symbol("bazqux"),
				VC.CodeWithScope("var hello = 'world';", NewDocument(EC.Null("ab"))), VC.Int32(12345),
				VC.Timestamp(10, 20), VC.Int64(1234567890), VC.Decimal128(d128), VC.MinKey(), VC.MaxKey(),
			)

			ec := bsoncodec.EncodeContext{Registry: DefaultRegistry}

			slc := make(bsonrw.SliceWriter, 0, 128)
			vw, err := bsonrw.NewBSONValueWriter(&slc)
			noerr(t, err)

			dr, err := vw.WriteDocument()
			noerr(t, err)
			vr, err := dr.WriteDocumentElement("foo")
			noerr(t, err)

			err = (PrimitiveCodecs{}).ArrayEncodeValue(ec, vr, want)
			noerr(t, err)

			err = dr.WriteDocumentEnd()
			noerr(t, err)

			elem, err := Reader(slc).Lookup("foo")
			noerr(t, err)
			rgot := elem.Value().ReaderArray()
			doc, err := ReadDocument(rgot)
			noerr(t, err)
			got := ArrayFromDocument(doc)
			if !got.Equal(want) {
				t.Error("Documents do not match")
				t.Errorf("\ngot :%v\nwant:%v", got, want)
			}
		})
	})

	t.Run("success path", func(t *testing.T) {
		oid := objectid.New()
		oids := []objectid.ObjectID{objectid.New(), objectid.New(), objectid.New()}
		var str = new(string)
		*str = "bar"
		now := time.Now().Truncate(time.Millisecond)
		murl, err := url.Parse("https://mongodb.com/random-url?hello=world")
		if err != nil {
			t.Errorf("Error parsing URL: %v", err)
			t.FailNow()
		}
		decimal128, err := decimal.ParseDecimal128("1.5e10")
		if err != nil {
			t.Errorf("Error parsing decimal128: %v", err)
			t.FailNow()
		}

		testCases := []struct {
			name  string
			value interface{}
			b     []byte
			err   error
		}{
			{
				"struct{}",
				struct {
					A bool
					B int32
					C int64
					D uint16
					E uint64
					F float64
					G string
					H map[string]string
					I []byte
					K [2]string
					L struct {
						M string
					}
					O  *Document
					P  Reader
					Q  objectid.ObjectID
					T  []struct{}
					Y  json.Number
					Z  time.Time
					AA json.Number
					AB *url.URL
					AC decimal.Decimal128
					AD *time.Time
					AE testValueMarshaler
				}{
					A: true,
					B: 123,
					C: 456,
					D: 789,
					E: 101112,
					F: 3.14159,
					G: "Hello, world",
					H: map[string]string{"foo": "bar"},
					I: []byte{0x01, 0x02, 0x03},
					K: [2]string{"baz", "qux"},
					L: struct {
						M string
					}{
						M: "foobar",
					},
					O:  NewDocument(EC.Int64("countdown", 9876543210)),
					P:  Reader{0x05, 0x00, 0x00, 0x00, 0x00},
					Q:  oid,
					T:  nil,
					Y:  json.Number("5"),
					Z:  now,
					AA: json.Number("10.1"),
					AB: murl,
					AC: decimal128,
					AD: &now,
					AE: testValueMarshaler{t: TypeString, buf: bsoncore.AppendString(nil, "hello, world")},
				},
				docToBytes(NewDocument(
					EC.Boolean("a", true),
					EC.Int32("b", 123),
					EC.Int64("c", 456),
					EC.Int32("d", 789),
					EC.Int64("e", 101112),
					EC.Double("f", 3.14159),
					EC.String("g", "Hello, world"),
					EC.SubDocumentFromElements("h", EC.String("foo", "bar")),
					EC.Binary("i", []byte{0x01, 0x02, 0x03}),
					EC.ArrayFromElements("k", VC.String("baz"), VC.String("qux")),
					EC.SubDocumentFromElements("l", EC.String("m", "foobar")),
					EC.SubDocumentFromElements("o", EC.Int64("countdown", 9876543210)),
					EC.SubDocumentFromElements("p"),
					EC.ObjectID("q", oid),
					EC.Null("t"),
					EC.Int64("y", 5),
					EC.DateTime("z", now.UnixNano()/int64(time.Millisecond)),
					EC.Double("aa", 10.1),
					EC.String("ab", murl.String()),
					EC.Decimal128("ac", decimal128),
					EC.DateTime("ad", now.UnixNano()/int64(time.Millisecond)),
					EC.String("ae", "hello, world"),
				)),
				nil,
			},
			{
				"struct{[]interface{}}",
				struct {
					A []bool
					B []int32
					C []int64
					D []uint16
					E []uint64
					F []float64
					G []string
					H []map[string]string
					I [][]byte
					K [1][2]string
					L []struct {
						M string
					}
					N  [][]string
					O  []*Element
					P  []*Document
					Q  []Reader
					R  []objectid.ObjectID
					T  []struct{}
					W  []map[string]struct{}
					X  []map[string]struct{}
					Y  []map[string]struct{}
					Z  []time.Time
					AA []json.Number
					AB []*url.URL
					AC []decimal.Decimal128
					AD []*time.Time
					AE []testValueMarshaler
				}{
					A: []bool{true},
					B: []int32{123},
					C: []int64{456},
					D: []uint16{789},
					E: []uint64{101112},
					F: []float64{3.14159},
					G: []string{"Hello, world"},
					H: []map[string]string{{"foo": "bar"}},
					I: [][]byte{{0x01, 0x02, 0x03}},
					K: [1][2]string{{"baz", "qux"}},
					L: []struct {
						M string
					}{
						{
							M: "foobar",
						},
					},
					N:  [][]string{{"foo", "bar"}},
					O:  []*Element{EC.Null("N")},
					P:  []*Document{NewDocument(EC.Int64("countdown", 9876543210))},
					Q:  []Reader{{0x05, 0x00, 0x00, 0x00, 0x00}},
					R:  oids,
					T:  nil,
					W:  nil,
					X:  []map[string]struct{}{},   // Should be empty BSON Array
					Y:  []map[string]struct{}{{}}, // Should be BSON array with one element, an empty BSON SubDocument
					Z:  []time.Time{now, now},
					AA: []json.Number{json.Number("5"), json.Number("10.1")},
					AB: []*url.URL{murl},
					AC: []decimal.Decimal128{decimal128},
					AD: []*time.Time{&now, &now},
					AE: []testValueMarshaler{
						{t: TypeString, buf: bsoncore.AppendString(nil, "hello")},
						{t: TypeString, buf: bsoncore.AppendString(nil, "world")},
					},
				},
				docToBytes(NewDocument(
					EC.ArrayFromElements("a", VC.Boolean(true)),
					EC.ArrayFromElements("b", VC.Int32(123)),
					EC.ArrayFromElements("c", VC.Int64(456)),
					EC.ArrayFromElements("d", VC.Int32(789)),
					EC.ArrayFromElements("e", VC.Int64(101112)),
					EC.ArrayFromElements("f", VC.Double(3.14159)),
					EC.ArrayFromElements("g", VC.String("Hello, world")),
					EC.ArrayFromElements("h", VC.DocumentFromElements(EC.String("foo", "bar"))),
					EC.ArrayFromElements("i", VC.Binary([]byte{0x01, 0x02, 0x03})),
					EC.ArrayFromElements("k", VC.ArrayFromValues(VC.String("baz"), VC.String("qux"))),
					EC.ArrayFromElements("l", VC.DocumentFromElements(EC.String("m", "foobar"))),
					EC.ArrayFromElements("n", VC.ArrayFromValues(VC.String("foo"), VC.String("bar"))),
					EC.SubDocumentFromElements("o", EC.Null("N")),
					EC.ArrayFromElements("p", VC.DocumentFromElements(EC.Int64("countdown", 9876543210))),
					EC.ArrayFromElements("q", VC.DocumentFromElements()),
					EC.ArrayFromElements("r", VC.ObjectID(oids[0]), VC.ObjectID(oids[1]), VC.ObjectID(oids[2])),
					EC.Null("t"),
					EC.Null("w"),
					EC.Array("x", NewArray()),
					EC.ArrayFromElements("y", VC.Document(NewDocument())),
					EC.ArrayFromElements("z", VC.DateTime(now.UnixNano()/int64(time.Millisecond)), VC.DateTime(now.UnixNano()/int64(time.Millisecond))),
					EC.ArrayFromElements("aa", VC.Int64(5), VC.Double(10.10)),
					EC.ArrayFromElements("ab", VC.String(murl.String())),
					EC.ArrayFromElements("ac", VC.Decimal128(decimal128)),
					EC.ArrayFromElements("ad", VC.DateTime(now.UnixNano()/int64(time.Millisecond)), VC.DateTime(now.UnixNano()/int64(time.Millisecond))),
					EC.ArrayFromElements("ae", VC.String("hello"), VC.String("world")),
				)),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				b := make(bsonrw.SliceWriter, 0, 512)
				vw, err := bsonrw.NewBSONValueWriter(&b)
				noerr(t, err)
				enc, err := NewEncoder(DefaultRegistry, vw)
				noerr(t, err)
				err = enc.Encode(tc.value)
				if err != tc.err {
					t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
				}
				if diff := cmp.Diff([]byte(b), tc.b); diff != "" {
					t.Errorf("Bytes written differ: (-got +want)\n%s", diff)
					t.Errorf("Bytes\ngot: %v\nwant:%v\n", b, tc.b)
					t.Errorf("Readers\ngot: %v\nwant:%v\n", Reader(b), Reader(tc.b))
				}
			})
		}
	})
}

func TestDefaultValueDecoders(t *testing.T) {
	var pc PrimitiveCodecs

	var pjs = new(JavaScriptCode)
	*pjs = JavaScriptCode("var hello = 'world';")
	var psymbol = new(Symbol)
	*psymbol = Symbol("foobarbaz")

	var wrong = func(string, string) string { return "wrong" }

	const cansetreflectiontest = "cansetreflectiontest"

	type subtest struct {
		name   string
		val    interface{}
		dctx   *bsoncodec.DecodeContext
		llvrw  *bsonrwtest.ValueReaderWriter
		invoke bsonrwtest.Invoked
		err    error
	}

	testCases := []struct {
		name     string
		vd       bsoncodec.ValueDecoder
		subtests []subtest
	}{
		{
			"JavaScriptDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.JavaScriptDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.JavaScript, Return: ""},
					bsonrwtest.ReadJavascript,
					bsoncodec.ValueDecoderError{Name: "JavaScriptDecodeValue", Types: []interface{}{(*JavaScriptCode)(nil), (**JavaScriptCode)(nil)}, Received: &wrong},
				},
				{
					"type not Javascript",
					JavaScriptCode(""),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a JavaScriptPrimitive", bsontype.String),
				},
				{
					"ReadJavascript Error",
					JavaScriptCode(""),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.JavaScript, Err: errors.New("rjs error"), ErrAfter: bsonrwtest.ReadJavascript},
					bsonrwtest.ReadJavascript,
					errors.New("rjs error"),
				},
				{
					"JavaScriptCode/success",
					JavaScriptCode("var hello = 'world';"),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.JavaScript, Return: "var hello = 'world';"},
					bsonrwtest.ReadJavascript,
					nil,
				},
				{
					"*JavaScriptCode/success",
					pjs,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.JavaScript, Return: "var hello = 'world';"},
					bsonrwtest.ReadJavascript,
					nil,
				},
			},
		},
		{
			"SymbolDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.SymbolDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Symbol, Return: ""},
					bsonrwtest.ReadSymbol,
					bsoncodec.ValueDecoderError{Name: "SymbolDecodeValue", Types: []interface{}{(*Symbol)(nil), (**Symbol)(nil)}, Received: &wrong},
				},
				{
					"type not Symbol",
					Symbol(""),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a SymbolPrimitive", bsontype.String),
				},
				{
					"ReadSymbol Error",
					Symbol(""),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Symbol, Err: errors.New("rjs error"), ErrAfter: bsonrwtest.ReadSymbol},
					bsonrwtest.ReadSymbol,
					errors.New("rjs error"),
				},
				{
					"Symbol/success",
					Symbol("var hello = 'world';"),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Symbol, Return: "var hello = 'world';"},
					bsonrwtest.ReadSymbol,
					nil,
				},
				{
					"*Symbol/success",
					psymbol,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Symbol, Return: "foobarbaz"},
					bsonrwtest.ReadSymbol,
					nil,
				},
			},
		},
		{
			"BinaryDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.BinaryDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{
						BSONType: bsontype.Binary,
						Return: bsoncore.Value{
							Type: bsontype.Binary,
							Data: bsoncore.AppendBinary(nil, 0x00, []byte{0x01, 0x02, 0x3}),
						},
					},
					bsonrwtest.ReadBinary,
					bsoncodec.ValueDecoderError{Name: "BinaryDecodeValue", Types: []interface{}{(*Binary)(nil)}, Received: &wrong},
				},
				{
					"type not binary",
					Binary{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a Binary", bsontype.String),
				},
				{
					"ReadBinary Error",
					Binary{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Binary, Err: errors.New("rb error"), ErrAfter: bsonrwtest.ReadBinary},
					bsonrwtest.ReadBinary,
					errors.New("rb error"),
				},
				{
					"Binary/success",
					Binary{Data: []byte{0x01, 0x02, 0x03}, Subtype: 0xFF},
					nil,
					&bsonrwtest.ValueReaderWriter{
						BSONType: bsontype.Binary,
						Return: bsoncore.Value{
							Type: bsontype.Binary,
							Data: bsoncore.AppendBinary(nil, 0xFF, []byte{0x01, 0x02, 0x03}),
						},
					},
					bsonrwtest.ReadBinary,
					nil,
				},
				{
					"*Binary/success",
					&Binary{Data: []byte{0x01, 0x02, 0x03}, Subtype: 0xFF},
					nil,
					&bsonrwtest.ValueReaderWriter{
						BSONType: bsontype.Binary,
						Return: bsoncore.Value{
							Type: bsontype.Binary,
							Data: bsoncore.AppendBinary(nil, 0xFF, []byte{0x01, 0x02, 0x03}),
						},
					},
					bsonrwtest.ReadBinary,
					nil,
				},
			},
		},
		{
			"UndefinedDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.UndefinedDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Undefined},
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{Name: "UndefinedDecodeValue", Types: []interface{}{(*Undefinedv2)(nil)}, Received: &wrong},
				},
				{
					"type not undefined",
					Undefinedv2{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into an Undefined", bsontype.String),
				},
				{
					"ReadUndefined Error",
					Undefinedv2{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Undefined, Err: errors.New("ru error"), ErrAfter: bsonrwtest.ReadUndefined},
					bsonrwtest.ReadUndefined,
					errors.New("ru error"),
				},
				{
					"ReadUndefined/success",
					Undefinedv2{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Undefined},
					bsonrwtest.ReadUndefined,
					nil,
				},
			},
		},
		{
			"DateTimeDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.DateTimeDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.DateTime},
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{Name: "DateTimeDecodeValue", Types: []interface{}{(*DateTime)(nil)}, Received: &wrong},
				},
				{
					"type not datetime",
					DateTime(0),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a DateTime", bsontype.String),
				},
				{
					"ReadDateTime Error",
					DateTime(0),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.DateTime, Err: errors.New("rdt error"), ErrAfter: bsonrwtest.ReadDateTime},
					bsonrwtest.ReadDateTime,
					errors.New("rdt error"),
				},
				{
					"success",
					DateTime(1234567890),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.DateTime, Return: int64(1234567890)},
					bsonrwtest.ReadDateTime,
					nil,
				},
			},
		},
		{
			"NullDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.NullDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Null},
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{Name: "NullDecodeValue", Types: []interface{}{(*Nullv2)(nil)}, Received: &wrong},
				},
				{
					"type not null",
					Nullv2{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a Null", bsontype.String),
				},
				{
					"ReadNull Error",
					Nullv2{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Null, Err: errors.New("rn error"), ErrAfter: bsonrwtest.ReadNull},
					bsonrwtest.ReadNull,
					errors.New("rn error"),
				},
				{
					"success",
					Nullv2{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Null},
					bsonrwtest.ReadNull,
					nil,
				},
			},
		},
		{
			"RegexDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.RegexDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Regex},
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{Name: "RegexDecodeValue", Types: []interface{}{(*Regex)(nil)}, Received: &wrong},
				},
				{
					"type not regex",
					Regex{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a Regex", bsontype.String),
				},
				{
					"ReadRegex Error",
					Regex{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Regex, Err: errors.New("rr error"), ErrAfter: bsonrwtest.ReadRegex},
					bsonrwtest.ReadRegex,
					errors.New("rr error"),
				},
				{
					"success",
					Regex{Pattern: "foo", Options: "bar"},
					nil,
					&bsonrwtest.ValueReaderWriter{
						BSONType: bsontype.Regex,
						Return: bsoncore.Value{
							Type: bsontype.Regex,
							Data: bsoncore.AppendRegex(nil, "foo", "bar"),
						},
					},
					bsonrwtest.ReadRegex,
					nil,
				},
			},
		},
		{
			"DBPointerDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.DBPointerDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.DBPointer},
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{Name: "DBPointerDecodeValue", Types: []interface{}{(*DBPointer)(nil)}, Received: &wrong},
				},
				{
					"type not dbpointer",
					DBPointer{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a DBPointer", bsontype.String),
				},
				{
					"ReadDBPointer Error",
					DBPointer{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.DBPointer, Err: errors.New("rdbp error"), ErrAfter: bsonrwtest.ReadDBPointer},
					bsonrwtest.ReadDBPointer,
					errors.New("rdbp error"),
				},
				{
					"success",
					DBPointer{
						DB:      "foobar",
						Pointer: objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					},
					nil,
					&bsonrwtest.ValueReaderWriter{
						BSONType: bsontype.DBPointer,
						Return: bsoncore.Value{
							Type: bsontype.DBPointer,
							Data: bsoncore.AppendDBPointer(
								nil, "foobar", objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
							),
						},
					},
					bsonrwtest.ReadDBPointer,
					nil,
				},
			},
		},
		{
			"CodeWithScopeDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.CodeWithScopeDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.CodeWithScope},
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{
						Name:     "CodeWithScopeDecodeValue",
						Types:    []interface{}{(*CodeWithScope)(nil)},
						Received: &wrong,
					},
				},
				{
					"type not codewithscope",
					CodeWithScope{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a CodeWithScope", bsontype.String),
				},
				{
					"ReadCodeWithScope Error",
					CodeWithScope{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.CodeWithScope, Err: errors.New("rcws error"), ErrAfter: bsonrwtest.ReadCodeWithScope},
					bsonrwtest.ReadCodeWithScope,
					errors.New("rcws error"),
				},
				{
					"decodeDocument Error",
					CodeWithScope{
						Code:  "var hello = 'world';",
						Scope: NewDocument(EC.Null("foo")),
					},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.CodeWithScope, Err: errors.New("dd error"), ErrAfter: bsonrwtest.ReadElement},
					bsonrwtest.ReadElement,
					errors.New("dd error"),
				},
			},
		},
		{
			"TimestampDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.TimestampDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Timestamp},
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{Name: "TimestampDecodeValue", Types: []interface{}{(*Timestamp)(nil)}, Received: &wrong},
				},
				{
					"type not timestamp",
					Timestamp{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a Timestamp", bsontype.String),
				},
				{
					"ReadTimestamp Error",
					Timestamp{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Timestamp, Err: errors.New("rt error"), ErrAfter: bsonrwtest.ReadTimestamp},
					bsonrwtest.ReadTimestamp,
					errors.New("rt error"),
				},
				{
					"success",
					Timestamp{T: 12345, I: 67890},
					nil,
					&bsonrwtest.ValueReaderWriter{
						BSONType: bsontype.Timestamp,
						Return: bsoncore.Value{
							Type: bsontype.Timestamp,
							Data: bsoncore.AppendTimestamp(nil, 12345, 67890),
						},
					},
					bsonrwtest.ReadTimestamp,
					nil,
				},
			},
		},
		{
			"MinKeyDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.MinKeyDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.MinKey},
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{Name: "MinKeyDecodeValue", Types: []interface{}{(*MinKeyv2)(nil)}, Received: &wrong},
				},
				{
					"type not null",
					MinKeyv2{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a MinKey", bsontype.String),
				},
				{
					"ReadMinKey Error",
					MinKeyv2{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.MinKey, Err: errors.New("rn error"), ErrAfter: bsonrwtest.ReadMinKey},
					bsonrwtest.ReadMinKey,
					errors.New("rn error"),
				},
				{
					"success",
					MinKeyv2{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.MinKey},
					bsonrwtest.ReadMinKey,
					nil,
				},
			},
		},
		{
			"MaxKeyDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.MaxKeyDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.MaxKey},
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{Name: "MaxKeyDecodeValue", Types: []interface{}{(*MaxKeyv2)(nil)}, Received: &wrong},
				},
				{
					"type not null",
					MaxKeyv2{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a MaxKey", bsontype.String),
				},
				{
					"ReadMaxKey Error",
					MaxKeyv2{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.MaxKey, Err: errors.New("rn error"), ErrAfter: bsonrwtest.ReadMaxKey},
					bsonrwtest.ReadMaxKey,
					errors.New("rn error"),
				},
				{
					"success",
					MaxKeyv2{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.MaxKey},
					bsonrwtest.ReadMaxKey,
					nil,
				},
			},
		},
		{
			"ValueDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.ValueDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{Name: "ValueDecodeValue", Types: []interface{}{(**Value)(nil)}, Received: &wrong},
				},
				{"invalid value", (**Value)(nil), nil, nil, bsonrwtest.Nothing, errors.New("ValueDecodeValue can only be used to decode non-nil **Value")},
				{
					"success",
					VC.Double(3.14159),
					&bsoncodec.DecodeContext{Registry: NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.14159)},
					bsonrwtest.ReadDouble,
					nil,
				},
			},
		},
		{
			"ReaderDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.ReaderDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{Name: "ReaderDecodeValue", Types: []interface{}{(*Reader)(nil)}, Received: &wrong},
				},
				{
					"*Reader is nil",
					(*Reader)(nil),
					nil,
					nil,
					bsonrwtest.Nothing,
					errors.New("ReaderDecodeValue can only be used to decode non-nil *Reader"),
				},
				{
					"Copy error",
					Reader{},
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("copy error"), ErrAfter: bsonrwtest.ReadDocument},
					bsonrwtest.ReadDocument,
					errors.New("copy error"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, rc := range tc.subtests {
				t.Run(rc.name, func(t *testing.T) {
					var dc bsoncodec.DecodeContext
					if rc.dctx != nil {
						dc = *rc.dctx
					}
					llvrw := new(bsonrwtest.ValueReaderWriter)
					if rc.llvrw != nil {
						llvrw = rc.llvrw
					}
					llvrw.T = t
					var got interface{}
					if rc.val == cansetreflectiontest { // We're doing a CanSet reflection test
						err := tc.vd.DecodeValue(dc, llvrw, nil)
						if !compareErrors(err, rc.err) {
							t.Errorf("Errors do not match. got %v; want %v", err, rc.err)
						}

						val := reflect.New(reflect.TypeOf(rc.val)).Elem().Interface()
						err = tc.vd.DecodeValue(dc, llvrw, val)
						if !compareErrors(err, rc.err) {
							t.Errorf("Errors do not match. got %v; want %v", err, rc.err)
						}
						return
					}
					var unwrap bool
					rtype := reflect.TypeOf(rc.val)
					if rtype.Kind() == reflect.Ptr {
						if reflect.ValueOf(rc.val).IsNil() {
							got = rc.val
						} else {
							val := reflect.New(rtype).Elem()
							elem := reflect.New(rtype.Elem())
							val.Set(elem)
							got = val.Addr().Interface()
							unwrap = true
						}
					} else {
						unwrap = true
						got = reflect.New(reflect.TypeOf(rc.val)).Interface()
					}
					want := rc.val
					err := tc.vd.DecodeValue(dc, llvrw, got)
					if !compareErrors(err, rc.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, rc.err)
					}
					invoked := llvrw.Invoked
					if !cmp.Equal(invoked, rc.invoke) {
						t.Errorf("Incorrect method invoked. got %v; want %v", invoked, rc.invoke)
					}
					if unwrap {
						got = reflect.ValueOf(got).Elem().Interface()
					}
					if rc.err == nil && !cmp.Equal(got, want, cmp.Comparer(compareValues)) {
						t.Errorf("Values do not match. got (%T)%v; want (%T)%v", got, got, want, want)
					}
				})
			}
		})
	}

	t.Run("CodeWithScopeCodec/DecodeValue/success", func(t *testing.T) {
		dc := bsoncodec.DecodeContext{Registry: NewRegistryBuilder().Build()}
		b, err := NewDocument(EC.CodeWithScope("foo", "var hello = 'world';", NewDocument(EC.Null("bar")))).MarshalBSON()
		noerr(t, err)
		dvr := bsonrw.NewBSONValueReader(b)
		dr, err := dvr.ReadDocument()
		noerr(t, err)
		_, vr, err := dr.ReadElement()
		noerr(t, err)

		want := CodeWithScope{
			Code:  "var hello = 'world';",
			Scope: NewDocument(EC.Null("bar")),
		}
		var got CodeWithScope
		err = pc.CodeWithScopeDecodeValue(dc, vr, &got)
		noerr(t, err)

		if !cmp.Equal(got, want) {
			t.Errorf("CodeWithScopes do not match. got %v; want %v", got, want)
		}
	})
	t.Run("DocumentDecodeValue", func(t *testing.T) {
		t.Run("CodecDecodeError", func(t *testing.T) {
			val := bool(true)
			want := bsoncodec.ValueDecoderError{Name: "DocumentDecodeValue", Types: []interface{}{(**Document)(nil)}, Received: val}
			got := pc.DocumentDecodeValue(bsoncodec.DecodeContext{}, &bsonrwtest.ValueReaderWriter{BSONType: bsontype.EmbeddedDocument}, val)
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("ReadDocument Error", func(t *testing.T) {
			want := errors.New("ReadDocument Error")
			llvrw := &bsonrwtest.ValueReaderWriter{
				T:        t,
				Err:      want,
				ErrAfter: bsonrwtest.ReadDocument,
				BSONType: bsontype.EmbeddedDocument,
			}
			got := pc.DocumentDecodeValue(bsoncodec.DecodeContext{}, llvrw, new(*Document))
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("decodeDocument errors", func(t *testing.T) {
			dc := bsoncodec.DecodeContext{}
			err := errors.New("decodeDocument error")
			testCases := []struct {
				name  string
				dc    bsoncodec.DecodeContext
				llvrw *bsonrwtest.ValueReaderWriter
				err   error
			}{
				{
					"ReadElement",
					dc,
					&bsonrwtest.ValueReaderWriter{T: t, Err: errors.New("re error"), ErrAfter: bsonrwtest.ReadElement},
					errors.New("re error"),
				},
				{"ReadDouble", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadDouble, BSONType: bsontype.Double}, err},
				{"ReadString", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadString, BSONType: bsontype.String}, err},
				{
					"ReadDocument (Lookup)", bsoncodec.DecodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t, BSONType: bsontype.EmbeddedDocument},
					bsoncodec.ErrNoDecoder{Type: tDocument},
				},
				{
					"ReadArray (Lookup)", bsoncodec.DecodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t, BSONType: bsontype.Array},
					bsoncodec.ErrNoDecoder{Type: tArray},
				},
				{"ReadBinary", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadBinary, BSONType: bsontype.Binary}, err},
				{"ReadUndefined", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadUndefined, BSONType: bsontype.Undefined}, err},
				{"ReadObjectID", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadObjectID, BSONType: bsontype.ObjectID}, err},
				{"ReadBoolean", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadBoolean, BSONType: bsontype.Boolean}, err},
				{"ReadDateTime", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadDateTime, BSONType: bsontype.DateTime}, err},
				{"ReadNull", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadNull, BSONType: bsontype.Null}, err},
				{"ReadRegex", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadRegex, BSONType: bsontype.Regex}, err},
				{"ReadDBPointer", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadDBPointer, BSONType: bsontype.DBPointer}, err},
				{"ReadJavascript", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadJavascript, BSONType: bsontype.JavaScript}, err},
				{"ReadSymbol", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadSymbol, BSONType: bsontype.Symbol}, err},
				{
					"ReadCodeWithScope (Lookup)", bsoncodec.DecodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadCodeWithScope, BSONType: bsontype.CodeWithScope},
					err,
				},
				{"ReadInt32", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadInt32, BSONType: bsontype.Int32}, err},
				{"ReadInt64", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadInt64, BSONType: bsontype.Int64}, err},
				{"ReadTimestamp", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadTimestamp, BSONType: bsontype.Timestamp}, err},
				{"ReadDecimal128", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadDecimal128, BSONType: bsontype.Decimal128}, err},
				{"ReadMinKey", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadMinKey, BSONType: bsontype.MinKey}, err},
				{"ReadMaxKey", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadMaxKey, BSONType: bsontype.MaxKey}, err},
				{"Invalid Type", dc, &bsonrwtest.ValueReaderWriter{T: t, BSONType: bsontype.Type(0)}, fmt.Errorf("Cannot read unknown BSON type %s", bsontype.Type(0))},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					err := pc.decodeDocument(tc.dc, tc.llvrw, new(*Document))
					if !compareErrors(err, tc.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, tc.err)
					}
				})
			}
		})

		t.Run("success", func(t *testing.T) {
			oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			d128 := decimal.NewDecimal128(10, 20)
			want := NewDocument(
				EC.Double("a", 3.14159), EC.String("b", "foo"), EC.SubDocumentFromElements("c", EC.Null("aa")),
				EC.ArrayFromElements("d", VC.Null()),
				EC.BinaryWithSubtype("e", []byte{0x01, 0x02, 0x03}, 0xFF), EC.Undefined("f"),
				EC.ObjectID("g", oid), EC.Boolean("h", true), EC.DateTime("i", 1234567890), EC.Null("j"), EC.Regex("k", "foo", "bar"),
				EC.DBPointer("l", "foobar", oid), EC.JavaScript("m", "var hello = 'world';"), EC.Symbol("n", "bazqux"),
				EC.CodeWithScope("o", "var hello = 'world';", NewDocument(EC.Null("ab"))), EC.Int32("p", 12345),
				EC.Timestamp("q", 10, 20), EC.Int64("r", 1234567890), EC.Decimal128("s", d128), EC.MinKey("t"), EC.MaxKey("u"),
			)
			var got *Document
			dc := bsoncodec.DecodeContext{Registry: NewRegistryBuilder().Build()}
			b, err := want.MarshalBSON()
			noerr(t, err)
			err = pc.DocumentDecodeValue(dc, bsonrw.NewBSONValueReader(b), &got)
			noerr(t, err)
			if !got.Equal(want) {
				t.Error("Documents do not match")
				t.Errorf("\ngot :%v\nwant:%v", got, want)
			}
		})
	})
	t.Run("ArrayDecodeValue", func(t *testing.T) {
		t.Run("CodecDecodeError", func(t *testing.T) {
			val := bool(true)
			want := bsoncodec.ValueDecoderError{Name: "ArrayDecodeValue", Types: []interface{}{(**Array)(nil)}, Received: val}
			got := pc.ArrayDecodeValue(bsoncodec.DecodeContext{}, &bsonrwtest.ValueReaderWriter{BSONType: bsontype.Array}, val)
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("ReadArray Error", func(t *testing.T) {
			want := errors.New("ReadArray Error")
			llvrw := &bsonrwtest.ValueReaderWriter{
				T:        t,
				Err:      want,
				ErrAfter: bsonrwtest.ReadArray,
				BSONType: bsontype.Array,
			}
			got := pc.ArrayDecodeValue(bsoncodec.DecodeContext{}, llvrw, new(*Array))
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("decode array errors", func(t *testing.T) {
			dc := bsoncodec.DecodeContext{}
			err := errors.New("decode array error")
			testCases := []struct {
				name  string
				dc    bsoncodec.DecodeContext
				llvrw *bsonrwtest.ValueReaderWriter
				err   error
			}{
				{
					"Returnue",
					dc,
					&bsonrwtest.ValueReaderWriter{T: t, Err: errors.New("re error"), ErrAfter: bsonrwtest.ReadValue},
					errors.New("re error"),
				},
				{"ReadDouble", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadDouble, BSONType: bsontype.Double}, err},
				{"ReadString", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadString, BSONType: bsontype.String}, err},
				{
					"ReadDocument (Lookup)", bsoncodec.DecodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t, BSONType: bsontype.EmbeddedDocument},
					bsoncodec.ErrNoDecoder{Type: tDocument},
				},
				{
					"ReadArray (Lookup)", bsoncodec.DecodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t, BSONType: bsontype.Array},
					bsoncodec.ErrNoDecoder{Type: tArray},
				},
				{"ReadBinary", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadBinary, BSONType: bsontype.Binary}, err},
				{"ReadUndefined", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadUndefined, BSONType: bsontype.Undefined}, err},
				{"ReadObjectID", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadObjectID, BSONType: bsontype.ObjectID}, err},
				{"ReadBoolean", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadBoolean, BSONType: bsontype.Boolean}, err},
				{"ReadDateTime", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadDateTime, BSONType: bsontype.DateTime}, err},
				{"ReadNull", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadNull, BSONType: bsontype.Null}, err},
				{"ReadRegex", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadRegex, BSONType: bsontype.Regex}, err},
				{"ReadDBPointer", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadDBPointer, BSONType: bsontype.DBPointer}, err},
				{"ReadJavascript", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadJavascript, BSONType: bsontype.JavaScript}, err},
				{"ReadSymbol", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadSymbol, BSONType: bsontype.Symbol}, err},
				{
					"ReadCodeWithScope (Lookup)", bsoncodec.DecodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadCodeWithScope, BSONType: bsontype.CodeWithScope},
					err,
				},
				{"ReadInt32", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadInt32, BSONType: bsontype.Int32}, err},
				{"ReadInt64", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadInt64, BSONType: bsontype.Int64}, err},
				{"ReadTimestamp", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadTimestamp, BSONType: bsontype.Timestamp}, err},
				{"ReadDecimal128", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadDecimal128, BSONType: bsontype.Decimal128}, err},
				{"ReadMinKey", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadMinKey, BSONType: bsontype.MinKey}, err},
				{"ReadMaxKey", dc, &bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.ReadMaxKey, BSONType: bsontype.MaxKey}, err},
				{"Invalid Type", dc, &bsonrwtest.ValueReaderWriter{T: t, BSONType: bsontype.Type(0)}, fmt.Errorf("Cannot read unknown BSON type %s", bsontype.Type(0))},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					err := pc.ArrayDecodeValue(tc.dc, tc.llvrw, new(*Array))
					if !compareErrors(err, tc.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, tc.err)
					}
				})
			}
		})

		t.Run("success", func(t *testing.T) {
			oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			d128 := decimal.NewDecimal128(10, 20)
			want := NewArray(
				VC.Double(3.14159), VC.String("foo"), VC.DocumentFromElements(EC.Null("aa")),
				VC.ArrayFromValues(VC.Null()),
				VC.BinaryWithSubtype([]byte{0x01, 0x02, 0x03}, 0xFF), VC.Undefined(),
				VC.ObjectID(oid), VC.Boolean(true), VC.DateTime(1234567890), VC.Null(), VC.Regex("foo", "bar"),
				VC.DBPointer("foobar", oid), VC.JavaScript("var hello = 'world';"), VC.Symbol("bazqux"),
				VC.CodeWithScope("var hello = 'world';", NewDocument(EC.Null("ab"))), VC.Int32(12345),
				VC.Timestamp(10, 20), VC.Int64(1234567890), VC.Decimal128(d128), VC.MinKey(), VC.MaxKey(),
			)
			dc := bsoncodec.DecodeContext{Registry: NewRegistryBuilder().Build()}

			b, err := NewDocument(EC.Array("", want)).MarshalBSON()
			noerr(t, err)
			dvr := bsonrw.NewBSONValueReader(b)
			dr, err := dvr.ReadDocument()
			noerr(t, err)
			_, vr, err := dr.ReadElement()
			noerr(t, err)

			var got *Array
			err = pc.ArrayDecodeValue(dc, vr, &got)
			noerr(t, err)
			if !got.Equal(want) {
				t.Error("Documents do not match")
				t.Errorf("\ngot :%v\nwant:%v", got, want)
			}
		})
	})

	t.Run("success path", func(t *testing.T) {
		oid := objectid.New()
		oids := []objectid.ObjectID{objectid.New(), objectid.New(), objectid.New()}
		var str = new(string)
		*str = "bar"
		now := time.Now().Truncate(time.Millisecond)
		murl, err := url.Parse("https://mongodb.com/random-url?hello=world")
		if err != nil {
			t.Errorf("Error parsing URL: %v", err)
			t.FailNow()
		}
		decimal128, err := decimal.ParseDecimal128("1.5e10")
		if err != nil {
			t.Errorf("Error parsing decimal128: %v", err)
			t.FailNow()
		}

		testCases := []struct {
			name  string
			value interface{}
			b     []byte
			err   error
		}{
			{
				"map[string]int",
				map[string]int32{"foo": 1},
				[]byte{
					0x0E, 0x00, 0x00, 0x00,
					0x10, 'f', 'o', 'o', 0x00,
					0x01, 0x00, 0x00, 0x00,
					0x00,
				},
				nil,
			},
			{
				"map[string]objectid.ObjectID",
				map[string]objectid.ObjectID{"foo": oid},
				docToBytes(NewDocument(EC.ObjectID("foo", oid))),
				nil,
			},
			{
				"map[string][]*Element",
				map[string][]*Element{"Z": {EC.Int32("A", 1), EC.Int32("B", 2), EC.Int32("EC", 3)}},
				docToBytes(NewDocument(
					EC.SubDocumentFromElements("Z", EC.Int32("A", 1), EC.Int32("B", 2), EC.Int32("EC", 3)),
				)),
				nil,
			},
			{
				"map[string][]*Value",
				map[string][]*Value{"Z": {VC.Int32(1), VC.Int32(2), VC.Int32(3)}},
				docToBytes(NewDocument(
					EC.ArrayFromElements("Z", VC.Int32(1), VC.Int32(2), VC.Int32(3)),
				)),
				nil,
			},
			{
				"map[string]*Document",
				map[string]*Document{"Z": NewDocument(EC.Null("foo"))},
				docToBytes(NewDocument(
					EC.SubDocumentFromElements("Z", EC.Null("foo")),
				)),
				nil,
			},
			{
				"map[string]Reader",
				map[string]Reader{"Z": {0x05, 0x00, 0x00, 0x00, 0x00}},
				docToBytes(NewDocument(
					EC.SubDocumentFromReader("Z", Reader{0x05, 0x00, 0x00, 0x00, 0x00}),
				)),
				nil,
			},
			{
				"map[string][]int32",
				map[string][]int32{"Z": {1, 2, 3}},
				docToBytes(NewDocument(
					EC.ArrayFromElements("Z", VC.Int32(1), VC.Int32(2), VC.Int32(3)),
				)),
				nil,
			},
			{
				"map[string][]objectid.ObjectID",
				map[string][]objectid.ObjectID{"Z": oids},
				docToBytes(NewDocument(
					EC.ArrayFromElements("Z", VC.ObjectID(oids[0]), VC.ObjectID(oids[1]), VC.ObjectID(oids[2])),
				)),
				nil,
			},
			{
				"map[string][]json.Number(int64)",
				map[string][]json.Number{"Z": {json.Number("5"), json.Number("10")}},
				docToBytes(NewDocument(
					EC.ArrayFromElements("Z", VC.Int64(5), VC.Int64(10)),
				)),
				nil,
			},
			{
				"map[string][]json.Number(float64)",
				map[string][]json.Number{"Z": {json.Number("5"), json.Number("10.1")}},
				docToBytes(NewDocument(
					EC.ArrayFromElements("Z", VC.Int64(5), VC.Double(10.1)),
				)),
				nil,
			},
			{
				"map[string][]*url.URL",
				map[string][]*url.URL{"Z": {murl}},
				docToBytes(NewDocument(
					EC.ArrayFromElements("Z", VC.String(murl.String())),
				)),
				nil,
			},
			{
				"map[string][]decimal.Decimal128",
				map[string][]decimal.Decimal128{"Z": {decimal128}},
				docToBytes(NewDocument(
					EC.ArrayFromElements("Z", VC.Decimal128(decimal128)),
				)),
				nil,
			},
			{
				"-",
				struct {
					A string `bson:"-"`
				}{
					A: "",
				},
				docToBytes(NewDocument()),
				nil,
			},
			{
				"omitempty",
				struct {
					A string `bson:",omitempty"`
				}{
					A: "",
				},
				docToBytes(NewDocument()),
				nil,
			},
			{
				"omitempty, empty time",
				struct {
					A time.Time `bson:",omitempty"`
				}{
					A: time.Time{},
				},
				docToBytes(NewDocument()),
				nil,
			},
			{
				"no private fields",
				noPrivateFields{a: "should be empty"},
				docToBytes(NewDocument()),
				nil,
			},
			{
				"minsize",
				struct {
					A int64 `bson:",minsize"`
				}{
					A: 12345,
				},
				docToBytes(NewDocument(EC.Int32("a", 12345))),
				nil,
			},
			{
				"inline",
				struct {
					Foo struct {
						A int64 `bson:",minsize"`
					} `bson:",inline"`
				}{
					Foo: struct {
						A int64 `bson:",minsize"`
					}{
						A: 12345,
					},
				},
				docToBytes(NewDocument(EC.Int32("a", 12345))),
				nil,
			},
			{
				"inline map",
				struct {
					Foo map[string]string `bson:",inline"`
				}{
					Foo: map[string]string{"foo": "bar"},
				},
				docToBytes(NewDocument(EC.String("foo", "bar"))),
				nil,
			},
			{
				"alternate name bson:name",
				struct {
					A string `bson:"foo"`
				}{
					A: "bar",
				},
				docToBytes(NewDocument(EC.String("foo", "bar"))),
				nil,
			},
			{
				"alternate name",
				struct {
					A string `bson:"foo"`
				}{
					A: "bar",
				},
				docToBytes(NewDocument(EC.String("foo", "bar"))),
				nil,
			},
			{
				"inline, omitempty",
				struct {
					A   string
					Foo zeroTest `bson:"omitempty,inline"`
				}{
					A:   "bar",
					Foo: zeroTest{true},
				},
				docToBytes(NewDocument(EC.String("a", "bar"))),
				nil,
			},
			{
				"struct{}",
				struct {
					A bool
					B int32
					C int64
					D uint16
					E uint64
					F float64
					G string
					H map[string]string
					I []byte
					K [2]string
					L struct {
						M string
					}
					O  *Document
					P  Reader
					Q  objectid.ObjectID
					T  []struct{}
					Y  json.Number
					Z  time.Time
					AA json.Number
					AB *url.URL
					AC decimal.Decimal128
					AD *time.Time
					AE *testValueUnmarshaler
				}{
					A: true,
					B: 123,
					C: 456,
					D: 789,
					E: 101112,
					F: 3.14159,
					G: "Hello, world",
					H: map[string]string{"foo": "bar"},
					I: []byte{0x01, 0x02, 0x03},
					K: [2]string{"baz", "qux"},
					L: struct {
						M string
					}{
						M: "foobar",
					},
					O:  NewDocument(EC.Int64("countdown", 9876543210)),
					P:  Reader{0x05, 0x00, 0x00, 0x00, 0x00},
					Q:  oid,
					T:  nil,
					Y:  json.Number("5"),
					Z:  now,
					AA: json.Number("10.1"),
					AB: murl,
					AC: decimal128,
					AD: &now,
					AE: &testValueUnmarshaler{t: bsontype.String, val: bsoncore.AppendString(nil, "hello, world!")},
				},
				docToBytes(NewDocument(
					EC.Boolean("a", true),
					EC.Int32("b", 123),
					EC.Int64("c", 456),
					EC.Int32("d", 789),
					EC.Int64("e", 101112),
					EC.Double("f", 3.14159),
					EC.String("g", "Hello, world"),
					EC.SubDocumentFromElements("h", EC.String("foo", "bar")),
					EC.Binary("i", []byte{0x01, 0x02, 0x03}),
					EC.ArrayFromElements("k", VC.String("baz"), VC.String("qux")),
					EC.SubDocumentFromElements("l", EC.String("m", "foobar")),
					EC.SubDocumentFromElements("o", EC.Int64("countdown", 9876543210)),
					EC.SubDocumentFromElements("p"),
					EC.ObjectID("q", oid),
					EC.Null("t"),
					EC.Int64("y", 5),
					EC.DateTime("z", now.UnixNano()/int64(time.Millisecond)),
					EC.Double("aa", 10.1),
					EC.String("ab", murl.String()),
					EC.Decimal128("ac", decimal128),
					EC.DateTime("ad", now.UnixNano()/int64(time.Millisecond)),
					EC.String("ae", "hello, world!"),
				)),
				nil,
			},
			{
				"struct{[]interface{}}",
				struct {
					A []bool
					B []int32
					C []int64
					D []uint16
					E []uint64
					F []float64
					G []string
					H []map[string]string
					I [][]byte
					K [1][2]string
					L []struct {
						M string
					}
					N  [][]string
					O  []*Element
					P  []*Document
					Q  []Reader
					R  []objectid.ObjectID
					T  []struct{}
					W  []map[string]struct{}
					X  []map[string]struct{}
					Y  []map[string]struct{}
					Z  []time.Time
					AA []json.Number
					AB []*url.URL
					AC []decimal.Decimal128
					AD []*time.Time
					AE []*testValueUnmarshaler
				}{
					A: []bool{true},
					B: []int32{123},
					C: []int64{456},
					D: []uint16{789},
					E: []uint64{101112},
					F: []float64{3.14159},
					G: []string{"Hello, world"},
					H: []map[string]string{{"foo": "bar"}},
					I: [][]byte{{0x01, 0x02, 0x03}},
					K: [1][2]string{{"baz", "qux"}},
					L: []struct {
						M string
					}{
						{
							M: "foobar",
						},
					},
					N:  [][]string{{"foo", "bar"}},
					O:  []*Element{EC.Null("N")},
					P:  []*Document{NewDocument(EC.Int64("countdown", 9876543210))},
					Q:  []Reader{{0x05, 0x00, 0x00, 0x00, 0x00}},
					R:  oids,
					T:  nil,
					W:  nil,
					X:  []map[string]struct{}{},   // Should be empty BSON Array
					Y:  []map[string]struct{}{{}}, // Should be BSON array with one element, an empty BSON SubDocument
					Z:  []time.Time{now, now},
					AA: []json.Number{json.Number("5"), json.Number("10.1")},
					AB: []*url.URL{murl},
					AC: []decimal.Decimal128{decimal128},
					AD: []*time.Time{&now, &now},
					AE: []*testValueUnmarshaler{
						{t: bsontype.String, val: bsoncore.AppendString(nil, "hello")},
						{t: bsontype.String, val: bsoncore.AppendString(nil, "world")},
					},
				},
				docToBytes(NewDocument(
					EC.ArrayFromElements("a", VC.Boolean(true)),
					EC.ArrayFromElements("b", VC.Int32(123)),
					EC.ArrayFromElements("c", VC.Int64(456)),
					EC.ArrayFromElements("d", VC.Int32(789)),
					EC.ArrayFromElements("e", VC.Int64(101112)),
					EC.ArrayFromElements("f", VC.Double(3.14159)),
					EC.ArrayFromElements("g", VC.String("Hello, world")),
					EC.ArrayFromElements("h", VC.DocumentFromElements(EC.String("foo", "bar"))),
					EC.ArrayFromElements("i", VC.Binary([]byte{0x01, 0x02, 0x03})),
					EC.ArrayFromElements("k", VC.ArrayFromValues(VC.String("baz"), VC.String("qux"))),
					EC.ArrayFromElements("l", VC.DocumentFromElements(EC.String("m", "foobar"))),
					EC.ArrayFromElements("n", VC.ArrayFromValues(VC.String("foo"), VC.String("bar"))),
					EC.SubDocumentFromElements("o", EC.Null("N")),
					EC.ArrayFromElements("p", VC.DocumentFromElements(EC.Int64("countdown", 9876543210))),
					EC.ArrayFromElements("q", VC.DocumentFromElements()),
					EC.ArrayFromElements("r", VC.ObjectID(oids[0]), VC.ObjectID(oids[1]), VC.ObjectID(oids[2])),
					EC.Null("t"),
					EC.Null("w"),
					EC.Array("x", NewArray()),
					EC.ArrayFromElements("y", VC.Document(NewDocument())),
					EC.ArrayFromElements("z", VC.DateTime(now.UnixNano()/int64(time.Millisecond)), VC.DateTime(now.UnixNano()/int64(time.Millisecond))),
					EC.ArrayFromElements("aa", VC.Int64(5), VC.Double(10.10)),
					EC.ArrayFromElements("ab", VC.String(murl.String())),
					EC.ArrayFromElements("ac", VC.Decimal128(decimal128)),
					EC.ArrayFromElements("ad", VC.DateTime(now.UnixNano()/int64(time.Millisecond)), VC.DateTime(now.UnixNano()/int64(time.Millisecond))),
					EC.ArrayFromElements("ae", VC.String("hello"), VC.String("world")),
				)),
				nil,
			},
		}

		t.Run("Decode", func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					vr := bsonrw.NewBSONValueReader(tc.b)
					dec, err := NewDecoder(DefaultRegistry, vr)
					noerr(t, err)
					gotVal := reflect.New(reflect.TypeOf(tc.value))
					err = dec.Decode(gotVal.Interface())
					noerr(t, err)
					got := gotVal.Elem().Interface()
					want := tc.value
					if diff := cmp.Diff(
						got, want,
						cmp.Comparer(compareElements),
						cmp.Comparer(compareDecimal128),
						cmp.Comparer(compareValues),
						cmp.Comparer(compareNoPrivateFields),
						cmp.Comparer(compareZeroTest),
					); diff != "" {
						t.Errorf("difference:\n%s", diff)
						t.Errorf("Values are not equal.\ngot: %#v\nwant:%#v", got, want)
					}
				})
			}
		})
	})

	t.Run("EmptyInterfaceDecodeValue", func(t *testing.T) {
		t.Run("DecodeValue", func(t *testing.T) {
			testCases := []struct {
				name     string
				val      interface{}
				bsontype bsontype.Type
			}{
				{
					"Double - float64",
					float64(3.14159),
					bsontype.Double,
				},
				{
					"String - string",
					string("foo bar baz"),
					bsontype.String,
				},
				{
					"Embedded Document - *Document",
					NewDocument(EC.Null("foo")),
					bsontype.EmbeddedDocument,
				},
				{
					"Array - *Array",
					NewArray(VC.Double(3.14159)),
					bsontype.Array,
				},
				{
					"Binary - Binary",
					Binary{Subtype: 0xFF, Data: []byte{0x01, 0x02, 0x03}},
					bsontype.Binary,
				},
				{
					"Undefined - Undefined",
					Undefinedv2{},
					bsontype.Undefined,
				},
				{
					"ObjectID - objectid.ObjectID",
					objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					bsontype.ObjectID,
				},
				{
					"Boolean - bool",
					bool(true),
					bsontype.Boolean,
				},
				{
					"DateTime - DateTime",
					DateTime(1234567890),
					bsontype.DateTime,
				},
				{
					"Null - Null",
					Nullv2{},
					bsontype.Null,
				},
				{
					"Regex - Regex",
					Regex{Pattern: "foo", Options: "bar"},
					bsontype.Regex,
				},
				{
					"DBPointer - DBPointer",
					DBPointer{
						DB:      "foobar",
						Pointer: objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					},
					bsontype.DBPointer,
				},
				{
					"JavaScript - JavaScriptCode",
					JavaScriptCode("var foo = 'bar';"),
					bsontype.JavaScript,
				},
				{
					"Symbol - Symbol",
					Symbol("foobarbazlolz"),
					bsontype.Symbol,
				},
				{
					"CodeWithScope - CodeWithScope",
					CodeWithScope{
						Code:  "var foo = 'bar';",
						Scope: NewDocument(EC.Double("foo", 3.14159)),
					},
					bsontype.CodeWithScope,
				},
				{
					"Int32 - int32",
					int32(123456),
					bsontype.Int32,
				},
				{
					"Int64 - int64",
					int64(1234567890),
					bsontype.Int64,
				},
				{
					"Timestamp - Timestamp",
					Timestamp{T: 12345, I: 67890},
					bsontype.Timestamp,
				},
				{
					"Decimal128 - decimal.Decimal128",
					decimal.NewDecimal128(12345, 67890),
					bsontype.Decimal128,
				},
				{
					"MinKey - MinKey",
					MinKeyv2{},
					bsontype.MinKey,
				},
				{
					"MaxKey - MaxKey",
					MaxKeyv2{},
					bsontype.MaxKey,
				},
			}
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					llvr := &bsonrwtest.ValueReaderWriter{BSONType: tc.bsontype}

					t.Run("Lookup failure", func(t *testing.T) {
						val := new(interface{})
						dc := bsoncodec.DecodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()}
						want := bsoncodec.ErrNoDecoder{Type: reflect.TypeOf(tc.val)}
						got := pc.EmptyInterfaceDecodeValue(dc, llvr, val)
						if !compareErrors(got, want) {
							t.Errorf("Errors are not equal. got %v; want %v", got, want)
						}
					})

					t.Run("DecodeValue failure", func(t *testing.T) {
						want := errors.New("DecodeValue failure error")
						llc := &llCodec{t: t, err: want}
						dc := bsoncodec.DecodeContext{
							Registry: bsoncodec.NewRegistryBuilder().RegisterDecoder(reflect.TypeOf(tc.val), llc).Build(),
						}
						got := pc.EmptyInterfaceDecodeValue(dc, llvr, new(interface{}))
						if !compareErrors(got, want) {
							t.Errorf("Errors are not equal. got %v; want %v", got, want)
						}
					})

					t.Run("Success", func(t *testing.T) {
						want := tc.val
						llc := &llCodec{t: t, decodeval: tc.val}
						dc := bsoncodec.DecodeContext{
							Registry: bsoncodec.NewRegistryBuilder().RegisterDecoder(reflect.TypeOf(tc.val), llc).Build(),
						}
						got := new(interface{})
						err := pc.EmptyInterfaceDecodeValue(dc, llvr, got)
						noerr(t, err)
						if !cmp.Equal(*got, want, cmp.Comparer(compareDecimal128)) {
							t.Errorf("Did not receive expected value. got %v; want %v", *got, want)
						}
					})
				})
			}
		})

		t.Run("non-*interface{}", func(t *testing.T) {
			val := uint64(1234567890)
			want := fmt.Errorf("EmptyInterfaceDecodeValue can only be used to decode non-nil *interface{} values, provided type if %T", &val)
			got := pc.EmptyInterfaceDecodeValue(bsoncodec.DecodeContext{}, nil, &val)
			if !compareErrors(got, want) {
				t.Errorf("Errors are not equal. got %v; want %v", got, want)
			}
		})

		t.Run("nil *interface{}", func(t *testing.T) {
			var val *interface{}
			want := fmt.Errorf("EmptyInterfaceDecodeValue can only be used to decode non-nil *interface{} values, provided type if %T", val)
			got := pc.EmptyInterfaceDecodeValue(bsoncodec.DecodeContext{}, nil, val)
			if !compareErrors(got, want) {
				t.Errorf("Errors are not equal. got %v; want %v", got, want)
			}
		})

		t.Run("unknown BSON type", func(t *testing.T) {
			llvr := &bsonrwtest.ValueReaderWriter{BSONType: bsontype.Type(0)}
			want := fmt.Errorf("Type %s is not a valid BSON type and has no default Go type to decode into", bsontype.Type(0))
			got := pc.EmptyInterfaceDecodeValue(bsoncodec.DecodeContext{}, llvr, new(interface{}))
			if !compareErrors(got, want) {
				t.Errorf("Errors are not equal. got %v; want %v", got, want)
			}
		})
	})

}

type testValueMarshaler struct {
	t   bsontype.Type
	buf []byte
	err error
}

func (tvm testValueMarshaler) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return tvm.t, tvm.buf, tvm.err
}

type testValueUnmarshaler struct {
	t   bsontype.Type
	val []byte
	err error
}

func (tvu *testValueUnmarshaler) UnmarshalBSONValue(t bsontype.Type, val []byte) error {
	tvu.t, tvu.val = t, val
	return tvu.err
}
func (tvu testValueUnmarshaler) Equal(tvu2 testValueUnmarshaler) bool {
	return tvu.t == tvu2.t && bytes.Equal(tvu.val, tvu2.val)
}

type noPrivateFields struct {
	a string
}

func compareNoPrivateFields(npf1, npf2 noPrivateFields) bool {
	return npf1.a != npf2.a // We don't want these to be equal
}

type zeroTest struct {
	reportZero bool
}

func (z zeroTest) IsZero() bool { return z.reportZero }

func compareZeroTest(_, _ zeroTest) bool { return true }

type nonZeroer struct {
	value bool
}

type llCodec struct {
	t         *testing.T
	decodeval interface{}
	encodeval interface{}
	err       error
}

func (llc *llCodec) EncodeValue(_ bsoncodec.EncodeContext, _ bsonrw.ValueWriter, i interface{}) error {
	if llc.err != nil {
		return llc.err
	}

	llc.encodeval = i
	return nil
}

func (llc *llCodec) DecodeValue(_ bsoncodec.DecodeContext, _ bsonrw.ValueReader, i interface{}) error {
	if llc.err != nil {
		return llc.err
	}

	val := reflect.ValueOf(i)
	if val.Type().Kind() != reflect.Ptr {
		llc.t.Errorf("Value provided to DecodeValue must be a pointer, but got %T", i)
		return nil
	}

	switch val.Type() {
	case tDocument:
		decodeval, ok := llc.decodeval.(*Document)
		if !ok {
			llc.t.Errorf("decodeval must be a *Document if the i is a *Document. decodeval %T", llc.decodeval)
			return nil
		}

		doc := i.(*Document)
		doc.Reset()
		err := doc.Concat(decodeval)
		if err != nil {
			llc.t.Errorf("could not concatenate the decoded val to doc: %v", err)
			return err
		}

		return nil
	case tArray:
		decodeval, ok := llc.decodeval.(*Array)
		if !ok {
			llc.t.Errorf("decodeval must be a *Array if the i is a *Array. decodeval %T", llc.decodeval)
			return nil
		}

		arr := i.(*Array)
		arr.Reset()
		err := arr.Concat(decodeval)
		if err != nil {
			llc.t.Errorf("could not concatenate the decoded val to array: %v", err)
			return err
		}

		return nil
	}

	if !reflect.TypeOf(llc.decodeval).AssignableTo(val.Type().Elem()) {
		llc.t.Errorf("decodeval must be assignable to i provided to DecodeValue, but is not. decodeval %T; i %T", llc.decodeval, i)
		return nil
	}

	val.Elem().Set(reflect.ValueOf(llc.decodeval))
	return nil
}
