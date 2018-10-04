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
	"github.com/mongodb/mongo-go-driver/bson/bsoncore"
	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
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
					fmt.Errorf("Cannot copy unknown BSON type %s", bsontype.Type(0)),
				},
				{
					"success",
					testValueMarshaler{t: bsontype.String, buf: []byte{0x04, 0x00, 0x00, 0x00, 'f', 'o', 'o', 0x00}},
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
					// N  *bson.Element
					// O  *bson.Document
					// P  bson.Reader
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
					// N:  bson.EC.Null("n"),
					// O:  bson.NewDocument(bson.EC.Int64("countdown", 9876543210)),
					// P:  bson.Reader{0x05, 0x00, 0x00, 0x00, 0x00},
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
					// bson.EC.Null("n"),
					// bson.EC.SubDocumentFromElements("o", bson.EC.Int64("countdown", 9876543210)),
					// bson.EC.SubDocumentFromElements("p"),
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
					N [][]string
					// O  []*bson.Element
					// P  []*bson.Document
					// Q  []bson.Reader
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
					N: [][]string{{"foo", "bar"}},
					// O:  []*bson.Element{bson.EC.Null("N")},
					// P:  []*bson.Document{bson.NewDocument(bson.EC.Int64("countdown", 9876543210))},
					// Q:  []bson.Reader{{0x05, 0x00, 0x00, 0x00, 0x00}},
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
					// bson.EC.SubDocumentFromElements("o", bson.EC.Null("N")),
					// bson.EC.ArrayFromElements("p", bson.VC.DocumentFromElements(bson.EC.Int64("countdown", 9876543210))),
					// bson.EC.ArrayFromElements("q", bson.VC.DocumentFromElements()),
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
				b := make(bsonrw.SliceWriter, 0, 512)
				vw, err := bsonrw.NewBSONValueWriter(&b)
				noerr(t, err)
				enc, err := NewEncoder(NewRegistryBuilder().Build(), vw)
				noerr(t, err)
				err = enc.Encode(tc.value)
				if err != tc.err {
					t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
				}
				if diff := cmp.Diff([]byte(b), tc.b); diff != "" {
					t.Errorf("Bytes written differ: (-got +want)\n%s", diff)
					t.Errorf("Bytes\ngot: %v\nwant:%v\n", b, tc.b)
					t.Errorf("Readers\ngot: %v\nwant:%v\n", bson.Reader(b), bson.Reader(tc.b))
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
