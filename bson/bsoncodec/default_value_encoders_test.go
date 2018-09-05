package bsoncodec

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/internal/llbson"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

func TestDefaultValueEncoders(t *testing.T) {
	var dve DefaultValueEncoders
	var wrong = func(string, string) string { return "wrong" }

	type mybool bool
	type myint8 int8
	type myint16 int16
	type myint32 int32
	type myint64 int64
	type myint int
	type myuint8 uint8
	type myuint16 uint16
	type myuint32 uint32
	type myuint64 uint64
	type myuint uint
	type myfloat32 float32
	type myfloat64 float64
	type mystring string

	intAllowedTypes := []interface{}{int8(0), int16(0), int32(0), int64(0), int(0)}
	uintAllowedEncodeTypes := []interface{}{uint8(0), uint16(0), uint32(0), uint64(0), uint(0)}

	now := time.Now().Truncate(time.Millisecond)
	pdatetime := new(bson.DateTime)
	*pdatetime = bson.DateTime(1234567890)
	pjsnum := new(json.Number)
	*pjsnum = json.Number("3.14159")
	d128 := decimal.NewDecimal128(12345, 67890)

	type subtest struct {
		name   string
		val    interface{}
		ectx   *EncodeContext
		llvrw  *llValueReaderWriter
		invoke llvrwInvoked
		err    error
	}

	testCases := []struct {
		name     string
		ve       ValueEncoder
		subtests []subtest
	}{
		{
			"BooleanEncodeValue",
			ValueEncoderFunc(dve.BooleanEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{Name: "BooleanEncodeValue", Types: []interface{}{bool(true)}, Received: wrong},
				},
				{"fast path", bool(true), nil, nil, llvrwWriteBoolean, nil},
				{"reflection path", mybool(true), nil, nil, llvrwWriteBoolean, nil},
			},
		},
		{
			"IntEncodeValue",
			ValueEncoderFunc(dve.IntEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{Name: "IntEncodeValue", Types: intAllowedTypes, Received: wrong},
				},
				{"int8/fast path", int8(127), nil, nil, llvrwWriteInt32, nil},
				{"int16/fast path", int16(32767), nil, nil, llvrwWriteInt32, nil},
				{"int32/fast path", int32(2147483647), nil, nil, llvrwWriteInt32, nil},
				{"int64/fast path", int64(1234567890987), nil, nil, llvrwWriteInt64, nil},
				{"int/fast path", int(1234567), nil, nil, llvrwWriteInt64, nil},
				{"int64/fast path - minsize", int64(2147483647), &EncodeContext{MinSize: true}, nil, llvrwWriteInt32, nil},
				{"int/fast path - minsize", int(2147483647), &EncodeContext{MinSize: true}, nil, llvrwWriteInt32, nil},
				{"int64/fast path - minsize too large", int64(2147483648), &EncodeContext{MinSize: true}, nil, llvrwWriteInt64, nil},
				{"int/fast path - minsize too large", int(2147483648), &EncodeContext{MinSize: true}, nil, llvrwWriteInt64, nil},
				{"int8/reflection path", myint8(127), nil, nil, llvrwWriteInt32, nil},
				{"int16/reflection path", myint16(32767), nil, nil, llvrwWriteInt32, nil},
				{"int32/reflection path", myint32(2147483647), nil, nil, llvrwWriteInt32, nil},
				{"int64/reflection path", myint64(1234567890987), nil, nil, llvrwWriteInt64, nil},
				{"int/reflection path", myint(1234567890987), nil, nil, llvrwWriteInt64, nil},
				{"int64/reflection path - minsize", myint64(2147483647), &EncodeContext{MinSize: true}, nil, llvrwWriteInt32, nil},
				{"int/reflection path - minsize", myint(2147483647), &EncodeContext{MinSize: true}, nil, llvrwWriteInt32, nil},
				{"int64/reflection path - minsize too large", myint64(2147483648), &EncodeContext{MinSize: true}, nil, llvrwWriteInt64, nil},
				{"int/reflection path - minsize too large", myint(2147483648), &EncodeContext{MinSize: true}, nil, llvrwWriteInt64, nil},
			},
		},
		{
			"UintEncodeValue",
			ValueEncoderFunc(dve.UintEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{Name: "UintEncodeValue", Types: uintAllowedEncodeTypes, Received: wrong},
				},
				{"uint8/fast path", uint8(127), nil, nil, llvrwWriteInt32, nil},
				{"uint16/fast path", uint16(32767), nil, nil, llvrwWriteInt32, nil},
				{"uint32/fast path", uint32(2147483647), nil, nil, llvrwWriteInt64, nil},
				{"uint64/fast path", uint64(1234567890987), nil, nil, llvrwWriteInt64, nil},
				{"uint/fast path", uint(1234567), nil, nil, llvrwWriteInt64, nil},
				{"uint32/fast path - minsize", uint32(2147483647), &EncodeContext{MinSize: true}, nil, llvrwWriteInt32, nil},
				{"uint64/fast path - minsize", uint64(2147483647), &EncodeContext{MinSize: true}, nil, llvrwWriteInt32, nil},
				{"uint/fast path - minsize", uint(2147483647), &EncodeContext{MinSize: true}, nil, llvrwWriteInt32, nil},
				{"uint32/fast path - minsize too large", uint32(2147483648), &EncodeContext{MinSize: true}, nil, llvrwWriteInt64, nil},
				{"uint64/fast path - minsize too large", uint64(2147483648), &EncodeContext{MinSize: true}, nil, llvrwWriteInt64, nil},
				{"uint/fast path - minsize too large", uint(2147483648), &EncodeContext{MinSize: true}, nil, llvrwWriteInt64, nil},
				{"uint64/fast path - overflow", uint64(1 << 63), nil, nil, llvrwNothing, fmt.Errorf("%d overflows int64", uint(1<<63))},
				{"uint/fast path - overflow", uint(1 << 63), nil, nil, llvrwNothing, fmt.Errorf("%d overflows int64", uint(1<<63))},
				{"uint8/reflection path", myuint8(127), nil, nil, llvrwWriteInt32, nil},
				{"uint16/reflection path", myuint16(32767), nil, nil, llvrwWriteInt32, nil},
				{"uint32/reflection path", myuint32(2147483647), nil, nil, llvrwWriteInt64, nil},
				{"uint64/reflection path", myuint64(1234567890987), nil, nil, llvrwWriteInt64, nil},
				{"uint/reflection path", myuint(1234567890987), nil, nil, llvrwWriteInt64, nil},
				{"uint32/reflection path - minsize", myuint32(2147483647), &EncodeContext{MinSize: true}, nil, llvrwWriteInt32, nil},
				{"uint64/reflection path - minsize", myuint64(2147483647), &EncodeContext{MinSize: true}, nil, llvrwWriteInt32, nil},
				{"uint/reflection path - minsize", myuint(2147483647), &EncodeContext{MinSize: true}, nil, llvrwWriteInt32, nil},
				{"uint32/reflection path - minsize too large", myuint(1 << 31), &EncodeContext{MinSize: true}, nil, llvrwWriteInt64, nil},
				{"uint64/reflection path - minsize too large", myuint64(1 << 31), &EncodeContext{MinSize: true}, nil, llvrwWriteInt64, nil},
				{"uint/reflection path - minsize too large", myuint(2147483648), &EncodeContext{MinSize: true}, nil, llvrwWriteInt64, nil},
				{"uint64/reflection path - overflow", myuint64(1 << 63), nil, nil, llvrwNothing, fmt.Errorf("%d overflows int64", uint(1<<63))},
				{"uint/reflection path - overflow", myuint(1 << 63), nil, nil, llvrwNothing, fmt.Errorf("%d overflows int64", uint(1<<63))},
			},
		},
		{
			"FloatEncodeValue",
			ValueEncoderFunc(dve.FloatEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{Name: "FloatEncodeValue", Types: []interface{}{float32(0), float64(0)}, Received: wrong},
				},
				{"float32/fast path", float32(3.14159), nil, nil, llvrwWriteDouble, nil},
				{"float64/fast path", float64(3.14159), nil, nil, llvrwWriteDouble, nil},
				{"float32/reflection path", myfloat32(3.14159), nil, nil, llvrwWriteDouble, nil},
				{"float64/reflection path", myfloat64(3.14159), nil, nil, llvrwWriteDouble, nil},
			},
		},
		{
			"StringEncodeValue",
			ValueEncoderFunc(dve.StringEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "StringEncodeValue",
						Types:    []interface{}{string(""), bson.JavaScriptCode(""), bson.Symbol("")},
						Received: wrong,
					},
				},
				{"string/fast path", string("foobar"), nil, nil, llvrwWriteString, nil},
				{"JavaScript/fast path", bson.JavaScriptCode("foobar"), nil, nil, llvrwWriteJavascript, nil},
				{"Symbol/fast path", bson.Symbol("foobar"), nil, nil, llvrwWriteSymbol, nil},
				{"reflection path", mystring("foobarbaz"), nil, nil, llvrwWriteString, nil},
			},
		},
		{
			"TimeEncodeValue",
			ValueEncoderFunc(dve.TimeEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "TimeEncodeValue",
						Types:    []interface{}{time.Time{}, (*time.Time)(nil)},
						Received: wrong,
					},
				},
				{"time.Time", now, nil, nil, llvrwWriteDateTime, nil},
				{"*time.Time", &now, nil, nil, llvrwWriteDateTime, nil},
			},
		},
		{
			"MapEncodeValue",
			ValueEncoderFunc(dve.MapEncodeValue),
			[]subtest{
				{
					"wrong kind",
					wrong,
					nil,
					nil,
					llvrwNothing,
					errors.New("MapEncodeValue can only encode maps with string keys"),
				},
				{
					"wrong kind (non-string key)",
					map[int]interface{}{},
					nil,
					nil,
					llvrwNothing,
					errors.New("MapEncodeValue can only encode maps with string keys"),
				},
				{
					"WriteDocument Error",
					map[string]interface{}{},
					nil,
					&llValueReaderWriter{err: errors.New("wd error"), errAfter: llvrwWriteDocument},
					llvrwWriteDocument,
					errors.New("wd error"),
				},
				{
					"Lookup Error",
					map[string]interface{}{},
					&EncodeContext{Registry: NewEmptyRegistryBuilder().Build()},
					&llValueReaderWriter{},
					llvrwWriteDocument,
					ErrNoEncoder{Type: reflect.TypeOf((*interface{})(nil)).Elem()},
				},
				{
					"WriteDocumentElement Error",
					map[string]interface{}{"foo": "bar"},
					&EncodeContext{Registry: NewRegistryBuilder().Build()},
					&llValueReaderWriter{err: errors.New("wde error"), errAfter: llvrwWriteDocumentElement},
					llvrwWriteDocumentElement,
					errors.New("wde error"),
				},
				{
					"EncodeValue Error",
					map[string]interface{}{"foo": "bar"},
					&EncodeContext{Registry: NewRegistryBuilder().Build()},
					&llValueReaderWriter{err: errors.New("ev error"), errAfter: llvrwWriteString},
					llvrwWriteString,
					errors.New("ev error"),
				},
			},
		},
		{
			"SliceEncodeValue",
			ValueEncoderFunc(dve.SliceEncodeValue),
			[]subtest{
				{
					"wrong kind",
					wrong,
					nil,
					nil,
					llvrwNothing,
					errors.New("SliceEncodeValue can only encode arrays and slices"),
				},
				{
					"WriteArray Error",
					[]string{},
					nil,
					&llValueReaderWriter{err: errors.New("wa error"), errAfter: llvrwWriteArray},
					llvrwWriteArray,
					errors.New("wa error"),
				},
				{
					"Lookup Error",
					[]interface{}{},
					&EncodeContext{Registry: NewEmptyRegistryBuilder().Build()},
					&llValueReaderWriter{},
					llvrwWriteArray,
					ErrNoEncoder{Type: reflect.TypeOf((*interface{})(nil)).Elem()},
				},
				{
					"WriteArrayElement Error",
					[]string{"foo"},
					&EncodeContext{Registry: NewRegistryBuilder().Build()},
					&llValueReaderWriter{err: errors.New("wae error"), errAfter: llvrwWriteArrayElement},
					llvrwWriteArrayElement,
					errors.New("wae error"),
				},
				{
					"EncodeValue Error",
					[]string{"foo"},
					&EncodeContext{Registry: NewRegistryBuilder().Build()},
					&llValueReaderWriter{err: errors.New("ev error"), errAfter: llvrwWriteString},
					llvrwWriteString,
					errors.New("ev error"),
				},
			},
		},
		{
			"BinaryEncodeValue",
			ValueEncoderFunc(dve.BinaryEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "BinaryEncodeValue",
						Types:    []interface{}{bson.Binary{}, (*bson.Binary)(nil)},
						Received: wrong,
					},
				},
				{"Binary/success", bson.Binary{Data: []byte{0x01, 0x02}, Subtype: 0xFF}, nil, nil, llvrwWriteBinaryWithSubtype, nil},
				{"*Binary/success", &bson.Binary{Data: []byte{0x01, 0x02}, Subtype: 0xFF}, nil, nil, llvrwWriteBinaryWithSubtype, nil},
			},
		},
		{
			"UndefinedEncodeValue",
			ValueEncoderFunc(dve.UndefinedEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "UndefinedEncodeValue",
						Types:    []interface{}{bson.Undefinedv2{}, (*bson.Undefinedv2)(nil)},
						Received: wrong,
					},
				},
				{"Undefined/success", bson.Undefinedv2{}, nil, nil, llvrwWriteUndefined, nil},
				{"*Undefined/success", &bson.Undefinedv2{}, nil, nil, llvrwWriteUndefined, nil},
			},
		},
		{
			"ObjectIDEncodeValue",
			ValueEncoderFunc(dve.ObjectIDEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "ObjectIDEncodeValue",
						Types:    []interface{}{objectid.ObjectID{}, (*objectid.ObjectID)(nil)},
						Received: wrong,
					},
				},
				{
					"objectid.ObjectID/success",
					objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					nil, nil, llvrwWriteObjectID, nil,
				},
				{
					"*objectid.ObjectID/success",
					&objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					nil, nil, llvrwWriteObjectID, nil,
				},
			},
		},
		{
			"DateTimeEncodeValue",
			ValueEncoderFunc(dve.DateTimeEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "DateTimeEncodeValue",
						Types:    []interface{}{bson.DateTime(0), (*bson.DateTime)(nil)},
						Received: wrong,
					},
				},
				{"DateTime/success", bson.DateTime(1234567890), nil, nil, llvrwWriteDateTime, nil},
				{"*DateTime/success", pdatetime, nil, nil, llvrwWriteDateTime, nil},
			},
		},
		{
			"NullEncodeValue",
			ValueEncoderFunc(dve.NullEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "NullEncodeValue",
						Types:    []interface{}{bson.Nullv2{}, (*bson.Nullv2)(nil)},
						Received: wrong,
					},
				},
				{"Null/success", bson.Nullv2{}, nil, nil, llvrwWriteNull, nil},
				{"*Null/success", &bson.Nullv2{}, nil, nil, llvrwWriteNull, nil},
			},
		},
		{
			"RegexEncodeValue",
			ValueEncoderFunc(dve.RegexEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "RegexEncodeValue",
						Types:    []interface{}{bson.Regex{}, (*bson.Regex)(nil)},
						Received: wrong,
					},
				},
				{"Regex/success", bson.Regex{Pattern: "foo", Options: "bar"}, nil, nil, llvrwWriteRegex, nil},
				{"*Regex/success", &bson.Regex{Pattern: "foo", Options: "bar"}, nil, nil, llvrwWriteRegex, nil},
			},
		},
		{
			"DBPointerEncodeValue",
			ValueEncoderFunc(dve.DBPointerEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "DBPointerEncodeValue",
						Types:    []interface{}{bson.DBPointer{}, (*bson.DBPointer)(nil)},
						Received: wrong,
					},
				},
				{
					"DBPointer/success",
					bson.DBPointer{
						DB:      "foobar",
						Pointer: objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					},
					nil, nil, llvrwWriteDBPointer, nil,
				},
				{
					"*DBPointer/success",
					&bson.DBPointer{
						DB:      "foobar",
						Pointer: objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					},
					nil, nil, llvrwWriteDBPointer, nil,
				},
			},
		},
		{
			"CodeWithScopeEncodeValue",
			ValueEncoderFunc(dve.CodeWithScopeEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "CodeWithScopeEncodeValue",
						Types:    []interface{}{bson.CodeWithScope{}, (*bson.CodeWithScope)(nil)},
						Received: wrong,
					},
				},
				{
					"WriteCodeWithScope error",
					bson.CodeWithScope{},
					nil,
					&llValueReaderWriter{err: errors.New("wcws error"), errAfter: llvrwWriteCodeWithScope},
					llvrwWriteCodeWithScope,
					errors.New("wcws error"),
				},
				{
					"CodeWithScope/success",
					bson.CodeWithScope{
						Code:  "var hello = 'world';",
						Scope: bson.NewDocument(),
					},
					nil, nil, llvrwWriteDocumentEnd, nil,
				},
				{
					"*CodeWithScope/success",
					&bson.CodeWithScope{
						Code:  "var hello = 'world';",
						Scope: bson.NewDocument(),
					},
					nil, nil, llvrwWriteDocumentEnd, nil,
				},
			},
		},
		{
			"TimestampEncodeValue",
			ValueEncoderFunc(dve.TimestampEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "TimestampEncodeValue",
						Types:    []interface{}{bson.Timestamp{}, (*bson.Timestamp)(nil)},
						Received: wrong,
					},
				},
				{"Timestamp/success", bson.Timestamp{T: 12345, I: 67890}, nil, nil, llvrwWriteTimestamp, nil},
				{"*Timestamp/success", &bson.Timestamp{T: 12345, I: 67890}, nil, nil, llvrwWriteTimestamp, nil},
			},
		},
		{
			"Decimal128EncodeValue",
			ValueEncoderFunc(dve.Decimal128EncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "Decimal128EncodeValue",
						Types:    []interface{}{decimal.Decimal128{}, (*decimal.Decimal128)(nil)},
						Received: wrong,
					},
				},
				{"Decimal128/success", d128, nil, nil, llvrwWriteDecimal128, nil},
				{"*Decimal128/success", &d128, nil, nil, llvrwWriteDecimal128, nil},
			},
		},
		{
			"MinKeyEncodeValue",
			ValueEncoderFunc(dve.MinKeyEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "MinKeyEncodeValue",
						Types:    []interface{}{bson.MinKeyv2{}, (*bson.MinKeyv2)(nil)},
						Received: wrong,
					},
				},
				{"MinKey/success", bson.MinKeyv2{}, nil, nil, llvrwWriteMinKey, nil},
				{"*MinKey/success", &bson.MinKeyv2{}, nil, nil, llvrwWriteMinKey, nil},
			},
		},
		{
			"MaxKeyEncodeValue",
			ValueEncoderFunc(dve.MaxKeyEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "MaxKeyEncodeValue",
						Types:    []interface{}{bson.MaxKeyv2{}, (*bson.MaxKeyv2)(nil)},
						Received: wrong,
					},
				},
				{"MaxKey/success", bson.MaxKeyv2{}, nil, nil, llvrwWriteMaxKey, nil},
				{"*MaxKey/success", &bson.MaxKeyv2{}, nil, nil, llvrwWriteMaxKey, nil},
			},
		},
		{
			"elementEncodeValue",
			ValueEncoderFunc(dve.elementEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{Name: "elementEncodeValue", Types: []interface{}{(*bson.Element)(nil)}, Received: wrong},
				},
				{"invalid element", (*bson.Element)(nil), nil, nil, llvrwNothing, bson.ErrNilElement},
				{
					"success",
					bson.EC.Null("foo"),
					&EncodeContext{Registry: NewRegistryBuilder().Build()},
					&llValueReaderWriter{},
					llvrwWriteNull,
					nil,
				},
			},
		},
		{
			"ValueEncodeValue",
			ValueEncoderFunc(dve.ValueEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{Name: "ValueEncodeValue", Types: []interface{}{(*bson.Value)(nil)}, Received: wrong},
				},
				{"invalid value", &bson.Value{}, nil, nil, llvrwNothing, bson.ErrUninitializedElement},
				{
					"success",
					bson.VC.Null(),
					&EncodeContext{Registry: NewRegistryBuilder().Build()},
					&llValueReaderWriter{},
					llvrwWriteNull,
					nil,
				},
			},
		},
		{
			"JSONNumberEncodeValue",
			ValueEncoderFunc(dve.JSONNumberEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "JSONNumberEncodeValue",
						Types:    []interface{}{json.Number(""), (*json.Number)(nil)},
						Received: wrong,
					},
				},
				{
					"json.Number/invalid",
					json.Number("hello world"),
					nil, nil, llvrwNothing, errors.New(`strconv.ParseFloat: parsing "hello world": invalid syntax`),
				},
				{
					"json.Number/int64/success",
					json.Number("1234567890"),
					nil, nil, llvrwWriteInt64, nil,
				},
				{
					"json.Number/float64/success",
					json.Number("3.14159"),
					nil, nil, llvrwWriteDouble, nil,
				},
				{
					"*json.Number/int64/success",
					pjsnum,
					nil, nil, llvrwWriteDouble, nil,
				},
			},
		},
		{
			"URLEncodeValue",
			ValueEncoderFunc(dve.URLEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "URLEncodeValue",
						Types:    []interface{}{url.URL{}, (*url.URL)(nil)},
						Received: wrong,
					},
				},
				{"url.URL", url.URL{Scheme: "http", Host: "example.com"}, nil, nil, llvrwWriteString, nil},
				{"*url.URL", &url.URL{Scheme: "http", Host: "example.com"}, nil, nil, llvrwWriteString, nil},
			},
		},
		{
			"ReaderEncodeValue",
			ValueEncoderFunc(dve.ReaderEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{Name: "ReaderEncodeValue", Types: []interface{}{bson.Reader{}}, Received: wrong},
				},
				{
					"WriteDocument Error",
					bson.Reader{},
					nil,
					&llValueReaderWriter{err: errors.New("wd error"), errAfter: llvrwWriteDocument},
					llvrwWriteDocument,
					errors.New("wd error"),
				},
				{
					"Reader.Iterator Error",
					bson.Reader{0xFF, 0x00, 0x00, 0x00, 0x00},
					nil,
					&llValueReaderWriter{},
					llvrwWriteDocument,
					bson.ErrInvalidLength,
				},
				{
					"WriteDocumentElement Error",
					bson.Reader(bytesFromDoc(bson.NewDocument(bson.EC.Null("foo")))),
					nil,
					&llValueReaderWriter{err: errors.New("wde error"), errAfter: llvrwWriteDocumentElement},
					llvrwWriteDocumentElement,
					errors.New("wde error"),
				},
				{
					"encodeValue error",
					bson.Reader(bytesFromDoc(bson.NewDocument(bson.EC.Null("foo")))),
					nil,
					&llValueReaderWriter{err: errors.New("ev error"), errAfter: llvrwWriteNull},
					llvrwWriteNull,
					errors.New("ev error"),
				},
				{
					"iterator error",
					bson.Reader{0x0C, 0x00, 0x00, 0x00, 0x01, 'f', 'o', 'o', 0x00, 0x01, 0x02, 0x03},
					nil,
					&llValueReaderWriter{},
					llvrwWriteDocument,
					bson.NewErrTooSmall(),
				},
			},
		},
		{
			"ByteSliceEncodeValue",
			ValueEncoderFunc(dve.ByteSliceEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "ByteSliceEncodeValue",
						Types:    []interface{}{[]byte{}, (*[]byte)(nil)},
						Received: wrong,
					},
				},
				{"[]byte", []byte{0x01, 0x02, 0x03}, nil, nil, llvrwWriteBinary, nil},
				{"*[]byte", &([]byte{0x01, 0x02, 0x03}), nil, nil, llvrwWriteBinary, nil},
			},
		},
		{
			"ValueMarshalerEncodeValue",
			ValueEncoderFunc(dve.ValueMarshalerEncodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueEncoderError{
						Name:     "ValueMarshalerEncodeValue",
						Types:    []interface{}{(ValueMarshaler)(nil)},
						Received: wrong,
					},
				},
				{
					"MarshalBSONValue error",
					testValueMarshaler{err: errors.New("mbsonv error")},
					nil,
					nil,
					llvrwNothing,
					errors.New("mbsonv error"),
				},
				{
					"Copy error",
					testValueMarshaler{},
					nil,
					nil,
					llvrwNothing,
					fmt.Errorf("Cannot copy unknown BSON type %s", bson.Type(0)),
				},
				{
					"success",
					testValueMarshaler{t: bson.TypeString, buf: []byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00}},
					nil,
					nil,
					llvrwWriteString,
					nil,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, subtest := range tc.subtests {
				t.Run(subtest.name, func(t *testing.T) {
					var ec EncodeContext
					if subtest.ectx != nil {
						ec = *subtest.ectx
					}
					llvrw := new(llValueReaderWriter)
					if subtest.llvrw != nil {
						llvrw = subtest.llvrw
					}
					llvrw.t = t
					err := tc.ve.EncodeValue(ec, llvrw, subtest.val)
					if !compareErrors(err, subtest.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, subtest.err)
					}
					invoked := llvrw.invoked
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
			want := ValueEncoderError{Name: "DocumentEncodeValue", Types: []interface{}{(*bson.Document)(nil)}, Received: val}
			got := (DefaultValueEncoders{}).DocumentEncodeValue(EncodeContext{}, nil, val)
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("WriteDocument Error", func(t *testing.T) {
			want := errors.New("WriteDocument Error")
			llvrw := &llValueReaderWriter{
				t:        t,
				err:      want,
				errAfter: llvrwWriteDocument,
			}
			got := (DefaultValueEncoders{}).DocumentEncodeValue(EncodeContext{}, llvrw, bson.NewDocument())
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("encodeDocument errors", func(t *testing.T) {
			ec := EncodeContext{}
			err := errors.New("encodeDocument error")
			oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			badelem := &bson.Element{}
			testCases := []struct {
				name  string
				ec    EncodeContext
				llvrw *llValueReaderWriter
				doc   *bson.Document
				err   error
			}{
				{
					"WriteDocumentElement",
					ec,
					&llValueReaderWriter{t: t, err: errors.New("wde error"), errAfter: llvrwWriteDocumentElement},
					bson.NewDocument(bson.EC.Null("foo")),
					errors.New("wde error"),
				},
				{
					"WriteDouble", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteDouble},
					bson.NewDocument(bson.EC.Double("foo", 3.14159)), err,
				},
				{
					"WriteString", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteString},
					bson.NewDocument(bson.EC.String("foo", "bar")), err,
				},
				{
					"WriteDocument (Lookup)", EncodeContext{Registry: NewEmptyRegistryBuilder().Build()},
					&llValueReaderWriter{t: t},
					bson.NewDocument(bson.EC.SubDocument("foo", bson.NewDocument(bson.EC.Null("bar")))),
					ErrNoEncoder{Type: tDocument},
				},
				{
					"WriteArray (Lookup)", EncodeContext{Registry: NewEmptyRegistryBuilder().Build()},
					&llValueReaderWriter{t: t},
					bson.NewDocument(bson.EC.Array("foo", bson.NewArray(bson.VC.Null()))),
					ErrNoEncoder{Type: tArray},
				},
				{
					"WriteBinary", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteBinaryWithSubtype},
					bson.NewDocument(bson.EC.BinaryWithSubtype("foo", []byte{0x01, 0x02, 0x03}, 0xFF)), err,
				},
				{
					"WriteUndefined", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteUndefined},
					bson.NewDocument(bson.EC.Undefined("foo")), err,
				},
				{
					"WriteObjectID", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteObjectID},
					bson.NewDocument(bson.EC.ObjectID("foo", oid)), err,
				},
				{
					"WriteBoolean", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteBoolean},
					bson.NewDocument(bson.EC.Boolean("foo", true)), err,
				},
				{
					"WriteDateTime", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteDateTime},
					bson.NewDocument(bson.EC.DateTime("foo", 1234567890)), err,
				},
				{
					"WriteNull", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteNull},
					bson.NewDocument(bson.EC.Null("foo")), err,
				},
				{
					"WriteRegex", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteRegex},
					bson.NewDocument(bson.EC.Regex("foo", "bar", "baz")), err,
				},
				{
					"WriteDBPointer", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteDBPointer},
					bson.NewDocument(bson.EC.DBPointer("foo", "bar", oid)), err,
				},
				{
					"WriteJavascript", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteJavascript},
					bson.NewDocument(bson.EC.JavaScript("foo", "var hello = 'world';")), err,
				},
				{
					"WriteSymbol", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteSymbol},
					bson.NewDocument(bson.EC.Symbol("foo", "symbolbaz")), err,
				},
				{
					"WriteCodeWithScope (Lookup)", EncodeContext{Registry: NewEmptyRegistryBuilder().Build()},
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteCodeWithScope},
					bson.NewDocument(bson.EC.CodeWithScope("foo", "var hello = 'world';", bson.NewDocument(bson.EC.Null("bar")))),
					err,
				},
				{
					"WriteInt32", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteInt32},
					bson.NewDocument(bson.EC.Int32("foo", 12345)), err,
				},
				{
					"WriteInt64", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteInt64},
					bson.NewDocument(bson.EC.Int64("foo", 1234567890)), err,
				},
				{
					"WriteTimestamp", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteTimestamp},
					bson.NewDocument(bson.EC.Timestamp("foo", 10, 20)), err,
				},
				{
					"WriteDecimal128", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteDecimal128},
					bson.NewDocument(bson.EC.Decimal128("foo", decimal.NewDecimal128(10, 20))), err,
				},
				{
					"WriteMinKey", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteMinKey},
					bson.NewDocument(bson.EC.MinKey("foo")), err,
				},
				{
					"WriteMaxKey", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteMaxKey},
					bson.NewDocument(bson.EC.MaxKey("foo")), err,
				},
				{
					"Invalid Type", ec,
					&llValueReaderWriter{t: t, bsontype: bson.Type(0)},
					bson.NewDocument(badelem),
					bson.ErrUninitializedElement,
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					err := (DefaultValueEncoders{}).DocumentEncodeValue(tc.ec, tc.llvrw, tc.doc)
					if !compareErrors(err, tc.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, tc.err)
					}
				})
			}
		})

		t.Run("success", func(t *testing.T) {
			oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			d128 := decimal.NewDecimal128(10, 20)
			want := bson.NewDocument(
				bson.EC.Double("a", 3.14159), bson.EC.String("b", "foo"), bson.EC.SubDocumentFromElements("c", bson.EC.Null("aa")),
				bson.EC.ArrayFromElements("d", bson.VC.Null()),
				bson.EC.BinaryWithSubtype("e", []byte{0x01, 0x02, 0x03}, 0xFF), bson.EC.Undefined("f"),
				bson.EC.ObjectID("g", oid), bson.EC.Boolean("h", true), bson.EC.DateTime("i", 1234567890), bson.EC.Null("j"), bson.EC.Regex("k", "foo", "bar"),
				bson.EC.DBPointer("l", "foobar", oid), bson.EC.JavaScript("m", "var hello = 'world';"), bson.EC.Symbol("n", "bazqux"),
				bson.EC.CodeWithScope("o", "var hello = 'world';", bson.NewDocument(bson.EC.Null("ab"))), bson.EC.Int32("p", 12345),
				bson.EC.Timestamp("q", 10, 20), bson.EC.Int64("r", 1234567890), bson.EC.Decimal128("s", d128), bson.EC.MinKey("t"), bson.EC.MaxKey("u"),
			)
			got := bson.NewDocument()
			ec := EncodeContext{Registry: NewRegistryBuilder().Build()}
			err := (DefaultValueEncoders{}).DocumentEncodeValue(ec, newDocumentValueWriter(got), want)
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
			want := ValueEncoderError{Name: "ArrayEncodeValue", Types: []interface{}{(*bson.Array)(nil)}, Received: val}
			got := (DefaultValueEncoders{}).ArrayEncodeValue(EncodeContext{}, nil, val)
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("WriteArray Error", func(t *testing.T) {
			want := errors.New("WriteArray Error")
			llvrw := &llValueReaderWriter{
				t:        t,
				err:      want,
				errAfter: llvrwWriteArray,
			}
			got := (DefaultValueEncoders{}).ArrayEncodeValue(EncodeContext{}, llvrw, bson.NewArray())
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("encode array errors", func(t *testing.T) {
			ec := EncodeContext{}
			err := errors.New("encode array error")
			oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			badval := &bson.Value{}
			testCases := []struct {
				name  string
				ec    EncodeContext
				llvrw *llValueReaderWriter
				arr   *bson.Array
				err   error
			}{
				{
					"WriteDocumentElement",
					ec,
					&llValueReaderWriter{t: t, err: errors.New("wde error"), errAfter: llvrwWriteArrayElement},
					bson.NewArray(bson.VC.Null()),
					errors.New("wde error"),
				},
				{
					"WriteDouble", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteDouble},
					bson.NewArray(bson.VC.Double(3.14159)), err,
				},
				{
					"WriteString", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteString},
					bson.NewArray(bson.VC.String("bar")), err,
				},
				{
					"WriteDocument (Lookup)", EncodeContext{Registry: NewEmptyRegistryBuilder().Build()},
					&llValueReaderWriter{t: t},
					bson.NewArray(bson.VC.Document(bson.NewDocument(bson.EC.Null("bar")))),
					ErrNoEncoder{Type: tDocument},
				},
				{
					"WriteArray (Lookup)", EncodeContext{Registry: NewEmptyRegistryBuilder().Build()},
					&llValueReaderWriter{t: t},
					bson.NewArray(bson.VC.Array(bson.NewArray(bson.VC.Null()))),
					ErrNoEncoder{Type: tArray},
				},
				{
					"WriteBinary", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteBinaryWithSubtype},
					bson.NewArray(bson.VC.BinaryWithSubtype([]byte{0x01, 0x02, 0x03}, 0xFF)), err,
				},
				{
					"WriteUndefined", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteUndefined},
					bson.NewArray(bson.VC.Undefined()), err,
				},
				{
					"WriteObjectID", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteObjectID},
					bson.NewArray(bson.VC.ObjectID(oid)), err,
				},
				{
					"WriteBoolean", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteBoolean},
					bson.NewArray(bson.VC.Boolean(true)), err,
				},
				{
					"WriteDateTime", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteDateTime},
					bson.NewArray(bson.VC.DateTime(1234567890)), err,
				},
				{
					"WriteNull", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteNull},
					bson.NewArray(bson.VC.Null()), err,
				},
				{
					"WriteRegex", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteRegex},
					bson.NewArray(bson.VC.Regex("bar", "baz")), err,
				},
				{
					"WriteDBPointer", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteDBPointer},
					bson.NewArray(bson.VC.DBPointer("bar", oid)), err,
				},
				{
					"WriteJavascript", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteJavascript},
					bson.NewArray(bson.VC.JavaScript("var hello = 'world';")), err,
				},
				{
					"WriteSymbol", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteSymbol},
					bson.NewArray(bson.VC.Symbol("symbolbaz")), err,
				},
				{
					"WriteCodeWithScope (Lookup)", EncodeContext{Registry: NewEmptyRegistryBuilder().Build()},
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteCodeWithScope},
					bson.NewArray(bson.VC.CodeWithScope("var hello = 'world';", bson.NewDocument(bson.EC.Null("bar")))),
					err,
				},
				{
					"WriteInt32", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteInt32},
					bson.NewArray(bson.VC.Int32(12345)), err,
				},
				{
					"WriteInt64", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteInt64},
					bson.NewArray(bson.VC.Int64(1234567890)), err,
				},
				{
					"WriteTimestamp", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteTimestamp},
					bson.NewArray(bson.VC.Timestamp(10, 20)), err,
				},
				{
					"WriteDecimal128", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteDecimal128},
					bson.NewArray(bson.VC.Decimal128(decimal.NewDecimal128(10, 20))), err,
				},
				{
					"WriteMinKey", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteMinKey},
					bson.NewArray(bson.VC.MinKey()), err,
				},
				{
					"WriteMaxKey", ec,
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteMaxKey},
					bson.NewArray(bson.VC.MaxKey()), err,
				},
				{
					"Invalid Type", ec,
					&llValueReaderWriter{t: t, bsontype: bson.Type(0)},
					bson.NewArray(badval),
					bson.ErrUninitializedElement,
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					err := (DefaultValueEncoders{}).ArrayEncodeValue(tc.ec, tc.llvrw, tc.arr)
					if !compareErrors(err, tc.err) {
						t.Errorf("Errors do not match. got %v; want %v", err, tc.err)
					}
				})
			}
		})

		t.Run("success", func(t *testing.T) {
			oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
			d128 := decimal.NewDecimal128(10, 20)
			want := bson.NewArray(
				bson.VC.Double(3.14159), bson.VC.String("foo"), bson.VC.DocumentFromElements(bson.EC.Null("aa")),
				bson.VC.ArrayFromValues(bson.VC.Null()),
				bson.VC.BinaryWithSubtype([]byte{0x01, 0x02, 0x03}, 0xFF), bson.VC.Undefined(),
				bson.VC.ObjectID(oid), bson.VC.Boolean(true), bson.VC.DateTime(1234567890), bson.VC.Null(), bson.VC.Regex("foo", "bar"),
				bson.VC.DBPointer("foobar", oid), bson.VC.JavaScript("var hello = 'world';"), bson.VC.Symbol("bazqux"),
				bson.VC.CodeWithScope("var hello = 'world';", bson.NewDocument(bson.EC.Null("ab"))), bson.VC.Int32(12345),
				bson.VC.Timestamp(10, 20), bson.VC.Int64(1234567890), bson.VC.Decimal128(d128), bson.VC.MinKey(), bson.VC.MaxKey(),
			)

			ec := EncodeContext{Registry: NewRegistryBuilder().Build()}

			doc := bson.NewDocument()
			dvw := newDocumentValueWriter(doc)
			dr, err := dvw.WriteDocument()
			noerr(t, err)
			vr, err := dr.WriteDocumentElement("foo")
			noerr(t, err)

			err = (DefaultValueEncoders{}).ArrayEncodeValue(ec, vr, want)
			noerr(t, err)

			got := doc.Lookup("foo").MutableArray()
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
				docToBytes(bson.NewDocument(bson.EC.ObjectID("foo", oid))),
				nil,
			},
			{
				"map[string][]*Element",
				map[string][]*bson.Element{"Z": {bson.EC.Int32("A", 1), bson.EC.Int32("B", 2), bson.EC.Int32("EC", 3)}},
				docToBytes(bson.NewDocument(
					bson.EC.SubDocumentFromElements("Z", bson.EC.Int32("A", 1), bson.EC.Int32("B", 2), bson.EC.Int32("EC", 3)),
				)),
				nil,
			},
			{
				"map[string][]*Value",
				map[string][]*bson.Value{"Z": {bson.VC.Int32(1), bson.VC.Int32(2), bson.VC.Int32(3)}},
				docToBytes(bson.NewDocument(
					bson.EC.ArrayFromElements("Z", bson.VC.Int32(1), bson.VC.Int32(2), bson.VC.Int32(3)),
				)),
				nil,
			},
			{
				"map[string]*Element",
				map[string]*bson.Element{"Z": bson.EC.Int32("Z", 12345)},
				docToBytes(bson.NewDocument(
					bson.EC.Int32("Z", 12345),
				)),
				nil,
			},
			{
				"map[string]*Document",
				map[string]*bson.Document{"Z": bson.NewDocument(bson.EC.Null("foo"))},
				docToBytes(bson.NewDocument(
					bson.EC.SubDocumentFromElements("Z", bson.EC.Null("foo")),
				)),
				nil,
			},
			{
				"map[string]Reader",
				map[string]bson.Reader{"Z": {0x05, 0x00, 0x00, 0x00, 0x00}},
				docToBytes(bson.NewDocument(
					bson.EC.SubDocumentFromReader("Z", bson.Reader{0x05, 0x00, 0x00, 0x00, 0x00}),
				)),
				nil,
			},
			{
				"map[string][]int32",
				map[string][]int32{"Z": {1, 2, 3}},
				docToBytes(bson.NewDocument(
					bson.EC.ArrayFromElements("Z", bson.VC.Int32(1), bson.VC.Int32(2), bson.VC.Int32(3)),
				)),
				nil,
			},
			{
				"map[string][]objectid.ObjectID",
				map[string][]objectid.ObjectID{"Z": oids},
				docToBytes(bson.NewDocument(
					bson.EC.ArrayFromElements("Z", bson.VC.ObjectID(oids[0]), bson.VC.ObjectID(oids[1]), bson.VC.ObjectID(oids[2])),
				)),
				nil,
			},
			{
				"map[string][]json.Number(int64)",
				map[string][]json.Number{"Z": {json.Number("5"), json.Number("10")}},
				docToBytes(bson.NewDocument(
					bson.EC.ArrayFromElements("Z", bson.VC.Int64(5), bson.VC.Int64(10)),
				)),
				nil,
			},
			{
				"map[string][]json.Number(float64)",
				map[string][]json.Number{"Z": {json.Number("5"), json.Number("10.1")}},
				docToBytes(bson.NewDocument(
					bson.EC.ArrayFromElements("Z", bson.VC.Int64(5), bson.VC.Double(10.1)),
				)),
				nil,
			},
			{
				"map[string][]*url.URL",
				map[string][]*url.URL{"Z": {murl}},
				docToBytes(bson.NewDocument(
					bson.EC.ArrayFromElements("Z", bson.VC.String(murl.String())),
				)),
				nil,
			},
			{
				"map[string][]decimal.Decimal128",
				map[string][]decimal.Decimal128{"Z": {decimal128}},
				docToBytes(bson.NewDocument(
					bson.EC.ArrayFromElements("Z", bson.VC.Decimal128(decimal128)),
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
				docToBytes(bson.NewDocument()),
				nil,
			},
			{
				"omitempty",
				struct {
					A string `bson:",omitempty"`
				}{
					A: "",
				},
				docToBytes(bson.NewDocument()),
				nil,
			},
			{
				"omitempty, empty time",
				struct {
					A time.Time `bson:",omitempty"`
				}{
					A: time.Time{},
				},
				docToBytes(bson.NewDocument()),
				nil,
			},
			{
				"no private fields",
				noPrivateFields{a: "should be empty"},
				docToBytes(bson.NewDocument()),
				nil,
			},
			{
				"minsize",
				struct {
					A int64 `bson:",minsize"`
				}{
					A: 12345,
				},
				docToBytes(bson.NewDocument(bson.EC.Int32("a", 12345))),
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
				docToBytes(bson.NewDocument(bson.EC.Int32("a", 12345))),
				nil,
			},
			{
				"inline map",
				struct {
					Foo map[string]string `bson:",inline"`
				}{
					Foo: map[string]string{"foo": "bar"},
				},
				docToBytes(bson.NewDocument(bson.EC.String("foo", "bar"))),
				nil,
			},
			{
				"alternate name bson:name",
				struct {
					A string `bson:"foo"`
				}{
					A: "bar",
				},
				docToBytes(bson.NewDocument(bson.EC.String("foo", "bar"))),
				nil,
			},
			{
				"alternate name",
				struct {
					A string `bson:"foo"`
				}{
					A: "bar",
				},
				docToBytes(bson.NewDocument(bson.EC.String("foo", "bar"))),
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
				docToBytes(bson.NewDocument(bson.EC.String("a", "bar"))),
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
					N  *bson.Element
					O  *bson.Document
					P  bson.Reader
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
					N:  bson.EC.Null("n"),
					O:  bson.NewDocument(bson.EC.Int64("countdown", 9876543210)),
					P:  bson.Reader{0x05, 0x00, 0x00, 0x00, 0x00},
					Q:  oid,
					T:  nil,
					Y:  json.Number("5"),
					Z:  now,
					AA: json.Number("10.1"),
					AB: murl,
					AC: decimal128,
					AD: &now,
					AE: testValueMarshaler{t: bson.TypeString, buf: llbson.AppendString(nil, "hello, world")},
				},
				docToBytes(bson.NewDocument(
					bson.EC.Boolean("a", true),
					bson.EC.Int32("b", 123),
					bson.EC.Int64("c", 456),
					bson.EC.Int32("d", 789),
					bson.EC.Int64("e", 101112),
					bson.EC.Double("f", 3.14159),
					bson.EC.String("g", "Hello, world"),
					bson.EC.SubDocumentFromElements("h", bson.EC.String("foo", "bar")),
					bson.EC.Binary("i", []byte{0x01, 0x02, 0x03}),
					bson.EC.ArrayFromElements("k", bson.VC.String("baz"), bson.VC.String("qux")),
					bson.EC.SubDocumentFromElements("l", bson.EC.String("m", "foobar")),
					bson.EC.Null("n"),
					bson.EC.SubDocumentFromElements("o", bson.EC.Int64("countdown", 9876543210)),
					bson.EC.SubDocumentFromElements("p"),
					bson.EC.ObjectID("q", oid),
					bson.EC.Null("t"),
					bson.EC.Int64("y", 5),
					bson.EC.DateTime("z", now.UnixNano()/int64(time.Millisecond)),
					bson.EC.Double("aa", 10.1),
					bson.EC.String("ab", murl.String()),
					bson.EC.Decimal128("ac", decimal128),
					bson.EC.DateTime("ad", now.UnixNano()/int64(time.Millisecond)),
					bson.EC.String("ae", "hello, world"),
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
					O  []*bson.Element
					P  []*bson.Document
					Q  []bson.Reader
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
					O:  []*bson.Element{bson.EC.Null("N")},
					P:  []*bson.Document{bson.NewDocument(bson.EC.Int64("countdown", 9876543210))},
					Q:  []bson.Reader{{0x05, 0x00, 0x00, 0x00, 0x00}},
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
						{t: bson.TypeString, buf: llbson.AppendString(nil, "hello")},
						{t: bson.TypeString, buf: llbson.AppendString(nil, "world")},
					},
				},
				docToBytes(bson.NewDocument(
					bson.EC.ArrayFromElements("a", bson.VC.Boolean(true)),
					bson.EC.ArrayFromElements("b", bson.VC.Int32(123)),
					bson.EC.ArrayFromElements("c", bson.VC.Int64(456)),
					bson.EC.ArrayFromElements("d", bson.VC.Int32(789)),
					bson.EC.ArrayFromElements("e", bson.VC.Int64(101112)),
					bson.EC.ArrayFromElements("f", bson.VC.Double(3.14159)),
					bson.EC.ArrayFromElements("g", bson.VC.String("Hello, world")),
					bson.EC.ArrayFromElements("h", bson.VC.DocumentFromElements(bson.EC.String("foo", "bar"))),
					bson.EC.ArrayFromElements("i", bson.VC.Binary([]byte{0x01, 0x02, 0x03})),
					bson.EC.ArrayFromElements("k", bson.VC.ArrayFromValues(bson.VC.String("baz"), bson.VC.String("qux"))),
					bson.EC.ArrayFromElements("l", bson.VC.DocumentFromElements(bson.EC.String("m", "foobar"))),
					bson.EC.ArrayFromElements("n", bson.VC.ArrayFromValues(bson.VC.String("foo"), bson.VC.String("bar"))),
					bson.EC.SubDocumentFromElements("o", bson.EC.Null("N")),
					bson.EC.ArrayFromElements("p", bson.VC.DocumentFromElements(bson.EC.Int64("countdown", 9876543210))),
					bson.EC.ArrayFromElements("q", bson.VC.DocumentFromElements()),
					bson.EC.ArrayFromElements("r", bson.VC.ObjectID(oids[0]), bson.VC.ObjectID(oids[1]), bson.VC.ObjectID(oids[2])),
					bson.EC.Null("t"),
					bson.EC.Null("w"),
					bson.EC.Array("x", bson.NewArray()),
					bson.EC.ArrayFromElements("y", bson.VC.Document(bson.NewDocument())),
					bson.EC.ArrayFromElements("z", bson.VC.DateTime(now.UnixNano()/int64(time.Millisecond)), bson.VC.DateTime(now.UnixNano()/int64(time.Millisecond))),
					bson.EC.ArrayFromElements("aa", bson.VC.Int64(5), bson.VC.Double(10.10)),
					bson.EC.ArrayFromElements("ab", bson.VC.String(murl.String())),
					bson.EC.ArrayFromElements("ac", bson.VC.Decimal128(decimal128)),
					bson.EC.ArrayFromElements("ad", bson.VC.DateTime(now.UnixNano()/int64(time.Millisecond)), bson.VC.DateTime(now.UnixNano()/int64(time.Millisecond))),
					bson.EC.ArrayFromElements("ae", bson.VC.String("hello"), bson.VC.String("world")),
				)),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				b := make([]byte, 0, 512)
				vw := newValueWriterFromSlice(b)
				enc, err := NewEncoder(NewRegistryBuilder().Build(), vw)
				noerr(t, err)
				err = enc.Encode(tc.value)
				if err != tc.err {
					t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
				}
				b = vw.buf
				if diff := cmp.Diff(b, tc.b); diff != "" {
					t.Errorf("Bytes written differ: (-got +want)\n%s", diff)
					t.Errorf("Bytes\ngot: %v\nwant:%v\n", b, tc.b)
					t.Errorf("Readers\ngot: %v\nwant:%v\n", bson.Reader(b), bson.Reader(tc.b))
				}
			})
		}
	})
}

type testValueMarshaler struct {
	t   bson.Type
	buf []byte
	err error
}

func (tvm testValueMarshaler) MarshalBSONValue() (bson.Type, []byte, error) {
	return tvm.t, tvm.buf, tvm.err
}
