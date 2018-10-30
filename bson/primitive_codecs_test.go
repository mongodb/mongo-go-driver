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
	"github.com/mongodb/mongo-go-driver/bson/primitive"
)

func bytesFromDoc(doc *Documentv2) []byte {
	b, err := doc.MarshalBSON()
	if err != nil {
		panic(fmt.Errorf("Couldn't marshal BSON document: %v", err))
	}
	return b
}

func compareValues(v1, v2 Valuev2) bool     { return v1.Equal(v2) }
func compareElements(e1, e2 Elementv2) bool { return e1.Equal(e2) }

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

	var pjs = new(primitive.JavaScript)
	*pjs = primitive.JavaScript("var hello = 'world';")
	var psymbol = new(primitive.Symbol)
	*psymbol = primitive.Symbol("foobarbaz")

	var wrong = func(string, string) string { return "wrong" }

	pdatetime := new(primitive.DateTime)
	*pdatetime = primitive.DateTime(1234567890)

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
						Types:    []interface{}{primitive.JavaScript(""), (*primitive.JavaScript)(nil)},
						Received: wrong,
					},
				},
				{"JavaScript", primitive.JavaScript("foobar"), nil, nil, bsonrwtest.WriteJavascript, nil},
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
						Types:    []interface{}{primitive.Symbol(""), (*primitive.Symbol)(nil)},
						Received: wrong,
					},
				},
				{"Symbol", primitive.Symbol("foobar"), nil, nil, bsonrwtest.WriteJavascript, nil},
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
						Types:    []interface{}{primitive.Binary{}, (*primitive.Binary)(nil)},
						Received: wrong,
					},
				},
				{"Binary/success", primitive.Binary{Data: []byte{0x01, 0x02}, Subtype: 0xFF}, nil, nil, bsonrwtest.WriteBinaryWithSubtype, nil},
				{"*Binary/success", &primitive.Binary{Data: []byte{0x01, 0x02}, Subtype: 0xFF}, nil, nil, bsonrwtest.WriteBinaryWithSubtype, nil},
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
						Types:    []interface{}{primitive.Undefined{}, (*primitive.Undefined)(nil)},
						Received: wrong,
					},
				},
				{"Undefined/success", primitive.Undefined{}, nil, nil, bsonrwtest.WriteUndefined, nil},
				{"*Undefined/success", &primitive.Undefined{}, nil, nil, bsonrwtest.WriteUndefined, nil},
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
						Types:    []interface{}{primitive.DateTime(0), (*primitive.DateTime)(nil)},
						Received: wrong,
					},
				},
				{"DateTime/success", primitive.DateTime(1234567890), nil, nil, bsonrwtest.WriteDateTime, nil},
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
						Types:    []interface{}{primitive.Null{}, (*primitive.Null)(nil)},
						Received: wrong,
					},
				},
				{"Null/success", primitive.Null{}, nil, nil, bsonrwtest.WriteNull, nil},
				{"*Null/success", &primitive.Null{}, nil, nil, bsonrwtest.WriteNull, nil},
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
						Types:    []interface{}{primitive.Regex{}, (*primitive.Regex)(nil)},
						Received: wrong,
					},
				},
				{"Regex/success", primitive.Regex{Pattern: "foo", Options: "bar"}, nil, nil, bsonrwtest.WriteRegex, nil},
				{"*Regex/success", &primitive.Regex{Pattern: "foo", Options: "bar"}, nil, nil, bsonrwtest.WriteRegex, nil},
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
						Types:    []interface{}{primitive.DBPointer{}, (*primitive.DBPointer)(nil)},
						Received: wrong,
					},
				},
				{
					"DBPointer/success",
					primitive.DBPointer{
						DB:      "foobar",
						Pointer: objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					},
					nil, nil, bsonrwtest.WriteDBPointer, nil,
				},
				{
					"*DBPointer/success",
					&primitive.DBPointer{
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
						Types:    []interface{}{primitive.CodeWithScope{}, (*primitive.CodeWithScope)(nil)},
						Received: wrong,
					},
				},
				{
					"WriteCodeWithScope error",
					primitive.CodeWithScope{},
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("wcws error"), ErrAfter: bsonrwtest.WriteCodeWithScope},
					bsonrwtest.WriteCodeWithScope,
					errors.New("wcws error"),
				},
				{
					"CodeWithScope/success",
					primitive.CodeWithScope{
						Code:  "var hello = 'world';",
						Scope: NewDocumentv2(),
					},
					nil, nil, bsonrwtest.WriteDocumentEnd, nil,
				},
				{
					"*CodeWithScope/success",
					&primitive.CodeWithScope{
						Code:  "var hello = 'world';",
						Scope: NewDocumentv2(),
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
						Types:    []interface{}{primitive.Timestamp{}, (*primitive.Timestamp)(nil)},
						Received: wrong,
					},
				},
				{"Timestamp/success", primitive.Timestamp{T: 12345, I: 67890}, nil, nil, bsonrwtest.WriteTimestamp, nil},
				{"*Timestamp/success", &primitive.Timestamp{T: 12345, I: 67890}, nil, nil, bsonrwtest.WriteTimestamp, nil},
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
						Types:    []interface{}{primitive.MinKey{}, (*primitive.MinKey)(nil)},
						Received: wrong,
					},
				},
				{"MinKey/success", primitive.MinKey{}, nil, nil, bsonrwtest.WriteMinKey, nil},
				{"*MinKey/success", &primitive.MinKey{}, nil, nil, bsonrwtest.WriteMinKey, nil},
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
						Types:    []interface{}{primitive.MaxKey{}, (*primitive.MaxKey)(nil)},
						Received: wrong,
					},
				},
				{"MaxKey/success", primitive.MaxKey{}, nil, nil, bsonrwtest.WriteMaxKey, nil},
				{"*MaxKey/success", &primitive.MaxKey{}, nil, nil, bsonrwtest.WriteMaxKey, nil},
			},
		},
		{
			"RawValueEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.RawValueEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{
						Name:     "RawValueEncodeValue",
						Types:    []interface{}{RawValue{}, (*RawValue)(nil)},
						Received: wrong,
					},
				},
				{
					"RawValue/success",
					RawValue{Type: bsontype.Double, Value: bsoncore.AppendDouble(nil, 3.14159)},
					nil,
					nil,
					bsonrwtest.WriteDouble,
					nil,
				},
				{
					"*RawValue/success",
					&RawValue{Type: bsontype.Double, Value: bsoncore.AppendDouble(nil, 3.14159)},
					nil,
					nil,
					bsonrwtest.WriteDouble,
					nil,
				},
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
					bsoncodec.ValueEncoderError{Name: "ValueEncodeValue", Types: []interface{}{Valuev2{}}, Received: wrong},
				},
				{"empty value", Valuev2{}, nil, nil, bsonrwtest.WriteNull, nil},
				{
					"success",
					Null(),
					&bsoncodec.EncodeContext{Registry: DefaultRegistry},
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.WriteNull,
					nil,
				},
			},
		},
		{
			"RawEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.RawEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{Name: "RawEncodeValue", Types: []interface{}{Raw{}}, Received: wrong},
				},
				{
					"WriteDocument Error",
					Raw{},
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("wd error"), ErrAfter: bsonrwtest.WriteDocument},
					bsonrwtest.WriteDocument,
					errors.New("wd error"),
				},
				{
					"Raw.Elements Error",
					Raw{0xFF, 0x00, 0x00, 0x00, 0x00},
					nil,
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.WriteDocument,
					errors.New("length read exceeds number of bytes available. length=5 bytes=255"),
				},
				{
					"WriteDocumentElement Error",
					Raw(bytesFromDoc(NewDocumentv2(Elementv2{Key: "foo", Value: Null()}))),
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("wde error"), ErrAfter: bsonrwtest.WriteDocumentElement},
					bsonrwtest.WriteDocumentElement,
					errors.New("wde error"),
				},
				{
					"encodeValue error",
					Raw(bytesFromDoc(NewDocumentv2(Elementv2{Key: "foo", Value: Null()}))),
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("ev error"), ErrAfter: bsonrwtest.WriteNull},
					bsonrwtest.WriteNull,
					errors.New("ev error"),
				},
				{
					"iterator error",
					Raw{0x0C, 0x00, 0x00, 0x00, 0x01, 'f', 'o', 'o', 0x00, 0x01, 0x02, 0x03},
					nil,
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.WriteDocumentElement,
					errors.New("not enough bytes available to read type. bytes=3 type=double"),
				},
			},
		},
		{
			"DEncodeValue",
			bsoncodec.ValueEncoderFunc(pc.DEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueEncoderError{
						Name:     "DEncodeValue",
						Types:    []interface{}{D{}, (*D)(nil)},
						Received: wrong,
					},
				},
				{"D/success", D{{"hello", "world"}}, &bsoncodec.EncodeContext{Registry: DefaultRegistry}, nil, bsonrwtest.WriteDocumentEnd, nil},
				{"*D/success", &D{{"pi", 3.14159}}, &bsoncodec.EncodeContext{Registry: DefaultRegistry}, nil, bsonrwtest.WriteDocumentEnd, nil},
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
			want := bsoncodec.ValueEncoderError{Name: "DocumentEncodeValue", Types: []interface{}{(*Documentv2)(nil), (**Documentv2)(nil)}, Received: val}
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
			got := (PrimitiveCodecs{}).DocumentEncodeValue(bsoncodec.EncodeContext{}, llvrw, NewDocumentv2())
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("encodeDocument errors", func(t *testing.T) {
			ec := bsoncodec.EncodeContext{}
			err := errors.New("encodeDocument error")
			oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			testCases := []struct {
				name  string
				ec    bsoncodec.EncodeContext
				llvrw *bsonrwtest.ValueReaderWriter
				doc   *Documentv2
				err   error
			}{
				{
					"WriteDocumentElement",
					ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: errors.New("wde error"), ErrAfter: bsonrwtest.WriteDocumentElement},
					NewDocumentv2(EC("foo", Null())),
					errors.New("wde error"),
				},
				{
					"WriteDouble", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDouble},
					NewDocumentv2(Elementv2{Key: "foo", Value: Double(3.14159)}), err,
				},
				{
					"WriteString", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteString},
					NewDocumentv2(Elementv2{Key: "foo", Value: String("bar")}), err,
				},
				{
					"WriteDocument (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t},
					NewDocumentv2(Elementv2{Key: "foo", Value: EmbedElement("bar", Null())}),
					bsoncodec.ErrNoEncoder{Type: tDocument},
				},
				{
					"WriteArray (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t},
					NewDocumentv2(Elementv2{Key: "foo", Value: EmbedValues(Null())}),
					bsoncodec.ErrNoEncoder{Type: tArray},
				},
				{
					"WriteBinary", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteBinaryWithSubtype},
					NewDocumentv2(Elementv2{Key: "foo", Value: Binary(0xFF, []byte{0x01, 0x02, 0x03})}), err,
				},
				{
					"WriteUndefined", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteUndefined},
					NewDocumentv2(EC("foo", Undefined())), err,
				},
				{
					"WriteObjectID", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteObjectID},
					NewDocumentv2(Elementv2{Key: "foo", Value: ObjectID(oid)}), err,
				},
				{
					"WriteBoolean", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteBoolean},
					NewDocumentv2(Elementv2{Key: "foo", Value: Boolean(true)}), err,
				},
				{
					"WriteDateTime", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDateTime},
					NewDocumentv2(Elementv2{Key: "foo", Value: DateTime(1234567890)}), err,
				},
				{
					"WriteNull", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteNull},
					NewDocumentv2(EC("foo", Null())), err,
				},
				{
					"WriteRegex", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteRegex},
					NewDocumentv2(Elementv2{Key: "foo", Value: Regex("bar", "baz")}), err,
				},
				{
					"WriteDBPointer", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDBPointer},
					NewDocumentv2(Elementv2{Key: "foo", Value: DBPointer("bar", oid)}), err,
				},
				{
					"WriteJavascript", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteJavascript},
					NewDocumentv2(Elementv2{Key: "foo", Value: JavaScript("var hello = 'world';")}), err,
				},
				{
					"WriteSymbol", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteSymbol},
					NewDocumentv2(Elementv2{Key: "foo", Value: Symbol("symbolbaz")}), err,
				},
				{
					"WriteCodeWithScope (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteCodeWithScope},
					NewDocumentv2(Elementv2{Key: "foo", Value: CodeWithScope("var hello = 'world';", NewDocumentv2().Append("bar", Null()))}),
					err,
				},
				{
					"WriteInt32", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteInt32},
					NewDocumentv2(Elementv2{Key: "foo", Value: Int32(12345)}), err,
				},
				{
					"WriteInt64", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteInt64},
					NewDocumentv2(Elementv2{Key: "foo", Value: Int64(1234567890)}), err,
				},
				{
					"WriteTimestamp", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteTimestamp},
					NewDocumentv2(Elementv2{Key: "foo", Value: Timestamp(10, 20)}), err,
				},
				{
					"WriteDecimal128", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDecimal128},
					NewDocumentv2(Elementv2{Key: "foo", Value: Decimal128(decimal.NewDecimal128(10, 20))}), err,
				},
				{
					"WriteMinKey", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteMinKey},
					NewDocumentv2(EC("foo", MinKey())), err,
				},
				{
					"WriteMaxKey", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteMaxKey},
					NewDocumentv2(EC("foo", MaxKey())), err,
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
			want := NewDocumentv2(
				EC("a", Double(3.14159)), EC("b", String("foo")),
				EC("c", EmbedElement("aa", Null())), EC("d", EmbedValues(Null())),
				EC("e", Binary(0xFF, []byte{0x01, 0x02, 0x03})), EC("f", Undefined()),
				EC("g", ObjectID(oid)), EC("h", Boolean(true)),
				EC("i", DateTime(1234567890)), EC("j", Null()),
				EC("k", Regex("foo", "abr")),
				EC("l", DBPointer("foobar", oid)), EC("m", JavaScript("var hello = 'world';")),
				EC("n", Symbol("bazqux")),
				EC("o", CodeWithScope("var hello = 'world';", NewDocumentv2(EC("ab", Null())))),
				EC("p", Int32(12345)),
				EC("q", Timestamp(10, 20)), EC("r", Int64(1234567890)), EC("s", Decimal128(d128)), EC("t", MinKey()), EC("u", MaxKey()),
			)
			got := NewDocumentv2()
			slc := make(bsonrw.SliceWriter, 0, 128)
			vw, err := bsonrw.NewBSONValueWriter(&slc)
			noerr(t, err)

			ec := bsoncodec.EncodeContext{Registry: DefaultRegistry}
			err = (PrimitiveCodecs{}).DocumentEncodeValue(ec, vw, want)
			noerr(t, err)
			got, err = ReadDocumentv2(slc)
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
			want := bsoncodec.ValueEncoderError{Name: "ArrayEncodeValue", Types: []interface{}{(*Arrayv2)(nil)}, Received: val}
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
			got := (PrimitiveCodecs{}).ArrayEncodeValue(bsoncodec.EncodeContext{}, llvrw, NewArrayv2())
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("encode array errors", func(t *testing.T) {
			ec := bsoncodec.EncodeContext{}
			err := errors.New("encode array error")
			oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			testCases := []struct {
				name  string
				ec    bsoncodec.EncodeContext
				llvrw *bsonrwtest.ValueReaderWriter
				arr   *Arrayv2
				err   error
			}{
				{
					"WriteDocumentElement",
					ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: errors.New("wde error"), ErrAfter: bsonrwtest.WriteArrayElement},
					NewArrayv2(Null()),
					errors.New("wde error"),
				},
				{
					"WriteDouble", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDouble},
					NewArrayv2(Double(3.14159)), err,
				},
				{
					"WriteString", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteString},
					NewArrayv2(String("bar")), err,
				},
				{
					"WriteDocument (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t},
					NewArrayv2(Embed(NewDocumentv2(EC("bar", Null())))),
					bsoncodec.ErrNoEncoder{Type: tDocument},
				},
				{
					"WriteArray (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t},
					NewArrayv2(Embed(NewArrayv2(Null()))),
					bsoncodec.ErrNoEncoder{Type: tArray},
				},
				{
					"WriteBinary", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteBinaryWithSubtype},
					NewArrayv2(Binary(0xFF, []byte{0x01, 0x02, 0x03})), err,
				},
				{
					"WriteUndefined", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteUndefined},
					NewArrayv2(Undefined()), err,
				},
				{
					"WriteObjectID", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteObjectID},
					NewArrayv2(ObjectID(oid)), err,
				},
				{
					"WriteBoolean", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteBoolean},
					NewArrayv2(Boolean(true)), err,
				},
				{
					"WriteDateTime", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDateTime},
					NewArrayv2(DateTime(1234567890)), err,
				},
				{
					"WriteNull", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteNull},
					NewArrayv2(Null()), err,
				},
				{
					"WriteRegex", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteRegex},
					NewArrayv2(Regex("bar", "baz")), err,
				},
				{
					"WriteDBPointer", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDBPointer},
					NewArrayv2(DBPointer("bar", oid)), err,
				},
				{
					"WriteJavascript", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteJavascript},
					NewArrayv2(JavaScript("var hello = 'world';")), err,
				},
				{
					"WriteSymbol", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteSymbol},
					NewArrayv2(Symbol("symbolbaz")), err,
				},
				{
					"WriteCodeWithScope (Lookup)", bsoncodec.EncodeContext{Registry: bsoncodec.NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteCodeWithScope},
					NewArrayv2(CodeWithScope("var hello = 'world';", NewDocumentv2(EC("bar", Null())))),
					err,
				},
				{
					"WriteInt32", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteInt32},
					NewArrayv2(Int32(12345)), err,
				},
				{
					"WriteInt64", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteInt64},
					NewArrayv2(Int64(1234567890)), err,
				},
				{
					"WriteTimestamp", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteTimestamp},
					NewArrayv2(Timestamp(10, 20)), err,
				},
				{
					"WriteDecimal128", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteDecimal128},
					NewArrayv2(Decimal128(decimal.NewDecimal128(10, 20))), err,
				},
				{
					"WriteMinKey", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteMinKey},
					NewArrayv2(MinKey()), err,
				},
				{
					"WriteMaxKey", ec,
					&bsonrwtest.ValueReaderWriter{T: t, Err: err, ErrAfter: bsonrwtest.WriteMaxKey},
					NewArrayv2(MaxKey()), err,
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
			want := NewArrayv2(
				Double(3.14159), String("foo"), EmbedElement("aa", Null()),
				EmbedValues(Null()),
				Binary(0xFF, []byte{0x01, 0x02, 0x03}), Undefined(),
				ObjectID(oid), Boolean(true), DateTime(1234567890), Null(), Regex("foo", "abr"),
				DBPointer("foobar", oid), JavaScript("var hello = 'world';"), Symbol("bazqux"),
				CodeWithScope("var hello = 'world';", NewDocumentv2(EC("ab", Null()))), Int32(12345),
				Timestamp(10, 20), Int64(1234567890), Decimal128(d128), MinKey(), MaxKey(),
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

			val, err := bsoncore.Document(slc).LookupErr("foo")
			noerr(t, err)
			rgot := val.Array()
			doc, err := ReadDocumentv2(rgot)
			noerr(t, err)
			got := NewArrayv2()
			for _, elem := range doc.Elements() {
				got.Append(elem.Value)
			}
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
					O  *Documentv2
					P  Raw
					Q  objectid.ObjectID
					T  []struct{}
					Y  json.Number
					Z  time.Time
					AA json.Number
					AB *url.URL
					AC decimal.Decimal128
					AD *time.Time
					AE testValueMarshaler
					AF RawValue
					AG *RawValue
					AH D
					AI *D
					AJ *D
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
					O:  NewDocumentv2(Elementv2{Key: "countdown", Value: Int64(9876543210)}),
					P:  Raw{0x05, 0x00, 0x00, 0x00, 0x00},
					Q:  oid,
					T:  nil,
					Y:  json.Number("5"),
					Z:  now,
					AA: json.Number("10.1"),
					AB: murl,
					AC: decimal128,
					AD: &now,
					AE: testValueMarshaler{t: TypeString, buf: bsoncore.AppendString(nil, "hello, world")},
					AF: RawValue{Type: bsontype.String, Value: bsoncore.AppendString(nil, "hello, raw value")},
					AG: &RawValue{Type: bsontype.Double, Value: bsoncore.AppendDouble(nil, 3.14159)},
					AH: D{{"foo", "bar"}},
					AI: &D{{"pi", 3.14159}},
					AJ: nil,
				},
				docToBytes(NewDocumentv2(
					Elementv2{Key: "a", Value: Boolean(true)},
					Elementv2{Key: "b", Value: Int32(123)},
					Elementv2{Key: "c", Value: Int64(456)},
					Elementv2{Key: "d", Value: Int32(789)},
					Elementv2{Key: "e", Value: Int64(101112)},
					Elementv2{Key: "f", Value: Double(3.14159)},
					Elementv2{Key: "g", Value: String("Hello, world")},
					Elementv2{Key: "h", Value: EmbedElement("foo", String("bar"))},
					Elementv2{Key: "i", Value: Binary(0x00, []byte{0x01, 0x02, 0x03})},
					Elementv2{Key: "k", Value: EmbedValues(String("baz"), String("qux"))},
					Elementv2{Key: "l", Value: EmbedElement("m", String("foobar"))},
					Elementv2{Key: "o", Value: EmbedElement("countdown", Int64(9876543210))},
					EC("p", Embed(NewDocumentv2())),
					Elementv2{Key: "q", Value: ObjectID(oid)},
					EC("t", Null()),
					Elementv2{Key: "y", Value: Int64(5)},
					Elementv2{Key: "z", Value: DateTime(now.UnixNano() / int64(time.Millisecond))},
					Elementv2{Key: "aa", Value: Double(10.1)},
					Elementv2{Key: "ab", Value: String(murl.String())},
					Elementv2{Key: "ac", Value: Decimal128(decimal128)},
					Elementv2{Key: "ad", Value: DateTime(now.UnixNano() / int64(time.Millisecond))},
					Elementv2{Key: "ae", Value: String("hello, world")},
					Elementv2{Key: "af", Value: String("hello, raw value")},
					Elementv2{Key: "ag", Value: Double(3.14159)},
					Elementv2{Key: "ah", Value: EmbedElement("foo", String("bar"))},
					Elementv2{Key: "ai", Value: EmbedElement("pi", Double(3.14159))},
					EC("aj", Null()),
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
					O  []Elementv2
					P  []*Documentv2
					Q  []Raw
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
					AF []D
					AG []*D
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
					O:  []Elementv2{EC("N", Null())},
					P:  []*Documentv2{NewDocumentv2(Elementv2{Key: "countdown", Value: Int64(9876543210)})},
					Q:  []Raw{{0x05, 0x00, 0x00, 0x00, 0x00}},
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
					AF: []D{{{"foo", "bar"}}, {{"hello", "world"}, {"number", 12345}}},
					AG: []*D{{{"pi", 3.14159}}, nil},
				},
				docToBytes(NewDocumentv2(
					Elementv2{Key: "a", Value: EmbedValues(Boolean(true))},
					Elementv2{Key: "b", Value: EmbedValues(Int32(123))},
					Elementv2{Key: "c", Value: EmbedValues(Int64(456))},
					Elementv2{Key: "d", Value: EmbedValues(Int32(789))},
					Elementv2{Key: "e", Value: EmbedValues(Int64(101112))},
					Elementv2{Key: "f", Value: EmbedValues(Double(3.14159))},
					Elementv2{Key: "g", Value: EmbedValues(String("Hello, world"))},
					Elementv2{Key: "h", Value: EmbedValues(EmbedElement("foo", String("bar")))},
					Elementv2{Key: "i", Value: EmbedValues(Binary(0x00, []byte{0x01, 0x02, 0x03}))},
					Elementv2{Key: "k", Value: EmbedValues(EmbedValues(String("baz"), String("qux")))},
					Elementv2{Key: "l", Value: EmbedValues(EmbedElement("m", String("foobar")))},
					Elementv2{Key: "n", Value: EmbedValues(EmbedValues(String("foo"), String("bar")))},
					Elementv2{Key: "o", Value: EmbedElement("N", Null())},
					Elementv2{Key: "p", Value: EmbedValues(EmbedElement("countdown", Int64(9876543210)))},
					Elementv2{Key: "q", Value: EmbedValues(Embed(NewDocumentv2()))},
					Elementv2{Key: "r", Value: EmbedValues(ObjectID(oids[0]), ObjectID(oids[1]), ObjectID(oids[2]))},
					EC("t", Null()),
					EC("w", Null()),
					Elementv2{Key: "x", Value: Embed(NewArrayv2())},
					Elementv2{Key: "y", Value: EmbedValues(Embed(NewDocumentv2()))},
					Elementv2{Key: "z", Value: EmbedValues(DateTime(now.UnixNano()/int64(time.Millisecond)), DateTime(now.UnixNano()/int64(time.Millisecond)))},
					Elementv2{Key: "aa", Value: EmbedValues(Int64(5), Double(10.10))},
					Elementv2{Key: "ab", Value: EmbedValues(String(murl.String()))},
					Elementv2{Key: "ac", Value: EmbedValues(Decimal128(decimal128))},
					Elementv2{Key: "ad", Value: EmbedValues(DateTime(now.UnixNano()/int64(time.Millisecond)), DateTime(now.UnixNano()/int64(time.Millisecond)))},
					Elementv2{Key: "ae", Value: EmbedValues(String("hello"), String("world"))},
					EC("af", EmbedValues(
						EmbedElements([]Elementv2{{Key: "foo", Value: String("bar")}}),
						EmbedElements([]Elementv2{{Key: "hello", Value: String("world")}, {Key: "number", Value: Int64(12345)}}),
					)),
					Elementv2{Key: "ag", Value: EmbedValues(EmbedElement("pi", Double(3.14159)), Null())},
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
					t.Errorf("Readers\ngot: %v\nwant:%v\n", Raw(b), Raw(tc.b))
				}
			})
		}
	})
}

