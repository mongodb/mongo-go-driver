package bsoncodec

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
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

func TestDefaultValueDecoders(t *testing.T) {
	var dvd DefaultValueDecoders
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

	const cansetreflectiontest = "cansetreflectiontest"

	intAllowedDecodeTypes := []interface{}{(*int8)(nil), (*int16)(nil), (*int32)(nil), (*int64)(nil), (*int)(nil)}
	uintAllowedDecodeTypes := []interface{}{(*uint8)(nil), (*uint16)(nil), (*uint32)(nil), (*uint64)(nil), (*uint)(nil)}
	now := time.Now().Truncate(time.Millisecond)
	d128 := decimal.NewDecimal128(12345, 67890)
	var ptrPtrValueUnmarshaler **testValueUnmarshaler

	type subtest struct {
		name   string
		val    interface{}
		dctx   *DecodeContext
		llvrw  *bsonrwtest.ValueReaderWriter
		invoke bsonrwtest.Invoked
		err    error
	}

	testCases := []struct {
		name     string
		vd       ValueDecoder
		subtests []subtest
	}{
		{
			"BooleanDecodeValue",
			ValueDecoderFunc(dvd.BooleanDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Boolean},
					bsonrwtest.Nothing,
					ValueDecoderError{Name: "BooleanDecodeValue", Types: []interface{}{bool(true)}, Received: &wrong},
				},
				{
					"type not boolean",
					bool(false),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a boolean", bsontype.String),
				},
				{
					"fast path",
					bool(true),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Boolean, Return: bool(true)},
					bsonrwtest.ReadBoolean,
					nil,
				},
				{
					"reflection path",
					mybool(true),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Boolean, Return: bool(true)},
					bsonrwtest.ReadBoolean,
					nil,
				},
				{
					"reflection path error",
					mybool(true),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Boolean, Return: bool(true), Err: errors.New("ReadBoolean Error"), ErrAfter: bsonrwtest.ReadBoolean},
					bsonrwtest.ReadBoolean, errors.New("ReadBoolean Error"),
				},
				{
					"can set false",
					cansetreflectiontest,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Boolean},
					bsonrwtest.Nothing,
					errors.New("BooleanDecodeValue can only be used to decode settable (non-nil) values"),
				},
			},
		},
		{
			"IntDecodeValue",
			ValueDecoderFunc(dvd.IntDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0)},
					bsonrwtest.ReadInt32,
					ValueDecoderError{Name: "IntDecodeValue", Types: intAllowedDecodeTypes, Received: &wrong},
				},
				{
					"type not int32/int64",
					0,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into an integer type", bsontype.String),
				},
				{
					"ReadInt32 error",
					0,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0), Err: errors.New("ReadInt32 error"), ErrAfter: bsonrwtest.ReadInt32},
					bsonrwtest.ReadInt32,
					errors.New("ReadInt32 error"),
				},
				{
					"ReadInt64 error",
					0,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(0), Err: errors.New("ReadInt64 error"), ErrAfter: bsonrwtest.ReadInt64},
					bsonrwtest.ReadInt64,
					errors.New("ReadInt64 error"),
				},
				{
					"ReadDouble error",
					0,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(0), Err: errors.New("ReadDouble error"), ErrAfter: bsonrwtest.ReadDouble},
					bsonrwtest.ReadDouble,
					errors.New("ReadDouble error"),
				},
				{
					"ReadDouble", int64(3), &DecodeContext{},
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.00)}, bsonrwtest.ReadDouble,
					nil,
				},
				{
					"ReadDouble (truncate)", int64(3), &DecodeContext{Truncate: true},
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.14)}, bsonrwtest.ReadDouble,
					nil,
				},
				{
					"ReadDouble (no truncate)", int64(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.14)}, bsonrwtest.ReadDouble,
					errors.New("IntDecodeValue can only truncate float64 to an integer type when truncation is enabled"),
				},
				{
					"ReadDouble overflows int64", int64(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: math.MaxFloat64}, bsonrwtest.ReadDouble,
					fmt.Errorf("%g overflows int64", math.MaxFloat64),
				},
				{"int8/fast path", int8(127), nil, &bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(127)}, bsonrwtest.ReadInt32, nil},
				{"int16/fast path", int16(32676), nil, &bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(32676)}, bsonrwtest.ReadInt32, nil},
				{"int32/fast path", int32(1234), nil, &bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(1234)}, bsonrwtest.ReadInt32, nil},
				{"int64/fast path", int64(1234), nil, &bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(1234)}, bsonrwtest.ReadInt64, nil},
				{"int/fast path", int(1234), nil, &bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(1234)}, bsonrwtest.ReadInt64, nil},
				{
					"int8/fast path - nil", (*int8)(nil), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0)}, bsonrwtest.ReadInt32,
					errors.New("IntDecodeValue can only be used to decode non-nil *int8"),
				},
				{
					"int16/fast path - nil", (*int16)(nil), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0)}, bsonrwtest.ReadInt32,
					errors.New("IntDecodeValue can only be used to decode non-nil *int16"),
				},
				{
					"int32/fast path - nil", (*int32)(nil), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0)}, bsonrwtest.ReadInt32,
					errors.New("IntDecodeValue can only be used to decode non-nil *int32"),
				},
				{
					"int64/fast path - nil", (*int64)(nil), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0)}, bsonrwtest.ReadInt32,
					errors.New("IntDecodeValue can only be used to decode non-nil *int64"),
				},
				{
					"int/fast path - nil", (*int)(nil), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0)}, bsonrwtest.ReadInt32,
					errors.New("IntDecodeValue can only be used to decode non-nil *int"),
				},
				{
					"int8/fast path - overflow", int8(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(129)}, bsonrwtest.ReadInt32,
					fmt.Errorf("%d overflows int8", 129),
				},
				{
					"int16/fast path - overflow", int16(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(32768)}, bsonrwtest.ReadInt32,
					fmt.Errorf("%d overflows int16", 32768),
				},
				{
					"int32/fast path - overflow", int32(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(2147483648)}, bsonrwtest.ReadInt64,
					fmt.Errorf("%d overflows int32", 2147483648),
				},
				{
					"int8/fast path - overflow (negative)", int8(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(-129)}, bsonrwtest.ReadInt32,
					fmt.Errorf("%d overflows int8", -129),
				},
				{
					"int16/fast path - overflow (negative)", int16(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(-32769)}, bsonrwtest.ReadInt32,
					fmt.Errorf("%d overflows int16", -32769),
				},
				{
					"int32/fast path - overflow (negative)", int32(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(-2147483649)}, bsonrwtest.ReadInt64,
					fmt.Errorf("%d overflows int32", -2147483649),
				},
				{
					"int8/reflection path", myint8(127), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(127)}, bsonrwtest.ReadInt32,
					nil,
				},
				{
					"int16/reflection path", myint16(255), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(255)}, bsonrwtest.ReadInt32,
					nil,
				},
				{
					"int32/reflection path", myint32(511), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(511)}, bsonrwtest.ReadInt32,
					nil,
				},
				{
					"int64/reflection path", myint64(1023), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(1023)}, bsonrwtest.ReadInt32,
					nil,
				},
				{
					"int/reflection path", myint(2047), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(2047)}, bsonrwtest.ReadInt32,
					nil,
				},
				{
					"int8/reflection path - overflow", myint8(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(129)}, bsonrwtest.ReadInt32,
					fmt.Errorf("%d overflows int8", 129),
				},
				{
					"int16/reflection path - overflow", myint16(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(32768)}, bsonrwtest.ReadInt32,
					fmt.Errorf("%d overflows int16", 32768),
				},
				{
					"int32/reflection path - overflow", myint32(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(2147483648)}, bsonrwtest.ReadInt64,
					fmt.Errorf("%d overflows int32", 2147483648),
				},
				{
					"int8/reflection path - overflow (negative)", myint8(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(-129)}, bsonrwtest.ReadInt32,
					fmt.Errorf("%d overflows int8", -129),
				},
				{
					"int16/reflection path - overflow (negative)", myint16(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(-32769)}, bsonrwtest.ReadInt32,
					fmt.Errorf("%d overflows int16", -32769),
				},
				{
					"int32/reflection path - overflow (negative)", myint32(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(-2147483649)}, bsonrwtest.ReadInt64,
					fmt.Errorf("%d overflows int32", -2147483649),
				},
				{
					"can set false",
					cansetreflectiontest,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0)},
					bsonrwtest.Nothing,
					errors.New("IntDecodeValue can only be used to decode settable (non-nil) values"),
				},
			},
		},
		{
			"UintDecodeValue",
			ValueDecoderFunc(dvd.UintDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0)},
					bsonrwtest.ReadInt32,
					ValueDecoderError{Name: "UintDecodeValue", Types: uintAllowedDecodeTypes, Received: &wrong},
				},
				{
					"type not int32/int64",
					0,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into an integer type", bsontype.String),
				},
				{
					"ReadInt32 error",
					uint(0),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0), Err: errors.New("ReadInt32 error"), ErrAfter: bsonrwtest.ReadInt32},
					bsonrwtest.ReadInt32,
					errors.New("ReadInt32 error"),
				},
				{
					"ReadInt64 error",
					uint(0),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(0), Err: errors.New("ReadInt64 error"), ErrAfter: bsonrwtest.ReadInt64},
					bsonrwtest.ReadInt64,
					errors.New("ReadInt64 error"),
				},
				{
					"ReadDouble error",
					0,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(0), Err: errors.New("ReadDouble error"), ErrAfter: bsonrwtest.ReadDouble},
					bsonrwtest.ReadDouble,
					errors.New("ReadDouble error"),
				},
				{
					"ReadDouble", uint64(3), &DecodeContext{},
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.00)}, bsonrwtest.ReadDouble,
					nil,
				},
				{
					"ReadDouble (truncate)", uint64(3), &DecodeContext{Truncate: true},
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.14)}, bsonrwtest.ReadDouble,
					nil,
				},
				{
					"ReadDouble (no truncate)", uint64(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.14)}, bsonrwtest.ReadDouble,
					errors.New("UintDecodeValue can only truncate float64 to an integer type when truncation is enabled"),
				},
				{
					"ReadDouble overflows int64", uint64(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: math.MaxFloat64}, bsonrwtest.ReadDouble,
					fmt.Errorf("%g overflows int64", math.MaxFloat64),
				},
				{"uint8/fast path", uint8(127), nil, &bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(127)}, bsonrwtest.ReadInt32, nil},
				{"uint16/fast path", uint16(255), nil, &bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(255)}, bsonrwtest.ReadInt32, nil},
				{"uint32/fast path", uint32(1234), nil, &bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(1234)}, bsonrwtest.ReadInt32, nil},
				{"uint64/fast path", uint64(1234), nil, &bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(1234)}, bsonrwtest.ReadInt64, nil},
				{"uint/fast path", uint(1234), nil, &bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(1234)}, bsonrwtest.ReadInt64, nil},
				{
					"uint8/fast path - nil", (*uint8)(nil), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0)}, bsonrwtest.ReadInt32,
					errors.New("UintDecodeValue can only be used to decode non-nil *uint8"),
				},
				{
					"uint16/fast path - nil", (*uint16)(nil), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0)}, bsonrwtest.ReadInt32,
					errors.New("UintDecodeValue can only be used to decode non-nil *uint16"),
				},
				{
					"uint32/fast path - nil", (*uint32)(nil), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0)}, bsonrwtest.ReadInt32,
					errors.New("UintDecodeValue can only be used to decode non-nil *uint32"),
				},
				{
					"uint64/fast path - nil", (*uint64)(nil), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0)}, bsonrwtest.ReadInt32,
					errors.New("UintDecodeValue can only be used to decode non-nil *uint64"),
				},
				{
					"uint/fast path - nil", (*uint)(nil), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0)}, bsonrwtest.ReadInt32,
					errors.New("UintDecodeValue can only be used to decode non-nil *uint"),
				},
				{
					"uint8/fast path - overflow", uint8(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(1 << 8)}, bsonrwtest.ReadInt32,
					fmt.Errorf("%d overflows uint8", 1<<8),
				},
				{
					"uint16/fast path - overflow", uint16(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(1 << 16)}, bsonrwtest.ReadInt32,
					fmt.Errorf("%d overflows uint16", 1<<16),
				},
				{
					"uint32/fast path - overflow", uint32(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(1 << 32)}, bsonrwtest.ReadInt64,
					fmt.Errorf("%d overflows uint32", 1<<32),
				},
				{
					"uint8/fast path - overflow (negative)", uint8(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(-1)}, bsonrwtest.ReadInt32,
					fmt.Errorf("%d overflows uint8", -1),
				},
				{
					"uint16/fast path - overflow (negative)", uint16(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(-1)}, bsonrwtest.ReadInt32,
					fmt.Errorf("%d overflows uint16", -1),
				},
				{
					"uint32/fast path - overflow (negative)", uint32(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(-1)}, bsonrwtest.ReadInt64,
					fmt.Errorf("%d overflows uint32", -1),
				},
				{
					"uint64/fast path - overflow (negative)", uint64(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(-1)}, bsonrwtest.ReadInt64,
					fmt.Errorf("%d overflows uint64", -1),
				},
				{
					"uint/fast path - overflow (negative)", uint(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(-1)}, bsonrwtest.ReadInt64,
					fmt.Errorf("%d overflows uint", -1),
				},
				{
					"uint8/reflection path", myuint8(127), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(127)}, bsonrwtest.ReadInt32,
					nil,
				},
				{
					"uint16/reflection path", myuint16(255), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(255)}, bsonrwtest.ReadInt32,
					nil,
				},
				{
					"uint32/reflection path", myuint32(511), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(511)}, bsonrwtest.ReadInt32,
					nil,
				},
				{
					"uint64/reflection path", myuint64(1023), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(1023)}, bsonrwtest.ReadInt32,
					nil,
				},
				{
					"uint/reflection path", myuint(2047), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(2047)}, bsonrwtest.ReadInt32,
					nil,
				},
				{
					"uint8/reflection path - overflow", myuint8(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(1 << 8)}, bsonrwtest.ReadInt32,
					fmt.Errorf("%d overflows uint8", 1<<8),
				},
				{
					"uint16/reflection path - overflow", myuint16(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(1 << 16)}, bsonrwtest.ReadInt32,
					fmt.Errorf("%d overflows uint16", 1<<16),
				},
				{
					"uint32/reflection path - overflow", myuint32(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(1 << 32)}, bsonrwtest.ReadInt64,
					fmt.Errorf("%d overflows uint32", 1<<32),
				},
				{
					"uint8/reflection path - overflow (negative)", myuint8(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(-1)}, bsonrwtest.ReadInt32,
					fmt.Errorf("%d overflows uint8", -1),
				},
				{
					"uint16/reflection path - overflow (negative)", myuint16(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(-1)}, bsonrwtest.ReadInt32,
					fmt.Errorf("%d overflows uint16", -1),
				},
				{
					"uint32/reflection path - overflow (negative)", myuint32(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(-1)}, bsonrwtest.ReadInt64,
					fmt.Errorf("%d overflows uint32", -1),
				},
				{
					"uint64/reflection path - overflow (negative)", myuint64(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(-1)}, bsonrwtest.ReadInt64,
					fmt.Errorf("%d overflows uint64", -1),
				},
				{
					"uint/reflection path - overflow (negative)", myuint(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(-1)}, bsonrwtest.ReadInt64,
					fmt.Errorf("%d overflows uint", -1),
				},
				{
					"can set false",
					cansetreflectiontest,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0)},
					bsonrwtest.Nothing,
					errors.New("UintDecodeValue can only be used to decode settable (non-nil) values"),
				},
			},
		},
		{
			"FloatDecodeValue",
			ValueDecoderFunc(dvd.FloatDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(0)},
					bsonrwtest.ReadDouble,
					ValueDecoderError{Name: "FloatDecodeValue", Types: []interface{}{(*float32)(nil), (*float64)(nil)}, Received: &wrong},
				},
				{
					"type not double",
					0,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a float32 or float64 type", bsontype.String),
				},
				{
					"ReadDouble error",
					float64(0),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(0), Err: errors.New("ReadDouble error"), ErrAfter: bsonrwtest.ReadDouble},
					bsonrwtest.ReadDouble,
					errors.New("ReadDouble error"),
				},
				{
					"ReadInt32 error",
					float64(0),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0), Err: errors.New("ReadInt32 error"), ErrAfter: bsonrwtest.ReadInt32},
					bsonrwtest.ReadInt32,
					errors.New("ReadInt32 error"),
				},
				{
					"ReadInt64 error",
					float64(0),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(0), Err: errors.New("ReadInt64 error"), ErrAfter: bsonrwtest.ReadInt64},
					bsonrwtest.ReadInt64,
					errors.New("ReadInt64 error"),
				},
				{
					"float64/int32", float32(32.0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(32)}, bsonrwtest.ReadInt32,
					nil,
				},
				{
					"float64/int64", float32(64.0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(64)}, bsonrwtest.ReadInt64,
					nil,
				},
				{
					"float32/fast path (equal)", float32(3.0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.0)}, bsonrwtest.ReadDouble,
					nil,
				},
				{
					"float64/fast path", float64(3.14159), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.14159)}, bsonrwtest.ReadDouble,
					nil,
				},
				{
					"float32/fast path (truncate)", float32(3.14), &DecodeContext{Truncate: true},
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.14)}, bsonrwtest.ReadDouble,
					nil,
				},
				{
					"float32/fast path (no truncate)", float32(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.14)}, bsonrwtest.ReadDouble,
					errors.New("FloatDecodeValue can only convert float64 to float32 when truncation is allowed"),
				},
				{
					"float32/fast path - nil", (*float32)(nil), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(0)}, bsonrwtest.ReadDouble,
					errors.New("FloatDecodeValue can only be used to decode non-nil *float32"),
				},
				{
					"float64/fast path - nil", (*float64)(nil), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(0)}, bsonrwtest.ReadDouble,
					errors.New("FloatDecodeValue can only be used to decode non-nil *float64"),
				},
				{
					"float32/reflection path (equal)", myfloat32(3.0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.0)}, bsonrwtest.ReadDouble,
					nil,
				},
				{
					"float64/reflection path", myfloat64(3.14159), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.14159)}, bsonrwtest.ReadDouble,
					nil,
				},
				{
					"float32/reflection path (truncate)", myfloat32(3.14), &DecodeContext{Truncate: true},
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.14)}, bsonrwtest.ReadDouble,
					nil,
				},
				{
					"float32/reflection path (no truncate)", myfloat32(0), nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.14)}, bsonrwtest.ReadDouble,
					errors.New("FloatDecodeValue can only convert float64 to float32 when truncation is allowed"),
				},
				{
					"can set false",
					cansetreflectiontest,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(0)},
					bsonrwtest.Nothing,
					errors.New("FloatDecodeValue can only be used to decode settable (non-nil) values"),
				},
			},
		},
		{
			"TimeDecodeValue",
			ValueDecoderFunc(dvd.TimeDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(0)},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a time.Time", bsontype.Int32),
				},
				{
					"type not *time.Time",
					int64(0),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.DateTime, Return: int64(1234567890)},
					bsonrwtest.ReadDateTime,
					ValueDecoderError{
						Name:     "TimeDecodeValue",
						Types:    []interface{}{(*time.Time)(nil), (**time.Time)(nil)},
						Received: (*int64)(nil),
					},
				},
				{
					"ReadDateTime error",
					time.Time{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.DateTime, Return: int64(0), Err: errors.New("ReadDateTime error"), ErrAfter: bsonrwtest.ReadDateTime},
					bsonrwtest.ReadDateTime,
					errors.New("ReadDateTime error"),
				},
				{
					"time.Time",
					now,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.DateTime, Return: int64(now.UnixNano() / int64(time.Millisecond))},
					bsonrwtest.ReadDateTime,
					nil,
				},
				{
					"*time.Time",
					&now,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.DateTime, Return: int64(now.UnixNano() / int64(time.Millisecond))},
					bsonrwtest.ReadDateTime,
					nil,
				},
			},
		},
		{
			"MapDecodeValue",
			ValueDecoderFunc(dvd.MapDecodeValue),
			[]subtest{
				{
					"wrong kind",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.Nothing,
					errors.New("MapDecodeValue can only decode settable maps with string keys"),
				},
				{
					"wrong kind (non-string key)",
					map[int]interface{}{},
					nil,
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.Nothing,
					errors.New("MapDecodeValue can only decode settable maps with string keys"),
				},
				{
					"ReadDocument Error",
					make(map[string]interface{}),
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("rd error"), ErrAfter: bsonrwtest.ReadDocument},
					bsonrwtest.ReadDocument,
					errors.New("rd error"),
				},
				{
					"Lookup Error",
					map[string]string{},
					&DecodeContext{Registry: NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.ReadDocument,
					ErrNoDecoder{Type: reflect.TypeOf(string(""))},
				},
				{
					"ReadElement Error",
					make(map[string]interface{}),
					&DecodeContext{Registry: buildDefaultRegistry()},
					&bsonrwtest.ValueReaderWriter{Err: errors.New("re error"), ErrAfter: bsonrwtest.ReadElement},
					bsonrwtest.ReadElement,
					errors.New("re error"),
				},
				{
					"DecodeValue Error",
					map[string]string{"foo": "bar"},
					&DecodeContext{Registry: buildDefaultRegistry()},
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Err: errors.New("dv error"), ErrAfter: bsonrwtest.ReadString},
					bsonrwtest.ReadString,
					errors.New("dv error"),
				},
			},
		},
		{
			"SliceDecodeValue",
			ValueDecoderFunc(dvd.SliceDecodeValue),
			[]subtest{
				{
					"wrong kind",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.Nothing,
					fmt.Errorf("SliceDecodeValue can only decode settable slice and array values, got %T", &wrong),
				},
				{
					"can set false",
					(*[]string)(nil),
					nil,
					&bsonrwtest.ValueReaderWriter{},
					bsonrwtest.Nothing,
					fmt.Errorf("SliceDecodeValue can only be used to decode non-nil pointers to slice or array values, got %T", (*[]string)(nil)),
				},
				{
					"Not Type Array",
					[]interface{}{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					errors.New("cannot decode string into a slice"),
				},
				{
					"ReadArray Error",
					[]interface{}{},
					nil,
					&bsonrwtest.ValueReaderWriter{Err: errors.New("ra error"), ErrAfter: bsonrwtest.ReadArray, BSONType: bsontype.Array},
					bsonrwtest.ReadArray,
					errors.New("ra error"),
				},
				{
					"Lookup Error",
					[]string{},
					&DecodeContext{Registry: NewRegistryBuilder().Build()},
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Array},
					bsonrwtest.ReadArray,
					ErrNoDecoder{Type: reflect.TypeOf(string(""))},
				},
				{
					"ReadValue Error",
					[]string{},
					&DecodeContext{Registry: buildDefaultRegistry()},
					&bsonrwtest.ValueReaderWriter{Err: errors.New("rv error"), ErrAfter: bsonrwtest.ReadValue, BSONType: bsontype.Array},
					bsonrwtest.ReadValue,
					errors.New("rv error"),
				},
				{
					"DecodeValue Error",
					[]string{},
					&DecodeContext{Registry: buildDefaultRegistry()},
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Array},
					bsonrwtest.ReadValue,
					errors.New("cannot decode array into a string type"),
				},
			},
		},
		{
			"ObjectIDDecodeValue",
			ValueDecoderFunc(dvd.ObjectIDDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.ObjectID},
					bsonrwtest.Nothing,
					ValueDecoderError{Name: "ObjectIDDecodeValue", Types: []interface{}{(*objectid.ObjectID)(nil)}, Received: &wrong},
				},
				{
					"type not objectID",
					objectid.ObjectID{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into an ObjectID", bsontype.String),
				},
				{
					"ReadObjectID Error",
					objectid.ObjectID{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.ObjectID, Err: errors.New("roid error"), ErrAfter: bsonrwtest.ReadObjectID},
					bsonrwtest.ReadObjectID,
					errors.New("roid error"),
				},
				{
					"success",
					objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					nil,
					&bsonrwtest.ValueReaderWriter{
						BSONType: bsontype.ObjectID,
						Return:   objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					},
					bsonrwtest.ReadObjectID,
					nil,
				},
			},
		},
		{
			"Decimal128DecodeValue",
			ValueDecoderFunc(dvd.Decimal128DecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Decimal128},
					bsonrwtest.Nothing,
					ValueDecoderError{Name: "Decimal128DecodeValue", Types: []interface{}{(*decimal.Decimal128)(nil)}, Received: &wrong},
				},
				{
					"type not decimal128",
					decimal.Decimal128{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a decimal.Decimal128", bsontype.String),
				},
				{
					"ReadDecimal128 Error",
					decimal.Decimal128{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Decimal128, Err: errors.New("rd128 error"), ErrAfter: bsonrwtest.ReadDecimal128},
					bsonrwtest.ReadDecimal128,
					errors.New("rd128 error"),
				},
				{
					"success",
					d128,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Decimal128, Return: d128},
					bsonrwtest.ReadDecimal128,
					nil,
				},
			},
		},
		{
			"JSONNumberDecodeValue",
			ValueDecoderFunc(dvd.JSONNumberDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.ObjectID},
					bsonrwtest.Nothing,
					ValueDecoderError{Name: "JSONNumberDecodeValue", Types: []interface{}{(*json.Number)(nil)}, Received: &wrong},
				},
				{
					"type not double/int32/int64",
					json.Number(""),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a json.Number", bsontype.String),
				},
				{
					"ReadDouble Error",
					json.Number(""),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Err: errors.New("rd error"), ErrAfter: bsonrwtest.ReadDouble},
					bsonrwtest.ReadDouble,
					errors.New("rd error"),
				},
				{
					"ReadInt32 Error",
					json.Number(""),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Err: errors.New("ri32 error"), ErrAfter: bsonrwtest.ReadInt32},
					bsonrwtest.ReadInt32,
					errors.New("ri32 error"),
				},
				{
					"ReadInt64 Error",
					json.Number(""),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Err: errors.New("ri64 error"), ErrAfter: bsonrwtest.ReadInt64},
					bsonrwtest.ReadInt64,
					errors.New("ri64 error"),
				},
				{
					"success/double",
					json.Number("3.14159"),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Double, Return: float64(3.14159)},
					bsonrwtest.ReadDouble,
					nil,
				},
				{
					"success/int32",
					json.Number("12345"),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32, Return: int32(12345)},
					bsonrwtest.ReadInt32,
					nil,
				},
				{
					"success/int64",
					json.Number("1234567890"),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int64, Return: int64(1234567890)},
					bsonrwtest.ReadInt64,
					nil,
				},
			},
		},
		{
			"URLDecodeValue",
			ValueDecoderFunc(dvd.URLDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a *url.URL", bsontype.Int32),
				},
				{
					"type not *url.URL",
					int64(0),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Return: string("http://example.com")},
					bsonrwtest.ReadString,
					ValueDecoderError{Name: "URLDecodeValue", Types: []interface{}{(*url.URL)(nil), (**url.URL)(nil)}, Received: (*int64)(nil)},
				},
				{
					"ReadString error",
					url.URL{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Err: errors.New("rs error"), ErrAfter: bsonrwtest.ReadString},
					bsonrwtest.ReadString,
					errors.New("rs error"),
				},
				{
					"url.Parse error",
					url.URL{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Return: string("not-valid-%%%%://")},
					bsonrwtest.ReadString,
					errors.New("parse not-valid-%%%%://: first path segment in URL cannot contain colon"),
				},
				{
					"nil *url.URL",
					(*url.URL)(nil),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Return: string("http://example.com")},
					bsonrwtest.ReadString,
					ValueDecoderError{Name: "URLDecodeValue", Types: []interface{}{(*url.URL)(nil), (**url.URL)(nil)}, Received: (*url.URL)(nil)},
				},
				{
					"nil **url.URL",
					(**url.URL)(nil),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Return: string("http://example.com")},
					bsonrwtest.ReadString,
					ValueDecoderError{Name: "URLDecodeValue", Types: []interface{}{(*url.URL)(nil), (**url.URL)(nil)}, Received: (**url.URL)(nil)},
				},
				{
					"url.URL",
					url.URL{Scheme: "http", Host: "example.com"},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Return: string("http://example.com")},
					bsonrwtest.ReadString,
					nil,
				},
				{
					"*url.URL",
					&url.URL{Scheme: "http", Host: "example.com"},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Return: string("http://example.com")},
					bsonrwtest.ReadString,
					nil,
				},
			},
		},
		{
			"ByteSliceDecodeValue",
			ValueDecoderFunc(dvd.ByteSliceDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Int32},
					bsonrwtest.Nothing,
					fmt.Errorf("cannot decode %v into a *[]byte", bsontype.Int32),
				},
				{
					"type not *[]byte",
					int64(0),
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Binary, Return: bsoncore.Value{Type: bsontype.Binary}},
					bsonrwtest.Nothing,
					ValueDecoderError{Name: "ByteSliceDecodeValue", Types: []interface{}{(*[]byte)(nil)}, Received: (*int64)(nil)},
				},
				{
					"ReadBinary error",
					[]byte{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.Binary, Err: errors.New("rb error"), ErrAfter: bsonrwtest.ReadBinary},
					bsonrwtest.ReadBinary,
					errors.New("rb error"),
				},
				{
					"incorrect subtype",
					[]byte{},
					nil,
					&bsonrwtest.ValueReaderWriter{
						BSONType: bsontype.Binary,
						Return: bsoncore.Value{
							Type: bsontype.Binary,
							Data: bsoncore.AppendBinary(nil, 0xFF, []byte{0x01, 0x02, 0x03}),
						},
					},
					bsonrwtest.ReadBinary,
					fmt.Errorf("ByteSliceDecodeValue can only be used to decode subtype 0x00 for %s, got %v", bsontype.Binary, byte(0xFF)),
				},
			},
		},
		{
			"ValueUnmarshalerDecodeValue",
			ValueDecoderFunc(dvd.ValueUnmarshalerDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					bsonrwtest.Nothing,
					fmt.Errorf("ValueUnmarshalerDecodeValue can only handle types or pointers to types that are a ValueUnmarshaler, got %T", &wrong),
				},
				{
					"copy error",
					testValueUnmarshaler{},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Err: errors.New("copy error"), ErrAfter: bsonrwtest.ReadString},
					bsonrwtest.ReadString,
					errors.New("copy error"),
				},
				{
					"ValueUnmarshaler",
					testValueUnmarshaler{t: bsontype.String, val: bsoncore.AppendString(nil, "hello, world")},
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Return: string("hello, world")},
					bsonrwtest.ReadString,
					nil,
				},
				{
					"nil pointer to ValueUnmarshaler",
					ptrPtrValueUnmarshaler,
					nil,
					&bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Return: string("hello, world")},
					bsonrwtest.Nothing,
					fmt.Errorf("ValueUnmarshalerDecodeValue can only unmarshal into non-nil ValueUnmarshaler values, got %T", ptrPtrValueUnmarshaler),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, rc := range tc.subtests {
				t.Run(rc.name, func(t *testing.T) {
					var dc DecodeContext
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
					if rc.err == nil && !cmp.Equal(got, want, cmp.Comparer(compareDecimal128)) {
						t.Errorf("Values do not match. got (%T)%v; want (%T)%v", got, got, want, want)
					}
				})
			}
		})
	}

	t.Run("ValueUnmarshalerDecodeValue/UnmarshalBSONValue error", func(t *testing.T) {
		var dc DecodeContext
		llvrw := &bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Return: string("hello, world!")}
		llvrw.T = t

		want := errors.New("ubsonv error")
		valUnmarshaler := &testValueUnmarshaler{err: want}
		got := dvd.ValueUnmarshalerDecodeValue(dc, llvrw, valUnmarshaler)
		if !compareErrors(got, want) {
			t.Errorf("Errors do not match. got %v; want %v", got, want)
		}
	})
	t.Run("ValueUnmarshalerDecodeValue/Pointer to ValueUnmarshaler", func(t *testing.T) {
		var dc DecodeContext
		llvrw := &bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Return: string("hello, world!")}
		llvrw.T = t

		var want = new(*testValueUnmarshaler)
		var got = new(*testValueUnmarshaler)
		*want = &testValueUnmarshaler{t: bsontype.String, val: bsoncore.AppendString(nil, "hello, world!")}
		*got = new(testValueUnmarshaler)

		err := dvd.ValueUnmarshalerDecodeValue(dc, llvrw, got)
		noerr(t, err)
		if !cmp.Equal(*got, *want) {
			t.Errorf("Unmarshaled values do not match. got %v; want %v", *got, *want)
		}
	})
	t.Run("ValueUnmarshalerDecodeValue/nil pointer inside non-nil pointer", func(t *testing.T) {
		var dc DecodeContext
		llvrw := &bsonrwtest.ValueReaderWriter{BSONType: bsontype.String, Return: string("hello, world!")}
		llvrw.T = t

		var got = new(*testValueUnmarshaler)
		var want = new(*testValueUnmarshaler)
		*want = &testValueUnmarshaler{t: bsontype.String, val: bsoncore.AppendString(nil, "hello, world!")}

		err := dvd.ValueUnmarshalerDecodeValue(dc, llvrw, got)
		noerr(t, err)
		if !cmp.Equal(got, want) {
			t.Errorf("Results do not match. got %v; want %v", got, want)
		}
	})
	t.Run("MapCodec/DecodeValue/non-settable", func(t *testing.T) {
		var dc DecodeContext
		llvrw := new(bsonrwtest.ValueReaderWriter)
		llvrw.T = t

		want := fmt.Errorf("MapDecodeValue can only be used to decode non-nil pointers to map values, got %T", nil)
		got := dvd.MapDecodeValue(dc, llvrw, nil)
		if !compareErrors(got, want) {
			t.Errorf("Errors do not match. got %v; want %v", got, want)
		}

		want = fmt.Errorf("MapDecodeValue can only be used to decode non-nil pointers to map values, got %T", string(""))

		val := reflect.New(reflect.TypeOf(string(""))).Elem().Interface()
		got = dvd.MapDecodeValue(dc, llvrw, val)
		if !compareErrors(got, want) {
			t.Errorf("Errors do not match. got %v; want %v", got, want)
		}
		return
	})

	t.Run("SliceCodec/DecodeValue/can't set slice", func(t *testing.T) {
		var val []string
		want := fmt.Errorf("SliceDecodeValue can only be used to decode non-nil pointers to slice or array values, got %T", val)
		got := dvd.SliceDecodeValue(DecodeContext{}, nil, val)
		if !compareErrors(got, want) {
			t.Errorf("Errors do not match. got %v; want %v", got, want)
		}
	})
	t.Run("SliceCodec/DecodeValue/too many elements", func(t *testing.T) {
		idx, doc := bsoncore.AppendDocumentStart(nil)
		aidx, doc := bsoncore.AppendArrayElementStart(doc, "foo")
		doc = bsoncore.AppendStringElement(doc, "0", "foo")
		doc = bsoncore.AppendStringElement(doc, "1", "bar")
		doc, err := bsoncore.AppendArrayEnd(doc, aidx)
		noerr(t, err)
		doc, err = bsoncore.AppendDocumentEnd(doc, idx)
		noerr(t, err)
		dvr := bsonrw.NewBSONValueReader(doc)
		noerr(t, err)
		dr, err := dvr.ReadDocument()
		noerr(t, err)
		_, vr, err := dr.ReadElement()
		noerr(t, err)
		var val [1]string
		want := fmt.Errorf("more elements returned in array than can fit inside %T", val)

		dc := DecodeContext{Registry: buildDefaultRegistry()}
		got := dvd.SliceDecodeValue(dc, vr, &val)
		if !compareErrors(got, want) {
			t.Errorf("Errors do not match. got %v; want %v", got, want)
		}
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
				func() []byte {
					idx, doc := bsoncore.AppendDocumentStart(nil)
					doc = bsoncore.AppendObjectIDElement(doc, "foo", oid)
					doc, _ = bsoncore.AppendDocumentEnd(doc, idx)
					return doc
				}(),
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
					doc = bsoncore.AppendStringElement(doc, "ae", "hello, world!")
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

		t.Run("Decode", func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					vr := bsonrw.NewBSONValueReader(tc.b)
					reg := buildDefaultRegistry()
					vtype := reflect.TypeOf(tc.value)
					dec, err := reg.LookupDecoder(vtype)
					noerr(t, err)

					gotVal := reflect.New(reflect.TypeOf(tc.value))
					err = dec.DecodeValue(DecodeContext{Registry: reg}, vr, gotVal.Interface())
					noerr(t, err)

					got := gotVal.Elem().Interface()
					want := tc.value
					if diff := cmp.Diff(
						got, want,
						cmp.Comparer(compareDecimal128),
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

// buildDocumentArray inserts vals inside of an array inside of a document.
func buildDocumentArray(fn func([]byte) []byte) []byte {
	aix, doc := bsoncore.AppendArrayElementStart(nil, "Z")
	doc = fn(doc)
	doc, _ = bsoncore.AppendArrayEnd(doc, aix)
	return buildDocument(doc)
}

func buildArray(vals []byte) []byte {
	aix, doc := bsoncore.AppendArrayStart(nil)
	doc = append(doc, vals...)
	doc, _ = bsoncore.AppendArrayEnd(doc, aix)
	return doc
}

func buildArrayElement(key string, vals []byte) []byte {
	return appendArrayElement(nil, key, vals)
}

func appendArrayElement(dst []byte, key string, vals []byte) []byte {
	aix, doc := bsoncore.AppendArrayElementStart(dst, key)
	doc = append(doc, vals...)
	doc, _ = bsoncore.AppendArrayEnd(doc, aix)
	return doc
}

// buildDocument inserts elems inside of a document.
func buildDocument(elems []byte) []byte {
	idx, doc := bsoncore.AppendDocumentStart(nil)
	doc = append(doc, elems...)
	doc, _ = bsoncore.AppendDocumentEnd(doc, idx)
	return doc
}

func buildDocumentElement(key string, elems []byte) []byte {
	idx, doc := bsoncore.AppendDocumentElementStart(nil, key)
	doc = append(doc, elems...)
	doc, _ = bsoncore.AppendDocumentEnd(doc, idx)
	return doc
}
