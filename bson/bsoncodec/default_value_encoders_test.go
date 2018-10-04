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
	"github.com/mongodb/mongo-go-driver/bson/bsoncore"
	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
	"github.com/mongodb/mongo-go-driver/bson/bsonrw/bsonrwtest"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
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
	pjsnum := new(json.Number)
	*pjsnum = json.Number("3.14159")
	d128 := decimal.NewDecimal128(12345, 67890)

	type subtest struct {
		name   string
		val    interface{}
		ectx   *EncodeContext
		llvrw  *bsonrwtest.ValueReaderWriter
		invoke bsonrwtest.Invoked
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
					bsonrwtest.Nothing,
					ValueEncoderError{Name: "BooleanEncodeValue", Types: []interface{}{bool(true)}, Received: wrong},
				},
				{"fast path", bool(true), nil, nil, bsonrwtest.WriteBoolean, nil},
				{"reflection path", mybool(true), nil, nil, bsonrwtest.WriteBoolean, nil},
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
					bsonrwtest.Nothing,
					ValueEncoderError{Name: "IntEncodeValue", Types: intAllowedTypes, Received: wrong},
				},
				{"int8/fast path", int8(127), nil, nil, bsonrwtest.WriteInt32, nil},
				{"int16/fast path", int16(32767), nil, nil, bsonrwtest.WriteInt32, nil},
				{"int32/fast path", int32(2147483647), nil, nil, bsonrwtest.WriteInt32, nil},
				{"int64/fast path", int64(1234567890987), nil, nil, bsonrwtest.WriteInt64, nil},
				{"int/fast path", int(1234567), nil, nil, bsonrwtest.WriteInt64, nil},
				{"int64/fast path - minsize", int64(2147483647), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt32, nil},
				{"int/fast path - minsize", int(2147483647), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt32, nil},
				{"int64/fast path - minsize too large", int64(2147483648), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt64, nil},
				{"int/fast path - minsize too large", int(2147483648), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt64, nil},
				{"int8/reflection path", myint8(127), nil, nil, bsonrwtest.WriteInt32, nil},
				{"int16/reflection path", myint16(32767), nil, nil, bsonrwtest.WriteInt32, nil},
				{"int32/reflection path", myint32(2147483647), nil, nil, bsonrwtest.WriteInt32, nil},
				{"int64/reflection path", myint64(1234567890987), nil, nil, bsonrwtest.WriteInt64, nil},
				{"int/reflection path", myint(1234567890987), nil, nil, bsonrwtest.WriteInt64, nil},
				{"int64/reflection path - minsize", myint64(2147483647), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt32, nil},
				{"int/reflection path - minsize", myint(2147483647), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt32, nil},
				{"int64/reflection path - minsize too large", myint64(2147483648), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt64, nil},
				{"int/reflection path - minsize too large", myint(2147483648), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt64, nil},
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
					bsonrwtest.Nothing,
					ValueEncoderError{Name: "UintEncodeValue", Types: uintAllowedEncodeTypes, Received: wrong},
				},
				{"uint8/fast path", uint8(127), nil, nil, bsonrwtest.WriteInt32, nil},
				{"uint16/fast path", uint16(32767), nil, nil, bsonrwtest.WriteInt32, nil},
				{"uint32/fast path", uint32(2147483647), nil, nil, bsonrwtest.WriteInt64, nil},
				{"uint64/fast path", uint64(1234567890987), nil, nil, bsonrwtest.WriteInt64, nil},
				{"uint/fast path", uint(1234567), nil, nil, bsonrwtest.WriteInt64, nil},
				{"uint32/fast path - minsize", uint32(2147483647), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt32, nil},
				{"uint64/fast path - minsize", uint64(2147483647), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt32, nil},
				{"uint/fast path - minsize", uint(2147483647), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt32, nil},
				{"uint32/fast path - minsize too large", uint32(2147483648), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt64, nil},
				{"uint64/fast path - minsize too large", uint64(2147483648), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt64, nil},
				{"uint/fast path - minsize too large", uint(2147483648), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt64, nil},
				{"uint64/fast path - overflow", uint64(1 << 63), nil, nil, bsonrwtest.Nothing, fmt.Errorf("%d overflows int64", uint(1<<63))},
				{"uint/fast path - overflow", uint(1 << 63), nil, nil, bsonrwtest.Nothing, fmt.Errorf("%d overflows int64", uint(1<<63))},
				{"uint8/reflection path", myuint8(127), nil, nil, bsonrwtest.WriteInt32, nil},
				{"uint16/reflection path", myuint16(32767), nil, nil, bsonrwtest.WriteInt32, nil},
				{"uint32/reflection path", myuint32(2147483647), nil, nil, bsonrwtest.WriteInt64, nil},
				{"uint64/reflection path", myuint64(1234567890987), nil, nil, bsonrwtest.WriteInt64, nil},
				{"uint/reflection path", myuint(1234567890987), nil, nil, bsonrwtest.WriteInt64, nil},
				{"uint32/reflection path - minsize", myuint32(2147483647), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt32, nil},
				{"uint64/reflection path - minsize", myuint64(2147483647), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt32, nil},
				{"uint/reflection path - minsize", myuint(2147483647), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt32, nil},
				{"uint32/reflection path - minsize too large", myuint(1 << 31), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt64, nil},
				{"uint64/reflection path - minsize too large", myuint64(1 << 31), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt64, nil},
				{"uint/reflection path - minsize too large", myuint(2147483648), &EncodeContext{MinSize: true}, nil, bsonrwtest.WriteInt64, nil},
				{"uint64/reflection path - overflow", myuint64(1 << 63), nil, nil, bsonrwtest.Nothing, fmt.Errorf("%d overflows int64", uint(1<<63))},
				{"uint/reflection path - overflow", myuint(1 << 63), nil, nil, bsonrwtest.Nothing, fmt.Errorf("%d overflows int64", uint(1<<63))},
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
					bsonrwtest.Nothing,
					ValueEncoderError{Name: "FloatEncodeValue", Types: []interface{}{float32(0), float64(0)}, Received: wrong},
				},
				{"float32/fast path", float32(3.14159), nil, nil, bsonrwtest.WriteDouble, nil},
				{"float64/fast path", float64(3.14159), nil, nil, bsonrwtest.WriteDouble, nil},
				{"float32/reflection path", myfloat32(3.14159), nil, nil, bsonrwtest.WriteDouble, nil},
				{"float64/reflection path", myfloat64(3.14159), nil, nil, bsonrwtest.WriteDouble, nil},
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
					bsonrwtest.Nothing,
					ValueEncoderError{
						Name:     "TimeEncodeValue",
						Types:    []interface{}{time.Time{}, (*time.Time)(nil)},
						Received: wrong,
					},
				},
				{"time.Time", now, nil, nil, bsonrwtest.WriteDateTime, nil},
				{"*time.Time", &now, nil, nil, bsonrwtest.WriteDateTime, nil},
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
					bsonrwtest.Nothing,
					errors.New("MapEncodeValue can only encode maps with string keys"),
				},
				{
					"wrong kind (non-string key)",
					map[int]interface{}{},
					nil,
					nil,
					bsonrwtest.Nothing,
					errors.New("MapEncodeValue can only encode maps with string keys"),
				},
				{
					"WriteDocument Error",
					map[string]interface{}{},
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("wd error"), ErrAfter: bsonrwtest.WriteDocument},
					bsonrwtest.WriteDocument,
					errors.New("wd error"),
				},
				{
					"Lookup Error",
					map[string]interface{}{},
					&EncodeContext{Registry: NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.WriteDocument,
					ErrNoEncoder{Type: reflect.TypeOf((*interface{})(nil)).Elem()},
				},
				{
					"WriteDocumentElement Error",
					map[string]interface{}{"foo": "bar"},
					&EncodeContext{Registry: buildDefaultRegistry()},
					&bsonrwtest.ValueReaderWriter{Err: errors.New("wde error"), ErrAfter: bsonrwtest.WriteDocumentElement},
					bsonrwtest.WriteDocumentElement,
					errors.New("wde error"),
				},
				{
					"EncodeValue Error",
					map[string]interface{}{"foo": "bar"},
					&EncodeContext{Registry: buildDefaultRegistry()},
					&bsonrwtest.ValueReaderWriter{Err: errors.New("ev error"), ErrAfter: bsonrwtest.WriteString},
					bsonrwtest.WriteString,
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
					bsonrwtest.Nothing,
					errors.New("SliceEncodeValue can only encode arrays and slices"),
				},
				{
					"WriteArray Error",
					[]string{},
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("wa error"), ErrAfter: bsonrwtest.WriteArray},
					bsonrwtest.WriteArray,
					errors.New("wa error"),
				},
				{
					"Lookup Error",
					[]interface{}{},
					&EncodeContext{Registry: NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.WriteArray,
					ErrNoEncoder{Type: reflect.TypeOf((*interface{})(nil)).Elem()},
				},
				{
					"WriteArrayElement Error",
					[]string{"foo"},
					&EncodeContext{Registry: buildDefaultRegistry()},
					&bsonrwtest.ValueReaderWriter{Err: errors.New("wae error"), ErrAfter: bsonrwtest.WriteArrayElement},
					bsonrwtest.WriteArrayElement,
					errors.New("wae error"),
				},
				{
					"EncodeValue Error",
					[]string{"foo"},
					&EncodeContext{Registry: buildDefaultRegistry()},
					&bsonrwtest.ValueReaderWriter{Err: errors.New("ev error"), ErrAfter: bsonrwtest.WriteString},
					bsonrwtest.WriteString,
					errors.New("ev error"),
				},
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
					bsonrwtest.Nothing,
					ValueEncoderError{
						Name:     "ObjectIDEncodeValue",
						Types:    []interface{}{objectid.ObjectID{}, (*objectid.ObjectID)(nil)},
						Received: wrong,
					},
				},
				{
					"objectid.ObjectID/success",
					objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					nil, nil, bsonrwtest.WriteObjectID, nil,
				},
				{
					"*objectid.ObjectID/success",
					&objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					nil, nil, bsonrwtest.WriteObjectID, nil,
				},
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
					bsonrwtest.Nothing,
					ValueEncoderError{
						Name:     "Decimal128EncodeValue",
						Types:    []interface{}{decimal.Decimal128{}, (*decimal.Decimal128)(nil)},
						Received: wrong,
					},
				},
				{"Decimal128/success", d128, nil, nil, bsonrwtest.WriteDecimal128, nil},
				{"*Decimal128/success", &d128, nil, nil, bsonrwtest.WriteDecimal128, nil},
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
					bsonrwtest.Nothing,
					ValueEncoderError{
						Name:     "JSONNumberEncodeValue",
						Types:    []interface{}{json.Number(""), (*json.Number)(nil)},
						Received: wrong,
					},
				},
				{
					"json.Number/invalid",
					json.Number("hello world"),
					nil, nil, bsonrwtest.Nothing, errors.New(`strconv.ParseFloat: parsing "hello world": invalid syntax`),
				},
				{
					"json.Number/int64/success",
					json.Number("1234567890"),
					nil, nil, bsonrwtest.WriteInt64, nil,
				},
				{
					"json.Number/float64/success",
					json.Number("3.14159"),
					nil, nil, bsonrwtest.WriteDouble, nil,
				},
				{
					"*json.Number/int64/success",
					pjsnum,
					nil, nil, bsonrwtest.WriteDouble, nil,
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
					bsonrwtest.Nothing,
					ValueEncoderError{
						Name:     "URLEncodeValue",
						Types:    []interface{}{url.URL{}, (*url.URL)(nil)},
						Received: wrong,
					},
				},
				{"url.URL", url.URL{Scheme: "http", Host: "example.com"}, nil, nil, bsonrwtest.WriteString, nil},
				{"*url.URL", &url.URL{Scheme: "http", Host: "example.com"}, nil, nil, bsonrwtest.WriteString, nil},
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
					bsonrwtest.Nothing,
					ValueEncoderError{
						Name:     "ByteSliceEncodeValue",
						Types:    []interface{}{[]byte{}, (*[]byte)(nil)},
						Received: wrong,
					},
				},
				{"[]byte", []byte{0x01, 0x02, 0x03}, nil, nil, bsonrwtest.WriteBinary, nil},
				{"*[]byte", &([]byte{0x01, 0x02, 0x03}), nil, nil, bsonrwtest.WriteBinary, nil},
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
					bsonrwtest.Nothing,
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
					bsonrwtest.Nothing,
					errors.New("mbsonv error"),
				},
				{
					"Copy error",
					testValueMarshaler{},
					nil,
					nil,
					bsonrwtest.Nothing,
					fmt.Errorf("Cannot copy unknown BSON type %s", bsontype.Type(0)),
				},
				{
					"success",
					testValueMarshaler{t: bsontype.String, buf: []byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00}},
					nil,
					nil,
					bsonrwtest.WriteString,
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
				buildDocument(bsoncore.AppendObjectIDElement(nil, "foo", oid)),
				nil,
			},
			{
				"map[string][]int32",
				map[string][]int32{"Z": {1, 2, 3}},
				buildDocumentArray(func(doc []byte) []byte {
					doc = bsoncore.AppendInt32Element(doc, "0", 1)
					doc = bsoncore.AppendInt32Element(doc, "1", 2)
					return bsoncore.AppendInt32Element(doc, "2", 3)
				}),
				nil,
			},
			{
				"map[string][]objectid.ObjectID",
				map[string][]objectid.ObjectID{"Z": oids},
				buildDocumentArray(func(doc []byte) []byte {
					doc = bsoncore.AppendObjectIDElement(doc, "0", oids[0])
					doc = bsoncore.AppendObjectIDElement(doc, "1", oids[1])
					return bsoncore.AppendObjectIDElement(doc, "2", oids[2])
				}),
				nil,
			},
			{
				"map[string][]json.Number(int64)",
				map[string][]json.Number{"Z": {json.Number("5"), json.Number("10")}},
				buildDocumentArray(func(doc []byte) []byte {
					doc = bsoncore.AppendInt64Element(doc, "0", 5)
					return bsoncore.AppendInt64Element(doc, "1", 10)
				}),
				nil,
			},
			{
				"map[string][]json.Number(float64)",
				map[string][]json.Number{"Z": {json.Number("5"), json.Number("10.1")}},
				buildDocumentArray(func(doc []byte) []byte {
					doc = bsoncore.AppendInt64Element(doc, "0", 5)
					return bsoncore.AppendDoubleElement(doc, "1", 10.1)
				}),
				nil,
			},
			{
				"map[string][]*url.URL",
				map[string][]*url.URL{"Z": {murl}},
				buildDocumentArray(func(doc []byte) []byte {
					return bsoncore.AppendStringElement(doc, "0", murl.String())
				}),
				nil,
			},
			{
				"map[string][]decimal.Decimal128",
				map[string][]decimal.Decimal128{"Z": {decimal128}},
				buildDocumentArray(func(doc []byte) []byte {
					return bsoncore.AppendDecimal128Element(doc, "0", decimal128)
				}),
				nil,
			},
			{
				"-",
				struct {
					A string `bson:"-"`
				}{
					A: "",
				},
				[]byte{0x05, 0x00, 0x00, 0x00, 0x00},
				nil,
			},
			{
				"omitempty",
				struct {
					A string `bson:",omitempty"`
				}{
					A: "",
				},
				[]byte{0x05, 0x00, 0x00, 0x00, 0x00},
				nil,
			},
			{
				"omitempty, empty time",
				struct {
					A time.Time `bson:",omitempty"`
				}{
					A: time.Time{},
				},
				[]byte{0x05, 0x00, 0x00, 0x00, 0x00},
				nil,
			},
			{
				"no private fields",
				noPrivateFields{a: "should be empty"},
				[]byte{0x05, 0x00, 0x00, 0x00, 0x00},
				nil,
			},
			{
				"minsize",
				struct {
					A int64 `bson:",minsize"`
				}{
					A: 12345,
				},
				buildDocument(bsoncore.AppendInt32Element(nil, "a", 12345)),
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
				buildDocument(bsoncore.AppendInt32Element(nil, "a", 12345)),
				nil,
			},
			{
				"inline map",
				struct {
					Foo map[string]string `bson:",inline"`
				}{
					Foo: map[string]string{"foo": "bar"},
				},
				buildDocument(bsoncore.AppendStringElement(nil, "foo", "bar")),
				nil,
			},
			{
				"alternate name bson:name",
				struct {
					A string `bson:"foo"`
				}{
					A: "bar",
				},
				buildDocument(bsoncore.AppendStringElement(nil, "foo", "bar")),
				nil,
			},
			{
				"alternate name",
				struct {
					A string `bson:"foo"`
				}{
					A: "bar",
				},
				buildDocument(bsoncore.AppendStringElement(nil, "foo", "bar")),
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
				buildDocument(bsoncore.AppendStringElement(nil, "a", "bar")),
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
					Q:  oid,
					T:  nil,
					Y:  json.Number("5"),
					Z:  now,
					AA: json.Number("10.1"),
					AB: murl,
					AC: decimal128,
					AD: &now,
					AE: testValueMarshaler{t: bsontype.String, buf: bsoncore.AppendString(nil, "hello, world")},
				},
				buildDocument(func(doc []byte) []byte {
					doc = bsoncore.AppendBooleanElement(doc, "a", true)
					doc = bsoncore.AppendInt32Element(doc, "b", 123)
					doc = bsoncore.AppendInt64Element(doc, "c", 456)
					doc = bsoncore.AppendInt32Element(doc, "d", 789)
					doc = bsoncore.AppendInt64Element(doc, "e", 101112)
					doc = bsoncore.AppendDoubleElement(doc, "f", 3.14159)
					doc = bsoncore.AppendStringElement(doc, "g", "Hello, world")
					doc = bsoncore.AppendDocumentElement(doc, "h", buildDocument(bsoncore.AppendStringElement(nil, "foo", "bar")))
					doc = bsoncore.AppendBinaryElement(doc, "i", 0x00, []byte{0x01, 0x02, 0x03})
					doc = bsoncore.AppendArrayElement(doc, "k",
						buildArray(bsoncore.AppendStringElement(bsoncore.AppendStringElement(nil, "0", "baz"), "1", "qux")),
					)
					doc = bsoncore.AppendDocumentElement(doc, "l", buildDocument(bsoncore.AppendStringElement(nil, "m", "foobar")))
					doc = bsoncore.AppendObjectIDElement(doc, "q", oid)
					doc = bsoncore.AppendNullElement(doc, "t")
					doc = bsoncore.AppendInt64Element(doc, "y", 5)
					doc = bsoncore.AppendDateTimeElement(doc, "z", now.UnixNano()/int64(time.Millisecond))
					doc = bsoncore.AppendDoubleElement(doc, "aa", 10.1)
					doc = bsoncore.AppendStringElement(doc, "ab", murl.String())
					doc = bsoncore.AppendDecimal128Element(doc, "ac", decimal128)
					doc = bsoncore.AppendDateTimeElement(doc, "ad", now.UnixNano()/int64(time.Millisecond))
					doc = bsoncore.AppendStringElement(doc, "ae", "hello, world")
					return doc
				}(nil)),
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
						{t: bsontype.String, buf: bsoncore.AppendString(nil, "hello")},
						{t: bsontype.String, buf: bsoncore.AppendString(nil, "world")},
					},
				},
				buildDocument(func(doc []byte) []byte {
					doc = appendArrayElement(doc, "a", bsoncore.AppendBooleanElement(nil, "0", true))
					doc = appendArrayElement(doc, "b", bsoncore.AppendInt32Element(nil, "0", 123))
					doc = appendArrayElement(doc, "c", bsoncore.AppendInt64Element(nil, "0", 456))
					doc = appendArrayElement(doc, "d", bsoncore.AppendInt32Element(nil, "0", 789))
					doc = appendArrayElement(doc, "e", bsoncore.AppendInt64Element(nil, "0", 101112))
					doc = appendArrayElement(doc, "f", bsoncore.AppendDoubleElement(nil, "0", 3.14159))
					doc = appendArrayElement(doc, "g", bsoncore.AppendStringElement(nil, "0", "Hello, world"))
					doc = appendArrayElement(doc, "h", buildDocumentElement("0", bsoncore.AppendStringElement(nil, "foo", "bar")))
					doc = appendArrayElement(doc, "i", bsoncore.AppendBinaryElement(nil, "0", 0x00, []byte{0x01, 0x02, 0x03}))
					doc = appendArrayElement(doc, "k",
						buildArrayElement("0",
							bsoncore.AppendStringElement(bsoncore.AppendStringElement(nil, "0", "baz"), "1", "qux")),
					)
					doc = appendArrayElement(doc, "l", buildDocumentElement("0", bsoncore.AppendStringElement(nil, "m", "foobar")))
					doc = appendArrayElement(doc, "n",
						buildArrayElement("0",
							bsoncore.AppendStringElement(bsoncore.AppendStringElement(nil, "0", "foo"), "1", "bar")),
					)
					doc = appendArrayElement(doc, "r",
						bsoncore.AppendObjectIDElement(
							bsoncore.AppendObjectIDElement(
								bsoncore.AppendObjectIDElement(nil,
									"0", oids[0]),
								"1", oids[1]),
							"2", oids[2]),
					)
					doc = bsoncore.AppendNullElement(doc, "t")
					doc = bsoncore.AppendNullElement(doc, "w")
					doc = appendArrayElement(doc, "x", nil)
					doc = appendArrayElement(doc, "y", buildDocumentElement("0", nil))
					doc = appendArrayElement(doc, "z",
						bsoncore.AppendDateTimeElement(
							bsoncore.AppendDateTimeElement(
								nil, "0", now.UnixNano()/int64(time.Millisecond)),
							"1", now.UnixNano()/int64(time.Millisecond)),
					)
					doc = appendArrayElement(doc, "aa", bsoncore.AppendDoubleElement(bsoncore.AppendInt64Element(nil, "0", 5), "1", 10.10))
					doc = appendArrayElement(doc, "ab", bsoncore.AppendStringElement(nil, "0", murl.String()))
					doc = appendArrayElement(doc, "ac", bsoncore.AppendDecimal128Element(nil, "0", decimal128))
					doc = appendArrayElement(doc, "ad",
						bsoncore.AppendDateTimeElement(
							bsoncore.AppendDateTimeElement(nil, "0", now.UnixNano()/int64(time.Millisecond)),
							"1", now.UnixNano()/int64(time.Millisecond)),
					)
					doc = appendArrayElement(doc, "ae",
						bsoncore.AppendStringElement(bsoncore.AppendStringElement(nil, "0", "hello"), "1", "world"),
					)
					return doc
				}(nil)),
				nil,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				b := make(bsonrw.SliceWriter, 0, 512)
				vw, err := bsonrw.NewBSONValueWriter(&b)
				noerr(t, err)
				reg := buildDefaultRegistry()
				enc, err := reg.LookupEncoder(reflect.TypeOf(tc.value))
				noerr(t, err)
				err = enc.EncodeValue(EncodeContext{Registry: reg}, vw, tc.value)
				if err != tc.err {
					t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
				}
				if diff := cmp.Diff([]byte(b), tc.b); diff != "" {
					t.Errorf("Bytes written differ: (-got +want)\n%s", diff)
					t.Errorf("Bytes\ngot: %v\nwant:%v\n", b, tc.b)
					t.Errorf("Readers\ngot: %v\nwant:%v\n", b, tc.b)
				}
			})
		}
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