func TestDefaultValueDecoders(t *testing.T) {
	var pc PrimitiveCodecs

	var pjs = new(primitive.JavaScript)
	*pjs = primitive.JavaScript("var hello = 'world';")
	var psymbol = new(primitive.Symbol)
	*psymbol = primitive.Symbol("foobarbaz")

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
					bsoncodec.ValueDecoderError{Name: "JavaScriptDecodeValue", Types: []interface{}{(*primitive.JavaScript)(nil), (**primitive.JavaScript)(nil)}, Received: &wrong},
				},
				{
					"type not Javascript",
					primitive.JavaScript(""),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a primitive.JavaScript", bsontype.String),
				},
				{
					"ReadJavascript Error",
					primitive.JavaScript(""),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.JavaScript, Err: errors.New("rjs error"), ErrAfter: bsonrwtest.ReadJavascript},
					bsonrwtest.ReadJavascript,
					errors.New("rjs error"),
				},
				{
					"JavaScript/success",
					primitive.JavaScript("var hello = 'world';"),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.JavaScript, Return: "var hello = 'world';"},
					bsonrwtest.ReadJavascript,
					nil,
				},
				{
					"*JavaScript/success",
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
					bsoncodec.ValueDecoderError{Name: "SymbolDecodeValue", Types: []interface{}{(*primitive.Symbol)(nil), (**primitive.Symbol)(nil)}, Received: &wrong},
				},
				{
					"type not Symbol",
					primitive.Symbol(""),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a primitive.Symbol", bsontype.String),
				},
				{
					"ReadSymbol Error",
					primitive.Symbol(""),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Symbol, Err: errors.New("rjs error"), ErrAfter: bsonrwtest.ReadSymbol},
					bsonrwtest.ReadSymbol,
					errors.New("rjs error"),
				},
				{
					"Symbol/success",
					primitive.Symbol("var hello = 'world';"),
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
					bsoncodec.ValueDecoderError{Name: "BinaryDecodeValue", Types: []interface{}{(*primitive.Binary)(nil)}, Received: &wrong},
				},
				{
					"type not binary",
					primitive.Binary{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a Binary", bsontype.String),
				},
				{
					"ReadBinary Error",
					primitive.Binary{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Binary, Err: errors.New("rb error"), ErrAfter: bsonrwtest.ReadBinary},
					bsonrwtest.ReadBinary,
					errors.New("rb error"),
				},
				{
					"Binary/success",
					primitive.Binary{Data: []byte{0x01, 0x02, 0x03}, Subtype: 0xFF},
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
					&primitive.Binary{Data: []byte{0x01, 0x02, 0x03}, Subtype: 0xFF},
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
					bsoncodec.ValueDecoderError{Name: "UndefinedDecodeValue", Types: []interface{}{(*primitive.Undefined)(nil)}, Received: &wrong},
				},
				{
					"type not undefined",
					primitive.Undefined{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into an Undefined", bsontype.String),
				},
				{
					"ReadUndefined Error",
					primitive.Undefined{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Undefined, Err: errors.New("ru error"), ErrAfter: bsonrwtest.ReadUndefined},
					bsonrwtest.ReadUndefined,
					errors.New("ru error"),
				},
				{
					"ReadUndefined/success",
					primitive.Undefined{},
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
					bsoncodec.ValueDecoderError{Name: "DateTimeDecodeValue", Types: []interface{}{(*primitive.DateTime)(nil)}, Received: &wrong},
				},
				{
					"type not datetime",
					primitive.DateTime(0),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a DateTime", bsontype.String),
				},
				{
					"ReadDateTime Error",
					primitive.DateTime(0),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.DateTime, Err: errors.New("rdt error"), ErrAfter: bsonrwtest.ReadDateTime},
					bsonrwtest.ReadDateTime,
					errors.New("rdt error"),
				},
				{
					"success",
					primitive.DateTime(1234567890),
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
					bsoncodec.ValueDecoderError{Name: "NullDecodeValue", Types: []interface{}{(*primitive.Null)(nil)}, Received: &wrong},
				},
				{
					"type not null",
					primitive.Null{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a Null", bsontype.String),
				},
				{
					"ReadNull Error",
					primitive.Null{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Null, Err: errors.New("rn error"), ErrAfter: bsonrwtest.ReadNull},
					bsonrwtest.ReadNull,
					errors.New("rn error"),
				},
				{
					"success",
					primitive.Null{},
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
					bsoncodec.ValueDecoderError{Name: "RegexDecodeValue", Types: []interface{}{(*primitive.Regex)(nil)}, Received: &wrong},
				},
				{
					"type not regex",
					primitive.Regex{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a Regex", bsontype.String),
				},
				{
					"ReadRegex Error",
					primitive.Regex{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Regex, Err: errors.New("rr error"), ErrAfter: bsonrwtest.ReadRegex},
					bsonrwtest.ReadRegex,
					errors.New("rr error"),
				},
				{
					"success",
					primitive.Regex{Pattern: "foo", Options: "bar"},
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
					bsoncodec.ValueDecoderError{Name: "DBPointerDecodeValue", Types: []interface{}{(*primitive.DBPointer)(nil)}, Received: &wrong},
				},
				{
					"type not dbpointer",
					primitive.DBPointer{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a DBPointer", bsontype.String),
				},
				{
					"ReadDBPointer Error",
					primitive.DBPointer{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.DBPointer, Err: errors.New("rdbp error"), ErrAfter: bsonrwtest.ReadDBPointer},
					bsonrwtest.ReadDBPointer,
					errors.New("rdbp error"),
				},
				{
					"success",
					primitive.DBPointer{
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
						Types:    []interface{}{(*primitive.CodeWithScope)(nil)},
						Received: &wrong,
					},
				},
				{
					"type not codewithscope",
					primitive.CodeWithScope{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a CodeWithScope", bsontype.String),
				},
				{
					"ReadCodeWithScope Error",
					primitive.CodeWithScope{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.CodeWithScope, Err: errors.New("rcws error"), ErrAfter: bsonrwtest.ReadCodeWithScope},
					bsonrwtest.ReadCodeWithScope,
					errors.New("rcws error"),
				},
				{
					"decodeDocument Error",
					primitive.CodeWithScope{
						Code:  "var hello = 'world';",
						Scope: NewDocumentv2(EC("foo", Null())),
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
					bsoncodec.ValueDecoderError{Name: "TimestampDecodeValue", Types: []interface{}{(*primitive.Timestamp)(nil)}, Received: &wrong},
				},
				{
					"type not timestamp",
					primitive.Timestamp{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a Timestamp", bsontype.String),
				},
				{
					"ReadTimestamp Error",
					primitive.Timestamp{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Timestamp, Err: errors.New("rt error"), ErrAfter: bsonrwtest.ReadTimestamp},
					bsonrwtest.ReadTimestamp,
					errors.New("rt error"),
				},
				{
					"success",
					primitive.Timestamp{T: 12345, I: 67890},
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
					bsoncodec.ValueDecoderError{Name: "MinKeyDecodeValue", Types: []interface{}{(*primitive.MinKey)(nil)}, Received: &wrong},
				},
				{
					"type not null",
					primitive.MinKey{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a MinKey", bsontype.String),
				},
				{
					"ReadMinKey Error",
					primitive.MinKey{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.MinKey, Err: errors.New("rn error"), ErrAfter: bsonrwtest.ReadMinKey},
					bsonrwtest.ReadMinKey,
					errors.New("rn error"),
				},
				{
					"success",
					primitive.MinKey{},
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
					bsoncodec.ValueDecoderError{Name: "MaxKeyDecodeValue", Types: []interface{}{(*primitive.MaxKey)(nil)}, Received: &wrong},
				},
				{
					"type not null",
					primitive.MaxKey{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a MaxKey", bsontype.String),
				},
				{
					"ReadMaxKey Error",
					primitive.MaxKey{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.MaxKey, Err: errors.New("rn error"), ErrAfter: bsonrwtest.ReadMaxKey},
					bsonrwtest.ReadMaxKey,
					errors.New("rn error"),
				},
				{
					"success",
					primitive.MaxKey{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.MaxKey},
					bsonrwtest.ReadMaxKey,
					nil,
				},
			},
		},
		{
			"RawValueDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.RawValueDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{
						Name:     "RawValueDecodeValue",
						Types:    []interface{}{(*RawValue)(nil), (**RawValue)(nil)},
						Received: &wrong,
					},
				},
				{
					"ReadValue Error",
					RawValue{},
					nil,
					&bsonrwtest.ValueReaderWriter{
						BSONType: bsontype.Binary,
						Err:      errors.New("rb error"),
						ErrAfter: bsonrwtest.ReadBinary,
					},
					bsonrwtest.ReadBinary,
					errors.New("rb error"),
				},
				{
					"RawValue/success",
					RawValue{Type: bsontype.Binary, Value: bsoncore.AppendBinary(nil, 0xFF, []byte{0x01, 0x02, 0x03})},
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
					"*RawValue/success",
					&RawValue{Type: bsontype.Binary, Value: bsoncore.AppendBinary(nil, 0xFF, []byte{0x01, 0x02, 0x03})},
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
			"ValueDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.ValueDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{Name: "ValueDecodeValue", Types: []interface{}{(*Valuev2)(nil)}, Received: &wrong},
				},
				{"invalid value", (*Valuev2)(nil), nil, nil, bsonrwtest.Nothing, errors.New("ValueDecodeValue can only be used to decode non-nil *Value")},
				{
					"success",
					Double(3.14159),
					&bsoncodec.DecodeContext{Registry: NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.14159)},
					bsonrwtest.ReadDouble,
					nil,
				},
			},
		},
		{
			"RawDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.RawDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{Name: "RawDecodeValue", Types: []interface{}{(*Raw)(nil)}, Received: &wrong},
				},
				{
					"*Raw is nil",
					(*Raw)(nil),
					nil,
					nil,
					bsonrwtest.Nothing,
					errors.New("RawDecodeValue can only be used to decode non-nil *Reader"),
				},
				{
					"Copy error",
					Raw{},
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("copy error"), ErrAfter: bsonrwtest.ReadDocument},
					bsonrwtest.ReadDocument,
					errors.New("copy error"),
				},
			},
		},
		{
			"DDecodeValue",
			bsoncodec.ValueDecoderFunc(pc.DDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					bsoncodec.ValueDecoderError{Name: "DDecodeValue", Types: []interface{}{(*D)(nil), (**D)(nil)}, Received: &wrong},
				},
				{
					"type not valid",
					D{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a D", bsontype.String),
				},
				{
					"ReadDocument Error",
					D{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.EmbeddedDocument, Err: errors.New("rd error"), ErrAfter: bsonrwtest.ReadDocument},
					bsonrwtest.ReadDocument,
					errors.New("rd error"),
				},
				{
					"ReadElement Error",
					D{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.EmbeddedDocument, Err: errors.New("re error"), ErrAfter: bsonrwtest.ReadElement},
					bsonrwtest.ReadElement,
					errors.New("re error"),
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
		b, err := NewDocumentv2(Elementv2{Key: "foo", Value: CodeWithScope("var hello = 'world';", NewDocumentv2(EC("bar", Null())))}).MarshalBSON()
		noerr(t, err)
		dvr := bsonrw.NewBSONDocumentReader(b)
		dr, err := dvr.ReadDocument()
		noerr(t, err)
		_, vr, err := dr.ReadElement()
		noerr(t, err)

		want := primitive.CodeWithScope{
			Code:  "var hello = 'world';",
			Scope: NewDocumentv2(EC("bar", Null())),
		}
		var got primitive.CodeWithScope
		err = pc.CodeWithScopeDecodeValue(dc, vr, &got)
		noerr(t, err)

		if got.Code != want.Code && !cmp.Equal(got.Scope, want.Scope) {
			t.Errorf("CodeWithScopes do not match. got %v; want %v", got, want)
		}
	})
	t.Run("DocumentDecodeValue", func(t *testing.T) {
		t.Run("CodecDecodeError", func(t *testing.T) {
			val := bool(true)
			want := bsoncodec.ValueDecoderError{Name: "DocumentDecodeValue", Types: []interface{}{(**Documentv2)(nil)}, Received: val}
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
			got := pc.DocumentDecodeValue(bsoncodec.DecodeContext{}, llvrw, new(*Documentv2))
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
					err := pc.decodeDocument(tc.dc, tc.llvrw, new(*Documentv2))
					if !compareErrors(err, tc.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, tc.err)
					}
				})
			}
		})

		t.Run("success", func(t *testing.T) {
			oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			d128 := decimal.NewDecimal128(10, 20)
			want := NewDocumentv2(
				Elementv2{Key: "a", Value: Double(3.14159)}, Elementv2{Key: "b", Value: String("foo")},
				Elementv2{Key: "c", Value: EmbedElement("aa", Null())},
				Elementv2{Key: "d", Value: EmbedValues(Null())},
				Elementv2{Key: "e", Value: Binary(0xFF, []byte{0x01, 0x02, 0x03})}, EC("f", Undefined()),
				Elementv2{Key: "g", Value: ObjectID(oid)}, Elementv2{Key: "h", Value: Boolean(true)},
				Elementv2{Key: "i", Value: DateTime(1234567890)}, EC("j", Null()), Elementv2{Key: "k", Value: Regex("foo", "bar")},
				Elementv2{Key: "l", Value: DBPointer("foobar", oid)}, Elementv2{Key: "m", Value: JavaScript("var hello = 'world';")},
				Elementv2{Key: "n", Value: Symbol("bazqux")},
				Elementv2{Key: "o", Value: CodeWithScope("var hello = 'world';", NewDocumentv2(EC("ab", Null())))},
				Elementv2{Key: "p", Value: Int32(12345)},
				Elementv2{Key: "q", Value: Timestamp(10, 20)}, Elementv2{Key: "r", Value: Int64(1234567890)},
				Elementv2{Key: "s", Value: Decimal128(d128)}, EC("t", MinKey()), EC("u", MaxKey()),
			)
			var got *Documentv2
			dc := bsoncodec.DecodeContext{Registry: NewRegistryBuilder().Build()}
			b, err := want.MarshalBSON()
			noerr(t, err)
			err = pc.DocumentDecodeValue(dc, bsonrw.NewBSONDocumentReader(b), &got)
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
			want := bsoncodec.ValueDecoderError{Name: "ArrayDecodeValue", Types: []interface{}{(**Arrayv2)(nil)}, Received: val}
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
			got := pc.ArrayDecodeValue(bsoncodec.DecodeContext{}, llvrw, new(*Arrayv2))
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
					"ReadValue",
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
					err := pc.ArrayDecodeValue(tc.dc, tc.llvrw, new(*Arrayv2))
					if !compareErrors(err, tc.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, tc.err)
					}
				})
			}
		})

		t.Run("success", func(t *testing.T) {
			oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			d128 := decimal.NewDecimal128(10, 20)
			want := NewArrayv2(
				Double(3.14159), String("foo"), EmbedElement("aa", Null()),
				EmbedValues(Null()),
				Binary(0xFF, []byte{0x01, 0x02, 0x03}), Undefined(),
				ObjectID(oid), Boolean(true), DateTime(1234567890), Null(), Regex("foo", "bar"),
				DBPointer("foobar", oid), JavaScript("var hello = 'world';"), Symbol("bazqux"),
				CodeWithScope("var hello = 'world';", NewDocumentv2(EC("ab", Null()))), Int32(12345),
				Timestamp(10, 20), Int64(1234567890), Decimal128(d128), MinKey(), MaxKey(),
			)
			dc := bsoncodec.DecodeContext{Registry: NewRegistryBuilder().Build()}

			b, err := NewDocumentv2(EC("", Embed(want))).MarshalBSON()
			noerr(t, err)
			dvr := bsonrw.NewBSONDocumentReader(b)
			dr, err := dvr.ReadDocument()
			noerr(t, err)
			_, vr, err := dr.ReadElement()
			noerr(t, err)

			var got *Arrayv2
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
				docToBytes(NewDocumentv2(Elementv2{Key: "foo", Value: ObjectID(oid)})),
				nil,
			},
			{
				"map[string][]Element",
				map[string][]Elementv2{"Z": {Elementv2{Key: "A", Value: Int32(1)}, Elementv2{Key: "B", Value: Int32(2)}, Elementv2{Key: "EC", Value: Int32(3)}}},
				docToBytes(NewDocumentv2(
					Elementv2{Key: "Z", Value: EmbedElements([]Elementv2{EC("A", Int32(1)), {Key: "B", Value: Int32(2)}, {Key: "EC", Value: Int32(3)}})},
				)),
				nil,
			},
			{
				"map[string][]Value",
				map[string][]Valuev2{"Z": {Int32(1), Int32(2), Int32(3)}},
				docToBytes(NewDocumentv2(
					Elementv2{Key: "Z", Value: EmbedValues(Int32(1), Int32(2), Int32(3))},
				)),
				nil,
			},
			{
				"map[string]*Document",
				map[string]*Documentv2{"Z": NewDocumentv2(EC("foo", Null()))},
				docToBytes(NewDocumentv2(
					Elementv2{Key: "Z", Value: EmbedElement("foo", Null())},
				)),
				nil,
			},
			{
				"map[string]Reader",
				map[string]Raw{"Z": {0x05, 0x00, 0x00, 0x00, 0x00}},
				docToBytes(NewDocumentv2(
					Elementv2{Key: "Z", Value: Embed(rawToDoc(Raw{0x05, 0x00, 0x00, 0x00, 0x00}))},
				)),
				nil,
			},
			{
				"map[string][]int32",
				map[string][]int32{"Z": {1, 2, 3}},
				docToBytes(NewDocumentv2(
					Elementv2{Key: "Z", Value: EmbedValues(Int32(1), Int32(2), Int32(3))},
				)),
				nil,
			},
			{
				"map[string][]objectid.ObjectID",
				map[string][]objectid.ObjectID{"Z": oids},
				docToBytes(NewDocumentv2(
					Elementv2{Key: "Z", Value: EmbedValues(ObjectID(oids[0]), ObjectID(oids[1]), ObjectID(oids[2]))},
				)),
				nil,
			},
			{
				"map[string][]json.Number(int64)",
				map[string][]json.Number{"Z": {json.Number("5"), json.Number("10")}},
				docToBytes(NewDocumentv2(
					Elementv2{Key: "Z", Value: EmbedValues(Int64(5), Int64(10))},
				)),
				nil,
			},
			{
				"map[string][]json.Number(float64)",
				map[string][]json.Number{"Z": {json.Number("5"), json.Number("10.1")}},
				docToBytes(NewDocumentv2(
					Elementv2{Key: "Z", Value: EmbedValues(Int64(5), Double(10.1))},
				)),
				nil,
			},
			{
				"map[string][]*url.URL",
				map[string][]*url.URL{"Z": {murl}},
				docToBytes(NewDocumentv2(
					Elementv2{Key: "Z", Value: EmbedValues(String(murl.String()))},
				)),
				nil,
			},
			{
				"map[string][]decimal.Decimal128",
				map[string][]decimal.Decimal128{"Z": {decimal128}},
				docToBytes(NewDocumentv2(
					Elementv2{Key: "Z", Value: EmbedValues(Decimal128(decimal128))},
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
				docToBytes(NewDocumentv2()),
				nil,
			},
			{
				"omitempty",
				struct {
					A string `bson:",omitempty"`
				}{
					A: "",
				},
				docToBytes(NewDocumentv2()),
				nil,
			},
			{
				"omitempty, empty time",
				struct {
					A time.Time `bson:",omitempty"`
				}{
					A: time.Time{},
				},
				docToBytes(NewDocumentv2()),
				nil,
			},
			{
				"no private fields",
				noPrivateFields{a: "should be empty"},
				docToBytes(NewDocumentv2()),
				nil,
			},
			{
				"minsize",
				struct {
					A int64 `bson:",minsize"`
				}{
					A: 12345,
				},
				docToBytes(NewDocumentv2(Elementv2{Key: "a", Value: Int32(12345)})),
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
				docToBytes(NewDocumentv2(Elementv2{Key: "a", Value: Int32(12345)})),
				nil,
			},
			{
				"inline map",
				struct {
					Foo map[string]string `bson:",inline"`
				}{
					Foo: map[string]string{"foo": "bar"},
				},
				docToBytes(NewDocumentv2(Elementv2{Key: "foo", Value: String("bar")})),
				nil,
			},
			{
				"alternate name bson:name",
				struct {
					A string `bson:"foo"`
				}{
					A: "bar",
				},
				docToBytes(NewDocumentv2(Elementv2{Key: "foo", Value: String("bar")})),
				nil,
			},
			{
				"alternate name",
				struct {
					A string `bson:"foo"`
				}{
					A: "bar",
				},
				docToBytes(NewDocumentv2(Elementv2{Key: "foo", Value: String("bar")})),
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
				docToBytes(NewDocumentv2(Elementv2{Key: "a", Value: String("bar")})),
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
					O  *Documentv2
					P  Raw
					Q  objectid.ObjectID
					T  []struct{}
					Y  json.Number
					Z  time.Time
					AA json.Number
					AB *url.URL
					AC decimal.Decimal128
					AD *time.Time
					AE *testValueUnmarshaler
					AF RawValue
					AG *RawValue
					AH D
					AI *D
					AJ *D
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
					O:  NewDocumentv2(Elementv2{Key: "countdown", Value: Int64(9876543210)}),
					P:  Raw{0x05, 0x00, 0x00, 0x00, 0x00},
					Q:  oid,
					T:  nil,
					Y:  json.Number("5"),
					Z:  now,
					AA: json.Number("10.1"),
					AB: murl,
					AC: decimal128,
					AD: &now,
					AE: &testValueUnmarshaler{t: bsontype.String, val: bsoncore.AppendString(nil, "hello, world!")},
					AF: RawValue{Type: bsontype.Double, Value: bsoncore.AppendDouble(nil, 3.14159)},
					AG: &RawValue{Type: bsontype.Binary, Value: bsoncore.AppendBinary(nil, 0xFF, []byte{0x01, 0x02, 0x03})},
					AH: D{{"foo", "bar"}},
					AI: &D{{"pi", 3.14159}},
					AJ: nil,
				},
				docToBytes(NewDocumentv2(
					Elementv2{Key: "a", Value: Boolean(true)},
					Elementv2{Key: "b", Value: Int32(123)},
					Elementv2{Key: "c", Value: Int64(456)},
					Elementv2{Key: "d", Value: Int32(789)},
					Elementv2{Key: "e", Value: Int64(101112)},
					Elementv2{Key: "f", Value: Double(3.14159)},
					Elementv2{Key: "g", Value: String("Hello, world")},
					Elementv2{Key: "h", Value: EmbedElement("foo", String("bar"))},
					Elementv2{Key: "i", Value: Binary(0x00, []byte{0x01, 0x02, 0x03})},
					Elementv2{Key: "k", Value: EmbedValues(String("baz"), String("qux"))},
					Elementv2{Key: "l", Value: EmbedElement("m", String("foobar"))},
					Elementv2{Key: "o", Value: EmbedElement("countdown", Int64(9876543210))},
					EC("p", Embed(NewDocumentv2())),
					Elementv2{Key: "q", Value: ObjectID(oid)},
					EC("t", Null()),
					Elementv2{Key: "y", Value: Int64(5)},
					Elementv2{Key: "z", Value: DateTime(now.UnixNano() / int64(time.Millisecond))},
					Elementv2{Key: "aa", Value: Double(10.1)},
					Elementv2{Key: "ab", Value: String(murl.String())},
					Elementv2{Key: "ac", Value: Decimal128(decimal128)},
					Elementv2{Key: "ad", Value: DateTime(now.UnixNano() / int64(time.Millisecond))},
					Elementv2{Key: "ae", Value: String("hello, world!")},
					Elementv2{Key: "af", Value: Double(3.14159)},
					Elementv2{Key: "ag", Value: Binary(0xFF, []byte{0x01, 0x02, 0x03})},
					Elementv2{Key: "ah", Value: EmbedElement("foo", String("bar"))},
					Elementv2{Key: "ai", Value: EmbedElement("pi", Double(3.14159))},
					EC("aj", Null()),
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
					O  []Elementv2
					P  []*Documentv2
					Q  []Raw
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
					AF []D
					AG []*D
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
					O:  []Elementv2{EC("N", Null())},
					P:  []*Documentv2{NewDocumentv2(Elementv2{Key: "countdown", Value: Int64(9876543210)})},
					Q:  []Raw{{0x05, 0x00, 0x00, 0x00, 0x00}},
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
					AF: []D{{{"foo", "bar"}}, {{"hello", "world"}, {"number", int64(12345)}}},
					AG: []*D{{{"pi", 3.14159}}, nil},
				},
				docToBytes(NewDocumentv2(
					Elementv2{Key: "a", Value: EmbedValues(Boolean(true))},
					Elementv2{Key: "b", Value: EmbedValues(Int32(123))},
					Elementv2{Key: "c", Value: EmbedValues(Int64(456))},
					Elementv2{Key: "d", Value: EmbedValues(Int32(789))},
					Elementv2{Key: "e", Value: EmbedValues(Int64(101112))},
					Elementv2{Key: "f", Value: EmbedValues(Double(3.14159))},
					Elementv2{Key: "g", Value: EmbedValues(String("Hello, world"))},
					Elementv2{Key: "h", Value: EmbedValues(EmbedElement("foo", String("bar")))},
					Elementv2{Key: "i", Value: EmbedValues(Binary(0x00, []byte{0x01, 0x02, 0x03}))},
					Elementv2{Key: "k", Value: EmbedValues(EmbedValues(String("baz"), String("qux")))},
					Elementv2{Key: "l", Value: EmbedValues(EmbedElement("m", String("foobar")))},
					Elementv2{Key: "n", Value: EmbedValues(EmbedValues(String("foo"), String("bar")))},
					Elementv2{Key: "o", Value: EmbedElement("N", Null())},
					Elementv2{Key: "p", Value: EmbedValues(EmbedElement("countdown", Int64(9876543210)))},
					Elementv2{Key: "q", Value: EmbedValues(Embed(NewDocumentv2()))},
					Elementv2{Key: "r", Value: EmbedValues(ObjectID(oids[0]), ObjectID(oids[1]), ObjectID(oids[2]))},
					EC("t", Null()),
					EC("w", Null()),
					Elementv2{Key: "x", Value: Embed(NewArrayv2())},
					Elementv2{Key: "y", Value: EmbedValues(Embed(NewDocumentv2()))},
					Elementv2{Key: "z", Value: EmbedValues(DateTime(now.UnixNano()/int64(time.Millisecond)), DateTime(now.UnixNano()/int64(time.Millisecond)))},
					Elementv2{Key: "aa", Value: EmbedValues(Int64(5), Double(10.10))},
					Elementv2{Key: "ab", Value: EmbedValues(String(murl.String()))},
					Elementv2{Key: "ac", Value: EmbedValues(Decimal128(decimal128))},
					Elementv2{Key: "ad", Value: EmbedValues(DateTime(now.UnixNano()/int64(time.Millisecond)), DateTime(now.UnixNano()/int64(time.Millisecond)))},
					Elementv2{Key: "ae", Value: EmbedValues(String("hello"), String("world"))},
					EC("af", EmbedValues(
						EmbedElement("foo", String("bar")),
						EmbedElements([]Elementv2{{Key: "hello", Value: String("world")}, {Key: "number", Value: Int64(12345)}}),
					)),
					Elementv2{Key: "ag", Value: EmbedValues(EmbedElement("pi", Double(3.14159)), Null())},
				)),
				nil,
			},
		}

		t.Run("Decode", func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					vr := bsonrw.NewBSONDocumentReader(tc.b)
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
					NewDocumentv2(EC("foo", Null())),
					bsontype.EmbeddedDocument,
				},
				{
					"Array - *Arrayv2",
					NewArrayv2(Double(3.14159)),
					bsontype.Array,
				},
				{
					"Binary - Binary",
					primitive.Binary{Subtype: 0xFF, Data: []byte{0x01, 0x02, 0x03}},
					bsontype.Binary,
				},
				{
					"Undefined - Undefined",
					primitive.Undefined{},
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
					primitive.DateTime(1234567890),
					bsontype.DateTime,
				},
				{
					"Null - Null",
					primitive.Null{},
					bsontype.Null,
				},
				{
					"Regex - Regex",
					primitive.Regex{Pattern: "foo", Options: "bar"},
					bsontype.Regex,
				},
				{
					"DBPointer - DBPointer",
					primitive.DBPointer{
						DB:      "foobar",
						Pointer: objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					},
					bsontype.DBPointer,
				},
				{
					"JavaScript - JavaScript",
					primitive.JavaScript("var foo = 'bar';"),
					bsontype.JavaScript,
				},
				{
					"Symbol - Symbol",
					primitive.Symbol("foobarbazlolz"),
					bsontype.Symbol,
				},
				{
					"CodeWithScope - CodeWithScope",
					primitive.CodeWithScope{
						Code:  "var foo = 'bar';",
						Scope: NewDocumentv2(Elementv2{Key: "foo", Value: Double(3.14159)}),
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
					primitive.Timestamp{T: 12345, I: 67890},
					bsontype.Timestamp,
				},
				{
					"Decimal128 - decimal.Decimal128",
					decimal.NewDecimal128(12345, 67890),
					bsontype.Decimal128,
				},
				{
					"MinKey - MinKey",
					primitive.MinKey{},
					bsontype.MinKey,
				},
				{
					"MaxKey - MaxKey",
					primitive.MaxKey{},
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
		decodeval, ok := llc.decodeval.(*Documentv2)
		if !ok {
			llc.t.Errorf("decodeval must be a *Document if the i is a *Document. decodeval %T", llc.decodeval)
			return nil
		}

		doc := i.(*Documentv2)
		doc.Reset()
		doc = doc.AppendElements(decodeval.Elements()...)
		return nil
	case tArray:
		decodeval, ok := llc.decodeval.(*Arrayv2)
		if !ok {
			llc.t.Errorf("decodeval must be a *Array if the i is a *Array. decodeval %T", llc.decodeval)
			return nil
		}

		arr := i.(*Arrayv2)
		arr.Reset()
		arr = arr.Append(decodeval.Values()...)
		return nil
	}

	if !reflect.TypeOf(llc.decodeval).AssignableTo(val.Type().Elem()) {
		llc.t.Errorf("decodeval must be assignable to i provided to DecodeValue, but is not. decodeval %T; i %T", llc.decodeval, i)
		return nil
	}

	val.Elem().Set(reflect.ValueOf(llc.decodeval))
	return nil
}

func rawToDoc(raw Raw) *Documentv2 {
	doc, err := ReadDocumentv2(raw)
	if err != nil {
		panic(err)
	}
	return doc
}
