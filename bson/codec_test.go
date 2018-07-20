package bson

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

func TestProvidedCodecs(t *testing.T) {
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

	intAllowedTypes := []interface{}{int8(0), int16(0), int32(0), int64(0), int(0)}
	intAllowedDecodeTypes := []interface{}{(*int8)(nil), (*int16)(nil), (*int32)(nil), (*int64)(nil), (*int)(nil)}
	uintAllowedEncodeTypes := []interface{}{uint8(0), uint16(0), uint32(0), uint64(0), uint(0)}
	uintAllowedDecodeTypes := []interface{}{(*uint8)(nil), (*uint16)(nil), (*uint32)(nil), (*uint64)(nil), (*uint)(nil)}
	now := time.Now().Truncate(time.Millisecond)
	pdatetime := new(DateTime)
	*pdatetime = DateTime(1234567890)
	pjsnum := new(json.Number)
	*pjsnum = json.Number("3.14159")
	d128 := decimal.NewDecimal128(12345, 67890)

	type enccase struct {
		name   string
		val    interface{}
		ectx   *EncodeContext
		llvrw  *llValueReaderWriter
		invoke llvrwInvoked
		err    error
	}
	type deccase struct {
		name   string
		val    interface{}
		dctx   *DecodeContext
		llvrw  *llValueReaderWriter
		invoke llvrwInvoked
		err    error
	}
	testCases := []struct {
		name        string
		codec       Codec
		encodeCases []enccase
		decodeCases []deccase
	}{
		{
			"BooleanCodec",
			&BooleanCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &BooleanCodec{}, Types: []interface{}{bool(true)}, Received: wrong},
				},
				{"fast path", bool(true), nil, nil, llvrwWriteBoolean, nil},
				{"reflection path", mybool(true), nil, nil, llvrwWriteBoolean, nil},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeBoolean},
					llvrwNothing,
					CodecDecodeError{Codec: &BooleanCodec{}, Types: []interface{}{bool(true)}, Received: &wrong},
				},
				{
					"type not boolean",
					bool(false),
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a boolean", TypeString),
				},
				{
					"fast path",
					bool(true),
					nil,
					&llValueReaderWriter{bsontype: TypeBoolean, readval: bool(true)},
					llvrwReadBoolean,
					nil,
				},
				{
					"reflection path",
					mybool(true),
					nil,
					&llValueReaderWriter{bsontype: TypeBoolean, readval: bool(true)},
					llvrwReadBoolean,
					nil,
				},
				{
					"reflection path error",
					mybool(true),
					nil,
					&llValueReaderWriter{bsontype: TypeBoolean, readval: bool(true), err: errors.New("ReadBoolean Error")},
					llvrwReadBoolean, errors.New("ReadBoolean Error"),
				},
				{
					"can set false",
					cansetreflectiontest,
					nil,
					&llValueReaderWriter{bsontype: TypeBoolean},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode settable (non-nil) values", &BooleanCodec{}),
				},
			},
		},
		{
			"IntCodec",
			&IntCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &IntCodec{}, Types: intAllowedTypes, Received: wrong},
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
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0)},
					llvrwReadInt32,
					CodecDecodeError{Codec: &IntCodec{}, Types: intAllowedDecodeTypes, Received: &wrong},
				},
				{
					"type not int32/int64",
					0,
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into an integer type", TypeString),
				},
				{
					"ReadInt32 error",
					0,
					nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0), err: errors.New("ReadInt32 error"), errAfter: llvrwReadInt32},
					llvrwReadInt32,
					errors.New("ReadInt32 error"),
				},
				{
					"ReadInt64 error",
					0,
					nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(0), err: errors.New("ReadInt64 error"), errAfter: llvrwReadInt64},
					llvrwReadInt64,
					errors.New("ReadInt64 error"),
				},
				{
					"ReadDouble error",
					0,
					nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(0), err: errors.New("ReadDouble error"), errAfter: llvrwReadDouble},
					llvrwReadDouble,
					errors.New("ReadDouble error"),
				},
				{
					"ReadDouble", int64(3), &DecodeContext{},
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(3.00)}, llvrwReadDouble,
					nil,
				},
				{
					"ReadDouble (truncate)", int64(3), &DecodeContext{Truncate: true},
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(3.14)}, llvrwReadDouble,
					nil,
				},
				{
					"ReadDouble (no truncate)", int64(0), nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(3.14)}, llvrwReadDouble,
					fmt.Errorf("%T can only truncate float64 to an integer type when truncation is enabled", &IntCodec{}),
				},
				{
					"ReadDouble overflows int64", int64(0), nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: math.MaxFloat64}, llvrwReadDouble,
					fmt.Errorf("%g overflows int64", math.MaxFloat64),
				},
				{"int8/fast path", int8(127), nil, &llValueReaderWriter{bsontype: TypeInt32, readval: int32(127)}, llvrwReadInt32, nil},
				{"int16/fast path", int16(32676), nil, &llValueReaderWriter{bsontype: TypeInt32, readval: int32(32676)}, llvrwReadInt32, nil},
				{"int32/fast path", int32(1234), nil, &llValueReaderWriter{bsontype: TypeInt32, readval: int32(1234)}, llvrwReadInt32, nil},
				{"int64/fast path", int64(1234), nil, &llValueReaderWriter{bsontype: TypeInt64, readval: int64(1234)}, llvrwReadInt64, nil},
				{"int/fast path", int(1234), nil, &llValueReaderWriter{bsontype: TypeInt64, readval: int64(1234)}, llvrwReadInt64, nil},
				{
					"int8/fast path - nil", (*int8)(nil), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0)}, llvrwReadInt32,
					fmt.Errorf("%T can only be used to decode non-nil *int8", &IntCodec{}),
				},
				{
					"int16/fast path - nil", (*int16)(nil), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0)}, llvrwReadInt32,
					fmt.Errorf("%T can only be used to decode non-nil *int16", &IntCodec{}),
				},
				{
					"int32/fast path - nil", (*int32)(nil), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0)}, llvrwReadInt32,
					fmt.Errorf("%T can only be used to decode non-nil *int32", &IntCodec{}),
				},
				{
					"int64/fast path - nil", (*int64)(nil), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0)}, llvrwReadInt32,
					fmt.Errorf("%T can only be used to decode non-nil *int64", &IntCodec{}),
				},
				{
					"int/fast path - nil", (*int)(nil), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0)}, llvrwReadInt32,
					fmt.Errorf("%T can only be used to decode non-nil *int", &IntCodec{}),
				},
				{
					"int8/fast path - overflow", int8(0), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(129)}, llvrwReadInt32,
					fmt.Errorf("%d overflows int8", 129),
				},
				{
					"int16/fast path - overflow", int16(0), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(32768)}, llvrwReadInt32,
					fmt.Errorf("%d overflows int16", 32768),
				},
				{
					"int32/fast path - overflow", int32(0), nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(2147483648)}, llvrwReadInt64,
					fmt.Errorf("%d overflows int32", 2147483648),
				},
				{
					"int8/fast path - overflow (negative)", int8(0), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(-129)}, llvrwReadInt32,
					fmt.Errorf("%d overflows int8", -129),
				},
				{
					"int16/fast path - overflow (negative)", int16(0), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(-32769)}, llvrwReadInt32,
					fmt.Errorf("%d overflows int16", -32769),
				},
				{
					"int32/fast path - overflow (negative)", int32(0), nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(-2147483649)}, llvrwReadInt64,
					fmt.Errorf("%d overflows int32", -2147483649),
				},
				{
					"int8/reflection path", myint8(127), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(127)}, llvrwReadInt32,
					nil,
				},
				{
					"int16/reflection path", myint16(255), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(255)}, llvrwReadInt32,
					nil,
				},
				{
					"int32/reflection path", myint32(511), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(511)}, llvrwReadInt32,
					nil,
				},
				{
					"int64/reflection path", myint64(1023), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(1023)}, llvrwReadInt32,
					nil,
				},
				{
					"int/reflection path", myint(2047), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(2047)}, llvrwReadInt32,
					nil,
				},
				{
					"int8/reflection path - overflow", myint8(0), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(129)}, llvrwReadInt32,
					fmt.Errorf("%d overflows int8", 129),
				},
				{
					"int16/reflection path - overflow", myint16(0), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(32768)}, llvrwReadInt32,
					fmt.Errorf("%d overflows int16", 32768),
				},
				{
					"int32/reflection path - overflow", myint32(0), nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(2147483648)}, llvrwReadInt64,
					fmt.Errorf("%d overflows int32", 2147483648),
				},
				{
					"int8/reflection path - overflow (negative)", myint8(0), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(-129)}, llvrwReadInt32,
					fmt.Errorf("%d overflows int8", -129),
				},
				{
					"int16/reflection path - overflow (negative)", myint16(0), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(-32769)}, llvrwReadInt32,
					fmt.Errorf("%d overflows int16", -32769),
				},
				{
					"int32/reflection path - overflow (negative)", myint32(0), nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(-2147483649)}, llvrwReadInt64,
					fmt.Errorf("%d overflows int32", -2147483649),
				},
				{
					"can set false",
					cansetreflectiontest,
					nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0)},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode settable (non-nil) values", &IntCodec{}),
				},
			},
		},
		{
			"UintCodec",
			&UintCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &UintCodec{}, Types: uintAllowedEncodeTypes, Received: wrong},
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
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0)},
					llvrwReadInt32,
					CodecDecodeError{Codec: &UintCodec{}, Types: uintAllowedDecodeTypes, Received: &wrong},
				},
				{
					"type not int32/int64",
					0,
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into an integer type", TypeString),
				},
				{
					"ReadInt32 error",
					uint(0),
					nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0), err: errors.New("ReadInt32 error"), errAfter: llvrwReadInt32},
					llvrwReadInt32,
					errors.New("ReadInt32 error"),
				},
				{
					"ReadInt64 error",
					uint(0),
					nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(0), err: errors.New("ReadInt64 error"), errAfter: llvrwReadInt64},
					llvrwReadInt64,
					errors.New("ReadInt64 error"),
				},
				{
					"ReadDouble error",
					0,
					nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(0), err: errors.New("ReadDouble error"), errAfter: llvrwReadDouble},
					llvrwReadDouble,
					errors.New("ReadDouble error"),
				},
				{
					"ReadDouble", uint64(3), &DecodeContext{},
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(3.00)}, llvrwReadDouble,
					nil,
				},
				{
					"ReadDouble (truncate)", uint64(3), &DecodeContext{Truncate: true},
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(3.14)}, llvrwReadDouble,
					nil,
				},
				{
					"ReadDouble (no truncate)", uint64(0), nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(3.14)}, llvrwReadDouble,
					fmt.Errorf("%T can only truncate float64 to an integer type when truncation is enabled", &UintCodec{}),
				},
				{
					"ReadDouble overflows int64", uint64(0), nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: math.MaxFloat64}, llvrwReadDouble,
					fmt.Errorf("%g overflows int64", math.MaxFloat64),
				},
				{"uint8/fast path", uint8(127), nil, &llValueReaderWriter{bsontype: TypeInt32, readval: int32(127)}, llvrwReadInt32, nil},
				{"uint16/fast path", uint16(255), nil, &llValueReaderWriter{bsontype: TypeInt32, readval: int32(255)}, llvrwReadInt32, nil},
				{"uint32/fast path", uint32(1234), nil, &llValueReaderWriter{bsontype: TypeInt32, readval: int32(1234)}, llvrwReadInt32, nil},
				{"uint64/fast path", uint64(1234), nil, &llValueReaderWriter{bsontype: TypeInt64, readval: int64(1234)}, llvrwReadInt64, nil},
				{"uint/fast path", uint(1234), nil, &llValueReaderWriter{bsontype: TypeInt64, readval: int64(1234)}, llvrwReadInt64, nil},
				{
					"uint8/fast path - nil", (*uint8)(nil), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0)}, llvrwReadInt32,
					fmt.Errorf("%T can only be used to decode non-nil *uint8", &UintCodec{}),
				},
				{
					"uint16/fast path - nil", (*uint16)(nil), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0)}, llvrwReadInt32,
					fmt.Errorf("%T can only be used to decode non-nil *uint16", &UintCodec{}),
				},
				{
					"uint32/fast path - nil", (*uint32)(nil), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0)}, llvrwReadInt32,
					fmt.Errorf("%T can only be used to decode non-nil *uint32", &UintCodec{}),
				},
				{
					"uint64/fast path - nil", (*uint64)(nil), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0)}, llvrwReadInt32,
					fmt.Errorf("%T can only be used to decode non-nil *uint64", &UintCodec{}),
				},
				{
					"uint/fast path - nil", (*uint)(nil), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0)}, llvrwReadInt32,
					fmt.Errorf("%T can only be used to decode non-nil *uint", &UintCodec{}),
				},
				{
					"uint8/fast path - overflow", uint8(0), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(1 << 8)}, llvrwReadInt32,
					fmt.Errorf("%d overflows uint8", 1<<8),
				},
				{
					"uint16/fast path - overflow", uint16(0), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(1 << 16)}, llvrwReadInt32,
					fmt.Errorf("%d overflows uint16", 1<<16),
				},
				{
					"uint32/fast path - overflow", uint32(0), nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(1 << 32)}, llvrwReadInt64,
					fmt.Errorf("%d overflows uint32", 1<<32),
				},
				{
					"uint8/fast path - overflow (negative)", uint8(0), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(-1)}, llvrwReadInt32,
					fmt.Errorf("%d overflows uint8", -1),
				},
				{
					"uint16/fast path - overflow (negative)", uint16(0), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(-1)}, llvrwReadInt32,
					fmt.Errorf("%d overflows uint16", -1),
				},
				{
					"uint32/fast path - overflow (negative)", uint32(0), nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(-1)}, llvrwReadInt64,
					fmt.Errorf("%d overflows uint32", -1),
				},
				{
					"uint64/fast path - overflow (negative)", uint64(0), nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(-1)}, llvrwReadInt64,
					fmt.Errorf("%d overflows uint64", -1),
				},
				{
					"uint/fast path - overflow (negative)", uint(0), nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(-1)}, llvrwReadInt64,
					fmt.Errorf("%d overflows uint", -1),
				},
				{
					"uint8/reflection path", myuint8(127), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(127)}, llvrwReadInt32,
					nil,
				},
				{
					"uint16/reflection path", myuint16(255), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(255)}, llvrwReadInt32,
					nil,
				},
				{
					"uint32/reflection path", myuint32(511), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(511)}, llvrwReadInt32,
					nil,
				},
				{
					"uint64/reflection path", myuint64(1023), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(1023)}, llvrwReadInt32,
					nil,
				},
				{
					"uint/reflection path", myuint(2047), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(2047)}, llvrwReadInt32,
					nil,
				},
				{
					"uint8/reflection path - overflow", myuint8(0), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(1 << 8)}, llvrwReadInt32,
					fmt.Errorf("%d overflows uint8", 1<<8),
				},
				{
					"uint16/reflection path - overflow", myuint16(0), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(1 << 16)}, llvrwReadInt32,
					fmt.Errorf("%d overflows uint16", 1<<16),
				},
				{
					"uint32/reflection path - overflow", myuint32(0), nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(1 << 32)}, llvrwReadInt64,
					fmt.Errorf("%d overflows uint32", 1<<32),
				},
				{
					"uint8/reflection path - overflow (negative)", myuint8(0), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(-1)}, llvrwReadInt32,
					fmt.Errorf("%d overflows uint8", -1),
				},
				{
					"uint16/reflection path - overflow (negative)", myuint16(0), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(-1)}, llvrwReadInt32,
					fmt.Errorf("%d overflows uint16", -1),
				},
				{
					"uint32/reflection path - overflow (negative)", myuint32(0), nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(-1)}, llvrwReadInt64,
					fmt.Errorf("%d overflows uint32", -1),
				},
				{
					"uint64/reflection path - overflow (negative)", myuint64(0), nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(-1)}, llvrwReadInt64,
					fmt.Errorf("%d overflows uint64", -1),
				},
				{
					"uint/reflection path - overflow (negative)", myuint(0), nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(-1)}, llvrwReadInt64,
					fmt.Errorf("%d overflows uint", -1),
				},
				{
					"can set false",
					cansetreflectiontest,
					nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0)},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode settable (non-nil) values", &UintCodec{}),
				},
			},
		},
		{
			"FloatCodec",
			&FloatCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &FloatCodec{}, Types: []interface{}{float32(0), float64(0)}, Received: wrong},
				},
				{"float32/fast path", float32(3.14159), nil, nil, llvrwWriteDouble, nil},
				{"float64/fast path", float64(3.14159), nil, nil, llvrwWriteDouble, nil},
				{"float32/reflection path", myfloat32(3.14159), nil, nil, llvrwWriteDouble, nil},
				{"float64/reflection path", myfloat64(3.14159), nil, nil, llvrwWriteDouble, nil},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(0)},
					llvrwReadDouble,
					CodecDecodeError{Codec: &FloatCodec{}, Types: []interface{}{(*float32)(nil), (*float64)(nil)}, Received: &wrong},
				},
				{
					"type not double",
					0,
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a float32 or float64 type", TypeString),
				},
				{
					"ReadDouble error",
					float64(0),
					nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(0), err: errors.New("ReadDouble error"), errAfter: llvrwReadDouble},
					llvrwReadDouble,
					errors.New("ReadDouble error"),
				},
				{
					"ReadInt32 error",
					float64(0),
					nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0), err: errors.New("ReadInt32 error"), errAfter: llvrwReadInt32},
					llvrwReadInt32,
					errors.New("ReadInt32 error"),
				},
				{
					"ReadInt64 error",
					float64(0),
					nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(0), err: errors.New("ReadInt64 error"), errAfter: llvrwReadInt64},
					llvrwReadInt64,
					errors.New("ReadInt64 error"),
				},
				{
					"float64/int32", float32(32.0), nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(32)}, llvrwReadInt32,
					nil,
				},
				{
					"float64/int64", float32(64.0), nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(64)}, llvrwReadInt64,
					nil,
				},
				{
					"float32/fast path (equal)", float32(3.0), nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(3.0)}, llvrwReadDouble,
					nil,
				},
				{
					"float64/fast path", float64(3.14159), nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(3.14159)}, llvrwReadDouble,
					nil,
				},
				{
					"float32/fast path (truncate)", float32(3.14), &DecodeContext{Truncate: true},
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(3.14)}, llvrwReadDouble,
					nil,
				},
				{
					"float32/fast path (no truncate)", float32(0), nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(3.14)}, llvrwReadDouble,
					fmt.Errorf("%T can only convert float64 to float32 when truncation is allowed", &FloatCodec{}),
				},
				{
					"float32/fast path - nil", (*float32)(nil), nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(0)}, llvrwReadDouble,
					fmt.Errorf("%T can only be used to decode non-nil *float32", &FloatCodec{}),
				},
				{
					"float64/fast path - nil", (*float64)(nil), nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(0)}, llvrwReadDouble,
					fmt.Errorf("%T can only be used to decode non-nil *float64", &FloatCodec{}),
				},
				{
					"float32/reflection path (equal)", myfloat32(3.0), nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(3.0)}, llvrwReadDouble,
					nil,
				},
				{
					"float64/reflection path", myfloat64(3.14159), nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(3.14159)}, llvrwReadDouble,
					nil,
				},
				{
					"float32/reflection path (truncate)", myfloat32(3.14), &DecodeContext{Truncate: true},
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(3.14)}, llvrwReadDouble,
					nil,
				},
				{
					"float32/reflection path (no truncate)", myfloat32(0), nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(3.14)}, llvrwReadDouble,
					fmt.Errorf("%T can only convert float64 to float32 when truncation is allowed", &FloatCodec{}),
				},
				{
					"can set false",
					cansetreflectiontest,
					nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(0)},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode settable (non-nil) values", &FloatCodec{}),
				},
			},
		},
		{
			"StringCodec",
			&StringCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &StringCodec{}, Types: []interface{}{string(""), JavaScriptCode(""), Symbol("")}, Received: wrong},
				},
				{"string/fast path", string("foobar"), nil, nil, llvrwWriteString, nil},
				{"JavaScript/fast path", JavaScriptCode("foobar"), nil, nil, llvrwWriteJavascript, nil},
				{"Symbol/fast path", Symbol("foobar"), nil, nil, llvrwWriteSymbol, nil},
				{"reflection path", mystring("foobarbaz"), nil, nil, llvrwWriteString, nil},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeString, readval: string("")},
					llvrwReadString,
					CodecDecodeError{Codec: &StringCodec{}, Types: []interface{}{(*string)(nil), (*JavaScriptCode)(nil), (*Symbol)(nil)}, Received: &wrong},
				},
				{
					"type not string",
					string(""),
					nil,
					&llValueReaderWriter{bsontype: TypeBoolean},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a string type", TypeBoolean),
				},
				{
					"ReadString error",
					string(""),
					nil,
					&llValueReaderWriter{bsontype: TypeString, readval: string(""), err: errors.New("ReadString error"), errAfter: llvrwReadString},
					llvrwReadString,
					errors.New("ReadString error"),
				},
				{
					"ReadJavaScript error",
					string(""),
					nil,
					&llValueReaderWriter{bsontype: TypeJavaScript, readval: string(""), err: errors.New("ReadJS error"), errAfter: llvrwReadJavascript},
					llvrwReadJavascript,
					errors.New("ReadJS error"),
				},
				{
					"ReadSymbol error",
					string(""),
					nil,
					&llValueReaderWriter{bsontype: TypeSymbol, readval: string(""), err: errors.New("ReadSymbol error"), errAfter: llvrwReadSymbol},
					llvrwReadSymbol,
					errors.New("ReadSymbol error"),
				},
				{
					"string/fast path",
					string("foobar"),
					nil,
					&llValueReaderWriter{bsontype: TypeString, readval: string("foobar")},
					llvrwReadString,
					nil,
				},
				{
					"JavaScript/fast path",
					JavaScriptCode("var hello = 'world';"),
					nil,
					&llValueReaderWriter{bsontype: TypeJavaScript, readval: string("var hello = 'world';")},
					llvrwReadJavascript,
					nil,
				},
				{
					"Symbol/fast path",
					Symbol("foobarbaz"),
					nil,
					&llValueReaderWriter{bsontype: TypeSymbol, readval: Symbol("foobarbaz")},
					llvrwReadSymbol,
					nil,
				},
				{
					"string/fast path - nil", (*string)(nil), nil,
					&llValueReaderWriter{bsontype: TypeString, readval: string("")}, llvrwReadString,
					fmt.Errorf("%T can only be used to decode non-nil *string", &StringCodec{}),
				},
				{
					"JavaScript/fast path - nil", (*JavaScriptCode)(nil), nil,
					&llValueReaderWriter{bsontype: TypeJavaScript, readval: string("")}, llvrwReadJavascript,
					fmt.Errorf("%T can only be used to decode non-nil *JavaScriptCode", &StringCodec{}),
				},
				{
					"Symbol/fast path - nil", (*Symbol)(nil), nil,
					&llValueReaderWriter{bsontype: TypeSymbol, readval: Symbol("")}, llvrwReadSymbol,
					fmt.Errorf("%T can only be used to decode non-nil *Symbol", &StringCodec{}),
				},
				{
					"reflection path",
					mystring("foobarbaz"),
					nil,
					&llValueReaderWriter{bsontype: TypeString, readval: string("foobarbaz")},
					llvrwReadString,
					nil,
				},
				{
					"reflection path error",
					mystring("foobarbazqux"),
					nil,
					&llValueReaderWriter{bsontype: TypeString, readval: string("foobarbazqux"), err: errors.New("ReadString Error"), errAfter: llvrwReadString},
					llvrwReadString, errors.New("ReadString Error"),
				},
				{
					"can set false",
					cansetreflectiontest,
					nil,
					&llValueReaderWriter{bsontype: TypeString, readval: string("")},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode settable (non-nil) values", &StringCodec{}),
				},
			},
		},
		{
			"TimeCodec",
			&TimeCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &TimeCodec{}, Types: []interface{}{time.Time{}, (*time.Time)(nil)}, Received: wrong},
				},
				{"time.Time", now, nil, nil, llvrwWriteDateTime, nil},
				{"*time.Time", &now, nil, nil, llvrwWriteDateTime, nil},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0)},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a time.Time", TypeInt32),
				},
				{
					"type not *time.Time",
					int64(0),
					nil,
					&llValueReaderWriter{bsontype: TypeDateTime, readval: int64(1234567890)},
					llvrwReadDateTime,
					fmt.Errorf("%T can only be used to decode non-nil *time.Time values, got %T", &TimeCodec{}, (*int64)(nil)),
				},
				{
					"ReadDateTime error",
					time.Time{},
					nil,
					&llValueReaderWriter{bsontype: TypeDateTime, readval: int64(0), err: errors.New("ReadDateTime error"), errAfter: llvrwReadDateTime},
					llvrwReadDateTime,
					errors.New("ReadDateTime error"),
				},
				{
					"time.Time",
					now,
					nil,
					&llValueReaderWriter{bsontype: TypeDateTime, readval: int64(now.UnixNano() / int64(time.Millisecond))},
					llvrwReadDateTime,
					nil,
				},
				{
					"*time.Time",
					&now,
					nil,
					&llValueReaderWriter{bsontype: TypeDateTime, readval: int64(now.UnixNano() / int64(time.Millisecond))},
					llvrwReadDateTime,
					nil,
				},
			},
		},
		{
			"MapCodec",
			&MapCodec{},
			[]enccase{
				{
					"wrong kind",
					wrong,
					nil,
					nil,
					llvrwNothing,
					fmt.Errorf("%T can only encode maps with string keys", &MapCodec{}),
				},
				{
					"wrong kind (non-string key)",
					map[int]interface{}{},
					nil,
					nil,
					llvrwNothing,
					fmt.Errorf("%T can only encode maps with string keys", &MapCodec{}),
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
					ErrNoCodec{Type: reflect.TypeOf((*interface{})(nil)).Elem()},
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
			[]deccase{
				{
					"wrong kind",
					wrong,
					nil,
					&llValueReaderWriter{},
					llvrwNothing,
					fmt.Errorf("%T can only decode settable maps with string keys", &MapCodec{}),
				},
				{
					"wrong kind (non-string key)",
					map[int]interface{}{},
					nil,
					&llValueReaderWriter{},
					llvrwNothing,
					fmt.Errorf("%T can only decode settable maps with string keys", &MapCodec{}),
				},
				{
					"ReadDocument Error",
					make(map[string]interface{}),
					nil,
					&llValueReaderWriter{err: errors.New("rd error"), errAfter: llvrwReadDocument},
					llvrwReadDocument,
					errors.New("rd error"),
				},
				{
					"Lookup Error",
					map[string]string{},
					&DecodeContext{Registry: NewEmptyRegistryBuilder().Build()},
					&llValueReaderWriter{},
					llvrwReadDocument,
					ErrNoCodec{Type: reflect.TypeOf(string(""))},
				},
				{
					"ReadElement Error",
					make(map[string]interface{}),
					&DecodeContext{Registry: NewRegistryBuilder().Build()},
					&llValueReaderWriter{err: errors.New("re error"), errAfter: llvrwReadElement},
					llvrwReadElement,
					errors.New("re error"),
				},
				{
					"DecodeValue Error",
					map[string]string{"foo": "bar"},
					&DecodeContext{Registry: NewRegistryBuilder().Build()},
					&llValueReaderWriter{bsontype: TypeString, err: errors.New("dv error"), errAfter: llvrwReadString},
					llvrwReadString,
					errors.New("dv error"),
				},
			},
		},
		{
			"SliceCodec",
			&SliceCodec{},
			[]enccase{
				{
					"wrong kind",
					wrong,
					nil,
					nil,
					llvrwNothing,
					fmt.Errorf("%T can only encode arrays and slices", &SliceCodec{}),
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
					ErrNoCodec{Type: reflect.TypeOf((*interface{})(nil)).Elem()},
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
			[]deccase{
				{
					"wrong kind",
					wrong,
					nil,
					&llValueReaderWriter{},
					llvrwNothing,
					fmt.Errorf("%T can only decode settable slice and array values, got %T", &SliceCodec{}, &wrong),
				},
				{
					"can set false",
					(*[]string)(nil),
					nil,
					&llValueReaderWriter{},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode non-nil pointers to slice or array values, got %T", &SliceCodec{}, (*[]string)(nil)),
				},
				{
					"Not Type Array",
					[]interface{}{},
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					errors.New("cannot decode string into a slice"),
				},
				{
					"ReadArray Error",
					[]interface{}{},
					nil,
					&llValueReaderWriter{err: errors.New("ra error"), errAfter: llvrwReadArray, bsontype: TypeArray},
					llvrwReadArray,
					errors.New("ra error"),
				},
				{
					"Lookup Error",
					[]string{},
					&DecodeContext{Registry: NewEmptyRegistryBuilder().Build()},
					&llValueReaderWriter{bsontype: TypeArray},
					llvrwReadArray,
					ErrNoCodec{Type: reflect.TypeOf(string(""))},
				},
				{
					"ReadValue Error",
					[]string{},
					&DecodeContext{Registry: NewRegistryBuilder().Build()},
					&llValueReaderWriter{err: errors.New("rv error"), errAfter: llvrwReadValue, bsontype: TypeArray},
					llvrwReadValue,
					errors.New("rv error"),
				},
				{
					"DecodeValue Error",
					[]string{},
					&DecodeContext{Registry: NewRegistryBuilder().Build()},
					&llValueReaderWriter{bsontype: TypeArray},
					llvrwReadValue,
					errors.New("cannot decode array into a string type"),
				},
			},
		},
		{
			"BinaryCodec",
			&BinaryCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &BinaryCodec{}, Types: []interface{}{Binary{}, (*Binary)(nil)}, Received: wrong},
				},
				{"Binary/success", Binary{Data: []byte{0x01, 0x02}, Subtype: 0xFF}, nil, nil, llvrwWriteBinaryWithSubtype, nil},
				{"*Binary/success", &Binary{Data: []byte{0x01, 0x02}, Subtype: 0xFF}, nil, nil, llvrwWriteBinaryWithSubtype, nil},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeBinary, readval: Binary{}},
					llvrwReadBinary,
					fmt.Errorf("%T can only be used to decode non-nil *Binary values, got %T", &BinaryCodec{}, &wrong),
				},
				{
					"type not binary",
					Binary{},
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a Binary", TypeString),
				},
				{
					"ReadBinary Error",
					Binary{},
					nil,
					&llValueReaderWriter{bsontype: TypeBinary, err: errors.New("rb error"), errAfter: llvrwReadBinary},
					llvrwReadBinary,
					errors.New("rb error"),
				},
				{
					"Binary/success",
					Binary{Data: []byte{0x01, 0x02, 0x03}, Subtype: 0xFF},
					nil,
					&llValueReaderWriter{bsontype: TypeBinary, readval: Binary{Data: []byte{0x01, 0x02, 0x03}, Subtype: 0xFF}},
					llvrwReadBinary,
					nil,
				},
				{
					"*Binary/success",
					&Binary{Data: []byte{0x01, 0x02, 0x03}, Subtype: 0xFF},
					nil,
					&llValueReaderWriter{bsontype: TypeBinary, readval: Binary{Data: []byte{0x01, 0x02, 0x03}, Subtype: 0xFF}},
					llvrwReadBinary,
					nil,
				},
			},
		},
		{
			"UndefinedCodec",
			&UndefinedCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &UndefinedCodec{}, Types: []interface{}{Undefinedv2{}, (*Undefinedv2)(nil)}, Received: wrong},
				},
				{"Undefined/success", Undefinedv2{}, nil, nil, llvrwWriteUndefined, nil},
				{"*Undefined/success", &Undefinedv2{}, nil, nil, llvrwWriteUndefined, nil},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeUndefined},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode non-nil *Undefined values, got %T", &UndefinedCodec{}, &wrong),
				},
				{
					"type not undefined",
					Undefinedv2{},
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into an Undefined", TypeString),
				},
				{
					"ReadUndefined Error",
					Undefinedv2{},
					nil,
					&llValueReaderWriter{bsontype: TypeUndefined, err: errors.New("ru error"), errAfter: llvrwReadUndefined},
					llvrwReadUndefined,
					errors.New("ru error"),
				},
				{
					"ReadUndefined/success",
					Undefinedv2{},
					nil,
					&llValueReaderWriter{bsontype: TypeUndefined},
					llvrwReadUndefined,
					nil,
				},
			},
		},
		{
			"ObjectIDCodec",
			&ObjectIDCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &ObjectIDCodec{}, Types: []interface{}{objectid.ObjectID{}, (*objectid.ObjectID)(nil)}, Received: wrong},
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
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeObjectID},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode non-nil *objectid.ObjectID values, got %T", &ObjectIDCodec{}, &wrong),
				},
				{
					"type not objectID",
					objectid.ObjectID{},
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into an ObjectID", TypeString),
				},
				{
					"ReadObjectID Error",
					objectid.ObjectID{},
					nil,
					&llValueReaderWriter{bsontype: TypeObjectID, err: errors.New("roid error"), errAfter: llvrwReadObjectID},
					llvrwReadObjectID,
					errors.New("roid error"),
				},
				{
					"success",
					objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					nil,
					&llValueReaderWriter{
						bsontype: TypeObjectID,
						readval:  objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					},
					llvrwReadObjectID,
					nil,
				},
			},
		},
		{
			"DateTimeCodec",
			&DateTimeCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &DateTimeCodec{}, Types: []interface{}{DateTime(0), (*DateTime)(nil)}, Received: wrong},
				},
				{"DateTime/success", DateTime(1234567890), nil, nil, llvrwWriteDateTime, nil},
				{"*DateTime/success", pdatetime, nil, nil, llvrwWriteDateTime, nil},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeDateTime},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode non-nil *DateTime values, got %T", &DateTimeCodec{}, &wrong),
				},
				{
					"type not datetime",
					DateTime(0),
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a DateTime", TypeString),
				},
				{
					"ReadDateTime Error",
					DateTime(0),
					nil,
					&llValueReaderWriter{bsontype: TypeDateTime, err: errors.New("rdt error"), errAfter: llvrwReadDateTime},
					llvrwReadDateTime,
					errors.New("rdt error"),
				},
				{
					"success",
					DateTime(1234567890),
					nil,
					&llValueReaderWriter{bsontype: TypeDateTime, readval: int64(1234567890)},
					llvrwReadDateTime,
					nil,
				},
			},
		},
		{
			"NullCodec",
			&NullCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &NullCodec{}, Types: []interface{}{Nullv2{}, (*Nullv2)(nil)}, Received: wrong},
				},
				{"Null/success", Nullv2{}, nil, nil, llvrwWriteNull, nil},
				{"*Null/success", &Nullv2{}, nil, nil, llvrwWriteNull, nil},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeNull},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode non-nil *Null values, got %T", &NullCodec{}, &wrong),
				},
				{
					"type not null",
					Nullv2{},
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a Null", TypeString),
				},
				{
					"ReadNull Error",
					Nullv2{},
					nil,
					&llValueReaderWriter{bsontype: TypeNull, err: errors.New("rn error"), errAfter: llvrwReadNull},
					llvrwReadNull,
					errors.New("rn error"),
				},
				{
					"success",
					Nullv2{},
					nil,
					&llValueReaderWriter{bsontype: TypeNull},
					llvrwReadNull,
					nil,
				},
			},
		},
		{
			"RegexCodec",
			&RegexCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &RegexCodec{}, Types: []interface{}{Regex{}, (*Regex)(nil)}, Received: wrong},
				},
				{"Regex/success", Regex{Pattern: "foo", Options: "bar"}, nil, nil, llvrwWriteRegex, nil},
				{"*Regex/success", &Regex{Pattern: "foo", Options: "bar"}, nil, nil, llvrwWriteRegex, nil},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeRegex},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode non-nil *Regex values, got %T", &RegexCodec{}, &wrong),
				},
				{
					"type not regex",
					Regex{},
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a Regex", TypeString),
				},
				{
					"ReadRegex Error",
					Regex{},
					nil,
					&llValueReaderWriter{bsontype: TypeRegex, err: errors.New("rr error"), errAfter: llvrwReadRegex},
					llvrwReadRegex,
					errors.New("rr error"),
				},
				{
					"success",
					Regex{Pattern: "foo", Options: "bar"},
					nil,
					&llValueReaderWriter{bsontype: TypeRegex, readval: Regex{Pattern: "foo", Options: "bar"}},
					llvrwReadRegex,
					nil,
				},
			},
		},
		{
			"DBPointerCodec",
			&DBPointerCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &DBPointerCodec{}, Types: []interface{}{DBPointer{}, (*DBPointer)(nil)}, Received: wrong},
				},
				{
					"DBPointer/success",
					DBPointer{
						DB:      "foobar",
						Pointer: objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					},
					nil, nil, llvrwWriteDBPointer, nil,
				},
				{
					"*DBPointer/success",
					&DBPointer{
						DB:      "foobar",
						Pointer: objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					},
					nil, nil, llvrwWriteDBPointer, nil,
				},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeDBPointer},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode non-nil *DBPointer values, got %T", &DBPointerCodec{}, &wrong),
				},
				{
					"type not dbpointer",
					DBPointer{},
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a DBPointer", TypeString),
				},
				{
					"ReadDBPointer Error",
					DBPointer{},
					nil,
					&llValueReaderWriter{bsontype: TypeDBPointer, err: errors.New("rdbp error"), errAfter: llvrwReadDBPointer},
					llvrwReadDBPointer,
					errors.New("rdbp error"),
				},
				{
					"success",
					DBPointer{
						DB:      "foobar",
						Pointer: objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					},
					nil,
					&llValueReaderWriter{
						bsontype: TypeDBPointer,
						readval: DBPointer{
							DB:      "foobar",
							Pointer: objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
						},
					},
					llvrwReadDBPointer,
					nil,
				},
			},
		},
		{
			"CodeWithScopeCodec",
			&CodeWithScopeCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &CodeWithScopeCodec{}, Types: []interface{}{CodeWithScope{}, (*CodeWithScope)(nil)}, Received: wrong},
				},
				{
					"WriteCodeWithScope error",
					CodeWithScope{},
					nil,
					&llValueReaderWriter{err: errors.New("wcws error"), errAfter: llvrwWriteCodeWithScope},
					llvrwWriteCodeWithScope,
					errors.New("wcws error"),
				},
				{
					"CodeWithScope/success",
					CodeWithScope{
						Code:  "var hello = 'world';",
						Scope: NewDocument(),
					},
					nil, nil, llvrwWriteDocumentEnd, nil,
				},
				{
					"*CodeWithScope/success",
					&CodeWithScope{
						Code:  "var hello = 'world';",
						Scope: NewDocument(),
					},
					nil, nil, llvrwWriteDocumentEnd, nil,
				},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeCodeWithScope},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode non-nil *CodeWithScope values, got %T", &CodeWithScopeCodec{}, &wrong),
				},
				{
					"type not codewithscope",
					CodeWithScope{},
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a CodeWithScope", TypeString),
				},
				{
					"ReadCodeWithScope Error",
					CodeWithScope{},
					nil,
					&llValueReaderWriter{bsontype: TypeCodeWithScope, err: errors.New("rcws error"), errAfter: llvrwReadCodeWithScope},
					llvrwReadCodeWithScope,
					errors.New("rcws error"),
				},
				{
					"decodeDocument Error",
					CodeWithScope{
						Code:  "var hello = 'world';",
						Scope: NewDocument(EC.Null("foo")),
					},
					nil,
					&llValueReaderWriter{bsontype: TypeCodeWithScope, err: errors.New("dd error"), errAfter: llvrwReadElement},
					llvrwReadElement,
					errors.New("dd error"),
				},
			},
		},
		{
			"TimestampCodec",
			&TimestampCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &TimestampCodec{}, Types: []interface{}{Timestamp{}, (*Timestamp)(nil)}, Received: wrong},
				},
				{"Timestamp/success", Timestamp{T: 12345, I: 67890}, nil, nil, llvrwWriteTimestamp, nil},
				{"*Timestamp/success", &Timestamp{T: 12345, I: 67890}, nil, nil, llvrwWriteTimestamp, nil},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeTimestamp},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode non-nil *Timestamp values, got %T", &TimestampCodec{}, &wrong),
				},
				{
					"type not timestamp",
					Timestamp{},
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a Timestamp", TypeString),
				},
				{
					"ReadTimestamp Error",
					Timestamp{},
					nil,
					&llValueReaderWriter{bsontype: TypeTimestamp, err: errors.New("rt error"), errAfter: llvrwReadTimestamp},
					llvrwReadTimestamp,
					errors.New("rt error"),
				},
				{
					"success",
					Timestamp{T: 12345, I: 67890},
					nil,
					&llValueReaderWriter{bsontype: TypeTimestamp, readval: Timestamp{T: 12345, I: 67890}},
					llvrwReadTimestamp,
					nil,
				},
			},
		},
		{
			"Decimal128Codec",
			&Decimal128Codec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &Decimal128Codec{}, Types: []interface{}{decimal.Decimal128{}, (*decimal.Decimal128)(nil)}, Received: wrong},
				},
				{"Decimal128/success", d128, nil, nil, llvrwWriteDecimal128, nil},
				{"*Decimal128/success", &d128, nil, nil, llvrwWriteDecimal128, nil},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeDecimal128},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode non-nil *decimal.Decimal128 values, got %T", &Decimal128Codec{}, &wrong),
				},
				{
					"type not decimal128",
					decimal.Decimal128{},
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a decimal.Decimal128", TypeString),
				},
				{
					"ReadDecimal128 Error",
					decimal.Decimal128{},
					nil,
					&llValueReaderWriter{bsontype: TypeDecimal128, err: errors.New("rd128 error"), errAfter: llvrwReadDecimal128},
					llvrwReadDecimal128,
					errors.New("rd128 error"),
				},
				{
					"success",
					d128,
					nil,
					&llValueReaderWriter{bsontype: TypeDecimal128, readval: d128},
					llvrwReadDecimal128,
					nil,
				},
			},
		},
		{
			"MinKeyCodec",
			&MinKeyCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &MinKeyCodec{}, Types: []interface{}{MinKeyv2{}, (*MinKeyv2)(nil)}, Received: wrong},
				},
				{"MinKey/success", MinKeyv2{}, nil, nil, llvrwWriteMinKey, nil},
				{"*MinKey/success", &MinKeyv2{}, nil, nil, llvrwWriteMinKey, nil},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeMinKey},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode non-nil *MinKey values, got %T", &MinKeyCodec{}, &wrong),
				},
				{
					"type not null",
					MinKeyv2{},
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a MinKey", TypeString),
				},
				{
					"ReadMinKey Error",
					MinKeyv2{},
					nil,
					&llValueReaderWriter{bsontype: TypeMinKey, err: errors.New("rn error"), errAfter: llvrwReadMinKey},
					llvrwReadMinKey,
					errors.New("rn error"),
				},
				{
					"success",
					MinKeyv2{},
					nil,
					&llValueReaderWriter{bsontype: TypeMinKey},
					llvrwReadMinKey,
					nil,
				},
			},
		},
		{
			"MaxKeyCodec",
			&MaxKeyCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &MaxKeyCodec{}, Types: []interface{}{MaxKeyv2{}, (*MaxKeyv2)(nil)}, Received: wrong},
				},
				{"MaxKey/success", MaxKeyv2{}, nil, nil, llvrwWriteMaxKey, nil},
				{"*MaxKey/success", &MaxKeyv2{}, nil, nil, llvrwWriteMaxKey, nil},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeMaxKey},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode non-nil *MaxKey values, got %T", &MaxKeyCodec{}, &wrong),
				},
				{
					"type not null",
					MaxKeyv2{},
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a MaxKey", TypeString),
				},
				{
					"ReadMaxKey Error",
					MaxKeyv2{},
					nil,
					&llValueReaderWriter{bsontype: TypeMaxKey, err: errors.New("rn error"), errAfter: llvrwReadMaxKey},
					llvrwReadMaxKey,
					errors.New("rn error"),
				},
				{
					"success",
					MaxKeyv2{},
					nil,
					&llValueReaderWriter{bsontype: TypeMaxKey},
					llvrwReadMaxKey,
					nil,
				},
			},
		},
		{
			"elementCodec",
			&elementCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &elementCodec{}, Types: []interface{}{(*Element)(nil)}, Received: wrong},
				},
				{"invalid element", (*Element)(nil), nil, nil, llvrwNothing, ErrNilElement},
				{
					"success",
					EC.Null("foo"),
					&EncodeContext{Registry: NewRegistryBuilder().Build()},
					&llValueReaderWriter{},
					llvrwWriteNull,
					nil,
				},
			},
			[]deccase{
				{
					"cannot use directly",
					(*Element)(nil), nil, nil, llvrwNothing,
					errors.New("elementCodec's DecodeValue method should not be used directly"),
				},
			},
		},
		{
			"ValueCodec",
			&ValueCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &ValueCodec{}, Types: []interface{}{(*Value)(nil)}, Received: wrong},
				},
				{"invalid value", &Value{}, nil, nil, llvrwNothing, ErrUninitializedElement},
				{
					"success",
					VC.Null(),
					&EncodeContext{Registry: NewRegistryBuilder().Build()},
					&llValueReaderWriter{},
					llvrwWriteNull,
					nil,
				},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecDecodeError{Codec: &ValueCodec{}, Types: []interface{}{(**Value)(nil)}, Received: &wrong},
				},
				{"invalid value", (**Value)(nil), nil, nil, llvrwNothing, fmt.Errorf("%T can only be used to decode non-nil **Value", &ValueCodec{})},
				{
					"success",
					VC.Double(3.14159),
					&DecodeContext{Registry: NewRegistryBuilder().Build()},
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(3.14159)},
					llvrwReadDouble,
					nil,
				},
			},
		},
		{
			"JSONNumberCodec",
			&JSONNumberCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &JSONNumberCodec{}, Types: []interface{}{json.Number(""), (*json.Number)(nil)}, Received: wrong},
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
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeObjectID},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode non-nil *json.Number values, got %T", &JSONNumberCodec{}, &wrong),
				},
				{
					"type not double/int32/int64",
					json.Number(""),
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a json.Number", TypeString),
				},
				{
					"ReadDouble Error",
					json.Number(""),
					nil,
					&llValueReaderWriter{bsontype: TypeDouble, err: errors.New("rd error"), errAfter: llvrwReadDouble},
					llvrwReadDouble,
					errors.New("rd error"),
				},
				{
					"ReadInt32 Error",
					json.Number(""),
					nil,
					&llValueReaderWriter{bsontype: TypeInt32, err: errors.New("ri32 error"), errAfter: llvrwReadInt32},
					llvrwReadInt32,
					errors.New("ri32 error"),
				},
				{
					"ReadInt64 Error",
					json.Number(""),
					nil,
					&llValueReaderWriter{bsontype: TypeInt64, err: errors.New("ri64 error"), errAfter: llvrwReadInt64},
					llvrwReadInt64,
					errors.New("ri64 error"),
				},
				{
					"success/double",
					json.Number("3.14159"),
					nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(3.14159)},
					llvrwReadDouble,
					nil,
				},
				{
					"success/int32",
					json.Number("12345"),
					nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(12345)},
					llvrwReadInt32,
					nil,
				},
				{
					"success/int64",
					json.Number("1234567890"),
					nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(1234567890)},
					llvrwReadInt64,
					nil,
				},
			},
		},
		{
			"URLCodec",
			&URLCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &URLCodec{}, Types: []interface{}{url.URL{}, (*url.URL)(nil)}, Received: wrong},
				},
				{"url.URL", url.URL{Scheme: "http", Host: "example.com"}, nil, nil, llvrwWriteString, nil},
				{"*url.URL", &url.URL{Scheme: "http", Host: "example.com"}, nil, nil, llvrwWriteString, nil},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeInt32},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a *url.URL", TypeInt32),
				},
				{
					"type not *url.URL",
					int64(0),
					nil,
					&llValueReaderWriter{bsontype: TypeString, readval: string("http://example.com")},
					llvrwReadString,
					fmt.Errorf("%T can only be used to decode non-nil *url.URL values, got %T", &URLCodec{}, (*int64)(nil)),
				},
				{
					"ReadString error",
					url.URL{},
					nil,
					&llValueReaderWriter{bsontype: TypeString, err: errors.New("rs error"), errAfter: llvrwReadString},
					llvrwReadString,
					errors.New("rs error"),
				},
				{
					"url.Parse error",
					url.URL{},
					nil,
					&llValueReaderWriter{bsontype: TypeString, readval: string("not-valid-%%%%://")},
					llvrwReadString,
					errors.New("parse not-valid-%%%%://: first path segment in URL cannot contain colon"),
				},
				{
					"nil *url.URL",
					(*url.URL)(nil),
					nil,
					&llValueReaderWriter{bsontype: TypeString, readval: string("http://example.com")},
					llvrwReadString,
					fmt.Errorf("%T can only be used to decode non-nil *url.URL values, got %T", &URLCodec{}, (*url.URL)(nil)),
				},
				{
					"nil **url.URL",
					(**url.URL)(nil),
					nil,
					&llValueReaderWriter{bsontype: TypeString, readval: string("http://example.com")},
					llvrwReadString,
					fmt.Errorf("%T can only be used to decode non-nil *url.URL values, got %T", &URLCodec{}, (**url.URL)(nil)),
				},
				{
					"url.URL",
					url.URL{Scheme: "http", Host: "example.com"},
					nil,
					&llValueReaderWriter{bsontype: TypeString, readval: string("http://example.com")},
					llvrwReadString,
					nil,
				},
				{
					"*url.URL",
					&url.URL{Scheme: "http", Host: "example.com"},
					nil,
					&llValueReaderWriter{bsontype: TypeString, readval: string("http://example.com")},
					llvrwReadString,
					nil,
				},
			},
		},
		{
			"ReaderCodec",
			&ReaderCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &ReaderCodec{}, Types: []interface{}{Reader{}}, Received: wrong},
				},
				{
					"WriteDocument Error",
					Reader{},
					nil,
					&llValueReaderWriter{err: errors.New("wd error"), errAfter: llvrwWriteDocument},
					llvrwWriteDocument,
					errors.New("wd error"),
				},
				{
					"Reader.Iterator Error",
					Reader{0xFF, 0x00, 0x00, 0x00, 0x00},
					nil,
					&llValueReaderWriter{},
					llvrwWriteDocument,
					ErrInvalidLength,
				},
				{
					"WriteDocumentElement Error",
					Reader(bytesFromDoc(NewDocument(EC.Null("foo")))),
					nil,
					&llValueReaderWriter{err: errors.New("wde error"), errAfter: llvrwWriteDocumentElement},
					llvrwWriteDocumentElement,
					errors.New("wde error"),
				},
				{
					"encodeValue error",
					Reader(bytesFromDoc(NewDocument(EC.Null("foo")))),
					nil,
					&llValueReaderWriter{err: errors.New("ev error"), errAfter: llvrwWriteNull},
					llvrwWriteNull,
					errors.New("ev error"),
				},
				{
					"iterator error",
					Reader{0x0C, 0x00, 0x00, 0x00, 0x01, 'f', 'o', 'o', 0x00, 0x01, 0x02, 0x03},
					nil,
					&llValueReaderWriter{},
					llvrwWriteDocument,
					NewErrTooSmall(),
				},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{},
					llvrwNothing,
					CodecDecodeError{Codec: &ReaderCodec{}, Types: []interface{}{(*Reader)(nil)}, Received: &wrong},
				},
				{
					"*Reader is nil",
					(*Reader)(nil),
					nil,
					nil,
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode non-nil *Reader", &ReaderCodec{}),
				},
				{
					"Copy error",
					Reader{},
					nil,
					&llValueReaderWriter{err: errors.New("copy error"), errAfter: llvrwReadDocument},
					llvrwReadDocument,
					errors.New("copy error"),
				},
			},
		},
		{
			"ByteSliceCodec",
			&ByteSliceCodec{},
			[]enccase{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					CodecEncodeError{Codec: &ByteSliceCodec{}, Types: []interface{}{[]byte{}, (*[]byte)(nil)}, Received: wrong},
				},
				{"[]byte", []byte{0x01, 0x02, 0x03}, nil, nil, llvrwWriteBinary, nil},
				{"*[]byte", &([]byte{0x01, 0x02, 0x03}), nil, nil, llvrwWriteBinary, nil},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeInt32},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a *[]byte", TypeInt32),
				},
				{
					"type not *[]byte",
					int64(0),
					nil,
					&llValueReaderWriter{bsontype: TypeBinary, readval: Binary{}},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode non-nil *[]byte values, got %T", &ByteSliceCodec{}, (*int64)(nil)),
				},
				{
					"ReadBinary error",
					[]byte{},
					nil,
					&llValueReaderWriter{bsontype: TypeBinary, err: errors.New("rb error"), errAfter: llvrwReadBinary},
					llvrwReadBinary,
					errors.New("rb error"),
				},
				{
					"incorrect subtype",
					[]byte{},
					nil,
					&llValueReaderWriter{bsontype: TypeBinary, readval: Binary{Data: []byte{0x01, 0x02, 0x03}, Subtype: 0xFF}},
					llvrwReadBinary,
					fmt.Errorf("%T can only be used to decode subtype 0x00 for %s, got %v", &ByteSliceCodec{}, TypeBinary, byte(0xFF)),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("EncodeValue", func(t *testing.T) {
				for _, rc := range tc.encodeCases {
					t.Run(rc.name, func(t *testing.T) {
						var ec EncodeContext
						if rc.ectx != nil {
							ec = *rc.ectx
						}
						llvrw := new(llValueReaderWriter)
						if rc.llvrw != nil {
							llvrw = rc.llvrw
						}
						llvrw.t = t
						err := tc.codec.EncodeValue(ec, llvrw, rc.val)
						if !compareErrors(err, rc.err) {
							t.Errorf("Errors do not match. got %v; want %v", err, rc.err)
						}
						invoked := llvrw.invoked
						if !cmp.Equal(invoked, rc.invoke) {
							t.Errorf("Incorrect method invoked. got %v; want %v", invoked, rc.invoke)
						}
					})
				}
			})
			t.Run("DecodeValue", func(t *testing.T) {
				for _, rc := range tc.decodeCases {
					t.Run(rc.name, func(t *testing.T) {
						var dc DecodeContext
						if rc.dctx != nil {
							dc = *rc.dctx
						}
						llvrw := new(llValueReaderWriter)
						if rc.llvrw != nil {
							llvrw = rc.llvrw
						}
						llvrw.t = t
						var got interface{}
						if rc.val == cansetreflectiontest { // We're doing a CanSet reflection test
							err := tc.codec.DecodeValue(dc, llvrw, nil)
							if !compareErrors(err, rc.err) {
								t.Errorf("Errors do not match. got %v; want %v", err, rc.err)
							}

							val := reflect.New(reflect.TypeOf(rc.val)).Elem().Interface()
							err = tc.codec.DecodeValue(dc, llvrw, val)
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
						err := tc.codec.DecodeValue(dc, llvrw, got)
						if !compareErrors(err, rc.err) {
							t.Errorf("Errors do not match. got %v; want %v", err, rc.err)
						}
						invoked := llvrw.invoked
						if !cmp.Equal(invoked, rc.invoke) {
							t.Errorf("Incorrect method invoked. got %v; want %v", invoked, rc.invoke)
						}
						if unwrap {
							got = reflect.ValueOf(got).Elem().Interface()
						}
						if rc.err == nil && !cmp.Equal(got, want, cmp.Comparer(compareDecimal128), cmp.Comparer(compareValues)) {
							t.Errorf("Values do not match. got (%T)%v; want (%T)%v", got, got, want, want)
						}
					})
				}
			})
		})
	}

	t.Run("MapCodec/DecodeValue/non-settable", func(t *testing.T) {
		var dc DecodeContext
		llvrw := new(llValueReaderWriter)
		llvrw.t = t

		want := fmt.Errorf("%T can only be used to decode non-nil pointers to map values, got %T", &MapCodec{}, nil)
		got := (&MapCodec{}).DecodeValue(dc, llvrw, nil)
		if !compareErrors(got, want) {
			t.Errorf("Errors do not match. got %v; want %v", got, want)
		}

		want = fmt.Errorf("%T can only be used to decode non-nil pointers to map values, got %T", &MapCodec{}, string(""))

		val := reflect.New(reflect.TypeOf(string(""))).Elem().Interface()
		got = (&MapCodec{}).DecodeValue(dc, llvrw, val)
		if !compareErrors(got, want) {
			t.Errorf("Errors do not match. got %v; want %v", got, want)
		}
		return
	})

	t.Run("CodeWithScopeCodec/DecodeValue/success", func(t *testing.T) {
		dc := DecodeContext{Registry: NewRegistryBuilder().Build()}
		dvr := newDocumentValueReader(NewDocument(EC.CodeWithScope("foo", "var hello = 'world';", NewDocument(EC.Null("bar")))))
		dr, err := dvr.ReadDocument()
		noerr(t, err)
		_, vr, err := dr.ReadElement()
		noerr(t, err)

		want := CodeWithScope{
			Code:  "var hello = 'world';",
			Scope: NewDocument(EC.Null("bar")),
		}
		var got CodeWithScope
		err = (&CodeWithScopeCodec{}).DecodeValue(dc, vr, &got)
		noerr(t, err)

		if !cmp.Equal(got, want) {
			t.Errorf("CodeWithScopes do not match. got %v; want %v", got, want)
		}
	})

	t.Run("DocumentCodec", func(t *testing.T) {
		t.Run("EncodeValue", func(t *testing.T) {
			t.Run("CodecEncodeError", func(t *testing.T) {
				val := bool(true)
				want := CodecEncodeError{Codec: &DocumentCodec{}, Types: []interface{}{(*Document)(nil)}, Received: val}
				got := (&DocumentCodec{}).EncodeValue(EncodeContext{}, nil, val)
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
				got := (&DocumentCodec{}).EncodeValue(EncodeContext{}, llvrw, NewDocument())
				if !compareErrors(got, want) {
					t.Errorf("Errors do not match. got %v; want %v", got, want)
				}
			})
			t.Run("encodeDocument errors", func(t *testing.T) {
				ec := EncodeContext{}
				err := errors.New("encodeDocument error")
				oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
				badelem := EC.Null("foo")
				badelem.value.data[0] = 0x00
				testCases := []struct {
					name  string
					ec    EncodeContext
					llvrw *llValueReaderWriter
					doc   *Document
					err   error
				}{
					{
						"WriteDocumentElement",
						ec,
						&llValueReaderWriter{t: t, err: errors.New("wde error"), errAfter: llvrwWriteDocumentElement},
						NewDocument(EC.Null("foo")),
						errors.New("wde error"),
					},
					{
						"WriteDouble", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteDouble},
						NewDocument(EC.Double("foo", 3.14159)), err,
					},
					{
						"WriteString", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteString},
						NewDocument(EC.String("foo", "bar")), err,
					},
					{
						"WriteDocument (Lookup)", EncodeContext{Registry: NewEmptyRegistryBuilder().Build()},
						&llValueReaderWriter{t: t},
						NewDocument(EC.SubDocument("foo", NewDocument(EC.Null("bar")))),
						ErrNoCodec{Type: tDocument},
					},
					{
						"WriteArray (Lookup)", EncodeContext{Registry: NewEmptyRegistryBuilder().Build()},
						&llValueReaderWriter{t: t},
						NewDocument(EC.Array("foo", NewArray(VC.Null()))),
						ErrNoCodec{Type: tArray},
					},
					{
						"WriteBinary", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteBinaryWithSubtype},
						NewDocument(EC.BinaryWithSubtype("foo", []byte{0x01, 0x02, 0x03}, 0xFF)), err,
					},
					{
						"WriteUndefined", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteUndefined},
						NewDocument(EC.Undefined("foo")), err,
					},
					{
						"WriteObjectID", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteObjectID},
						NewDocument(EC.ObjectID("foo", oid)), err,
					},
					{
						"WriteBoolean", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteBoolean},
						NewDocument(EC.Boolean("foo", true)), err,
					},
					{
						"WriteDateTime", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteDateTime},
						NewDocument(EC.DateTime("foo", 1234567890)), err,
					},
					{
						"WriteNull", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteNull},
						NewDocument(EC.Null("foo")), err,
					},
					{
						"WriteRegex", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteRegex},
						NewDocument(EC.Regex("foo", "bar", "baz")), err,
					},
					{
						"WriteDBPointer", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteDBPointer},
						NewDocument(EC.DBPointer("foo", "bar", oid)), err,
					},
					{
						"WriteJavascript", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteJavascript},
						NewDocument(EC.JavaScript("foo", "var hello = 'world';")), err,
					},
					{
						"WriteSymbol", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteSymbol},
						NewDocument(EC.Symbol("foo", "symbolbaz")), err,
					},
					{
						"WriteCodeWithScope (Lookup)", EncodeContext{Registry: NewEmptyRegistryBuilder().Build()},
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteCodeWithScope},
						NewDocument(EC.CodeWithScope("foo", "var hello = 'world';", NewDocument(EC.Null("bar")))),
						err,
					},
					{
						"WriteInt32", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteInt32},
						NewDocument(EC.Int32("foo", 12345)), err,
					},
					{
						"WriteInt64", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteInt64},
						NewDocument(EC.Int64("foo", 1234567890)), err,
					},
					{
						"WriteTimestamp", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteTimestamp},
						NewDocument(EC.Timestamp("foo", 10, 20)), err,
					},
					{
						"WriteDecimal128", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteDecimal128},
						NewDocument(EC.Decimal128("foo", decimal.NewDecimal128(10, 20))), err,
					},
					{
						"WriteMinKey", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteMinKey},
						NewDocument(EC.MinKey("foo")), err,
					},
					{
						"WriteMaxKey", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteMaxKey},
						NewDocument(EC.MaxKey("foo")), err,
					},
					{
						"Invalid Type", ec,
						&llValueReaderWriter{t: t, bsontype: Type(0)},
						NewDocument(badelem),
						ErrInvalidElement,
					},
				}

				for _, tc := range testCases {
					t.Run(tc.name, func(t *testing.T) {
						err := (&DocumentCodec{}).EncodeValue(tc.ec, tc.llvrw, tc.doc)
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
				got := NewDocument()
				ec := EncodeContext{Registry: NewRegistryBuilder().Build()}
				err := (&DocumentCodec{}).EncodeValue(ec, newDocumentValueWriter(got), want)
				noerr(t, err)
				if !got.Equal(want) {
					t.Error("Documents do not match")
					t.Errorf("\ngot :%v\nwant:%v", got, want)
				}
			})
		})

		t.Run("DecodeValue", func(t *testing.T) {
			t.Run("CodecDecodeError", func(t *testing.T) {
				val := bool(true)
				want := CodecDecodeError{Codec: &DocumentCodec{}, Types: []interface{}{(**Document)(nil)}, Received: val}
				got := (&DocumentCodec{}).DecodeValue(DecodeContext{}, &llValueReaderWriter{bsontype: TypeEmbeddedDocument}, val)
				if !compareErrors(got, want) {
					t.Errorf("Errors do not match. got %v; want %v", got, want)
				}
			})
			t.Run("ReadDocument Error", func(t *testing.T) {
				want := errors.New("ReadDocument Error")
				llvrw := &llValueReaderWriter{
					t:        t,
					err:      want,
					errAfter: llvrwReadDocument,
					bsontype: TypeEmbeddedDocument,
				}
				got := (&DocumentCodec{}).DecodeValue(DecodeContext{}, llvrw, new(*Document))
				if !compareErrors(got, want) {
					t.Errorf("Errors do not match. got %v; want %v", got, want)
				}
			})
			t.Run("decodeDocument errors", func(t *testing.T) {
				dc := DecodeContext{}
				err := errors.New("decodeDocument error")
				badelem := EC.Null("foo")
				badelem.value.data[0] = 0x00
				testCases := []struct {
					name  string
					dc    DecodeContext
					llvrw *llValueReaderWriter
					err   error
				}{
					{
						"ReadElement",
						dc,
						&llValueReaderWriter{t: t, err: errors.New("re error"), errAfter: llvrwReadElement},
						errors.New("re error"),
					},
					{"ReadDouble", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadDouble, bsontype: TypeDouble}, err},
					{"ReadString", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadString, bsontype: TypeString}, err},
					{
						"ReadDocument (Lookup)", DecodeContext{Registry: NewEmptyRegistryBuilder().Build()},
						&llValueReaderWriter{t: t, bsontype: TypeEmbeddedDocument},
						ErrNoCodec{Type: tDocument},
					},
					{
						"ReadArray (Lookup)", DecodeContext{Registry: NewEmptyRegistryBuilder().Build()},
						&llValueReaderWriter{t: t, bsontype: TypeArray},
						ErrNoCodec{Type: tArray},
					},
					{"ReadBinary", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadBinary, bsontype: TypeBinary}, err},
					{"ReadUndefined", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadUndefined, bsontype: TypeUndefined}, err},
					{"ReadObjectID", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadObjectID, bsontype: TypeObjectID}, err},
					{"ReadBoolean", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadBoolean, bsontype: TypeBoolean}, err},
					{"ReadDateTime", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadDateTime, bsontype: TypeDateTime}, err},
					{"ReadNull", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadNull, bsontype: TypeNull}, err},
					{"ReadRegex", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadRegex, bsontype: TypeRegex}, err},
					{"ReadDBPointer", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadDBPointer, bsontype: TypeDBPointer}, err},
					{"ReadJavascript", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadJavascript, bsontype: TypeJavaScript}, err},
					{"ReadSymbol", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadSymbol, bsontype: TypeSymbol}, err},
					{
						"ReadCodeWithScope (Lookup)", DecodeContext{Registry: NewEmptyRegistryBuilder().Build()},
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwReadCodeWithScope, bsontype: TypeCodeWithScope},
						err,
					},
					{"ReadInt32", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadInt32, bsontype: TypeInt32}, err},
					{"ReadInt64", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadInt64, bsontype: TypeInt64}, err},
					{"ReadTimestamp", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadTimestamp, bsontype: TypeTimestamp}, err},
					{"ReadDecimal128", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadDecimal128, bsontype: TypeDecimal128}, err},
					{"ReadMinKey", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadMinKey, bsontype: TypeMinKey}, err},
					{"ReadMaxKey", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadMaxKey, bsontype: TypeMaxKey}, err},
					{"Invalid Type", dc, &llValueReaderWriter{t: t, bsontype: Type(0)}, fmt.Errorf("Cannot read unknown BSON type %s", Type(0))},
				}

				for _, tc := range testCases {
					t.Run(tc.name, func(t *testing.T) {
						err := (&DocumentCodec{}).decodeDocument(tc.dc, tc.llvrw, new(*Document))
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
				dc := DecodeContext{Registry: NewRegistryBuilder().Build()}
				err := (&DocumentCodec{}).DecodeValue(dc, newDocumentValueReader(want), &got)
				noerr(t, err)
				if !got.Equal(want) {
					t.Error("Documents do not match")
					t.Errorf("\ngot :%v\nwant:%v", got, want)
				}
			})
		})
	})

	t.Run("ArrayCodec", func(t *testing.T) {
		t.Run("EncodeValue", func(t *testing.T) {
			t.Run("CodecEncodeError", func(t *testing.T) {
				val := bool(true)
				want := CodecEncodeError{Codec: &ArrayCodec{}, Types: []interface{}{(*Array)(nil)}, Received: val}
				got := (&ArrayCodec{}).EncodeValue(EncodeContext{}, nil, val)
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
				got := (&ArrayCodec{}).EncodeValue(EncodeContext{}, llvrw, NewArray())
				if !compareErrors(got, want) {
					t.Errorf("Errors do not match. got %v; want %v", got, want)
				}
			})
			t.Run("encode array errors", func(t *testing.T) {
				ec := EncodeContext{}
				err := errors.New("encode array error")
				oid := objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C}
				badval := VC.Null()
				badval.data[0] = 0x00
				testCases := []struct {
					name  string
					ec    EncodeContext
					llvrw *llValueReaderWriter
					arr   *Array
					err   error
				}{
					{
						"WriteDocumentElement",
						ec,
						&llValueReaderWriter{t: t, err: errors.New("wde error"), errAfter: llvrwWriteArrayElement},
						NewArray(VC.Null()),
						errors.New("wde error"),
					},
					{
						"WriteDouble", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteDouble},
						NewArray(VC.Double(3.14159)), err,
					},
					{
						"WriteString", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteString},
						NewArray(VC.String("bar")), err,
					},
					{
						"WriteDocument (Lookup)", EncodeContext{Registry: NewEmptyRegistryBuilder().Build()},
						&llValueReaderWriter{t: t},
						NewArray(VC.Document(NewDocument(EC.Null("bar")))),
						ErrNoCodec{Type: tDocument},
					},
					{
						"WriteArray (Lookup)", EncodeContext{Registry: NewEmptyRegistryBuilder().Build()},
						&llValueReaderWriter{t: t},
						NewArray(VC.Array(NewArray(VC.Null()))),
						ErrNoCodec{Type: tArray},
					},
					{
						"WriteBinary", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteBinaryWithSubtype},
						NewArray(VC.BinaryWithSubtype([]byte{0x01, 0x02, 0x03}, 0xFF)), err,
					},
					{
						"WriteUndefined", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteUndefined},
						NewArray(VC.Undefined()), err,
					},
					{
						"WriteObjectID", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteObjectID},
						NewArray(VC.ObjectID(oid)), err,
					},
					{
						"WriteBoolean", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteBoolean},
						NewArray(VC.Boolean(true)), err,
					},
					{
						"WriteDateTime", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteDateTime},
						NewArray(VC.DateTime(1234567890)), err,
					},
					{
						"WriteNull", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteNull},
						NewArray(VC.Null()), err,
					},
					{
						"WriteRegex", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteRegex},
						NewArray(VC.Regex("bar", "baz")), err,
					},
					{
						"WriteDBPointer", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteDBPointer},
						NewArray(VC.DBPointer("bar", oid)), err,
					},
					{
						"WriteJavascript", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteJavascript},
						NewArray(VC.JavaScript("var hello = 'world';")), err,
					},
					{
						"WriteSymbol", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteSymbol},
						NewArray(VC.Symbol("symbolbaz")), err,
					},
					{
						"WriteCodeWithScope (Lookup)", EncodeContext{Registry: NewEmptyRegistryBuilder().Build()},
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteCodeWithScope},
						NewArray(VC.CodeWithScope("var hello = 'world';", NewDocument(EC.Null("bar")))),
						err,
					},
					{
						"WriteInt32", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteInt32},
						NewArray(VC.Int32(12345)), err,
					},
					{
						"WriteInt64", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteInt64},
						NewArray(VC.Int64(1234567890)), err,
					},
					{
						"WriteTimestamp", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteTimestamp},
						NewArray(VC.Timestamp(10, 20)), err,
					},
					{
						"WriteDecimal128", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteDecimal128},
						NewArray(VC.Decimal128(decimal.NewDecimal128(10, 20))), err,
					},
					{
						"WriteMinKey", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteMinKey},
						NewArray(VC.MinKey()), err,
					},
					{
						"WriteMaxKey", ec,
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwWriteMaxKey},
						NewArray(VC.MaxKey()), err,
					},
					{
						"Invalid Type", ec,
						&llValueReaderWriter{t: t, bsontype: Type(0)},
						NewArray(badval),
						ErrInvalidElement,
					},
				}

				for _, tc := range testCases {
					t.Run(tc.name, func(t *testing.T) {
						err := (&ArrayCodec{}).EncodeValue(tc.ec, tc.llvrw, tc.arr)
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

				ec := EncodeContext{Registry: NewRegistryBuilder().Build()}

				doc := NewDocument()
				dvw := newDocumentValueWriter(doc)
				dr, err := dvw.WriteDocument()
				noerr(t, err)
				vr, err := dr.WriteDocumentElement("foo")
				noerr(t, err)

				err = (&ArrayCodec{}).EncodeValue(ec, vr, want)
				noerr(t, err)

				got := doc.Lookup("foo").MutableArray()
				if !got.Equal(want) {
					t.Error("Documents do not match")
					t.Errorf("\ngot :%v\nwant:%v", got, want)
				}
			})
		})

		t.Run("DecodeValue", func(t *testing.T) {
			t.Run("CodecDecodeError", func(t *testing.T) {
				val := bool(true)
				want := CodecDecodeError{Codec: &ArrayCodec{}, Types: []interface{}{(**Array)(nil)}, Received: val}
				got := (&ArrayCodec{}).DecodeValue(DecodeContext{}, &llValueReaderWriter{bsontype: TypeArray}, val)
				if !compareErrors(got, want) {
					t.Errorf("Errors do not match. got %v; want %v", got, want)
				}
			})
			t.Run("ReadArray Error", func(t *testing.T) {
				want := errors.New("ReadArray Error")
				llvrw := &llValueReaderWriter{
					t:        t,
					err:      want,
					errAfter: llvrwReadArray,
					bsontype: TypeArray,
				}
				got := (&ArrayCodec{}).DecodeValue(DecodeContext{}, llvrw, new(*Array))
				if !compareErrors(got, want) {
					t.Errorf("Errors do not match. got %v; want %v", got, want)
				}
			})
			t.Run("decode array errors", func(t *testing.T) {
				dc := DecodeContext{}
				err := errors.New("decode array error")
				badval := VC.Null()
				badval.data[0] = 0x00
				testCases := []struct {
					name  string
					dc    DecodeContext
					llvrw *llValueReaderWriter
					err   error
				}{
					{
						"ReadValue",
						dc,
						&llValueReaderWriter{t: t, err: errors.New("re error"), errAfter: llvrwReadValue},
						errors.New("re error"),
					},
					{"ReadDouble", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadDouble, bsontype: TypeDouble}, err},
					{"ReadString", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadString, bsontype: TypeString}, err},
					{
						"ReadDocument (Lookup)", DecodeContext{Registry: NewEmptyRegistryBuilder().Build()},
						&llValueReaderWriter{t: t, bsontype: TypeEmbeddedDocument},
						ErrNoCodec{Type: tDocument},
					},
					{
						"ReadArray (Lookup)", DecodeContext{Registry: NewEmptyRegistryBuilder().Build()},
						&llValueReaderWriter{t: t, bsontype: TypeArray},
						ErrNoCodec{Type: tArray},
					},
					{"ReadBinary", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadBinary, bsontype: TypeBinary}, err},
					{"ReadUndefined", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadUndefined, bsontype: TypeUndefined}, err},
					{"ReadObjectID", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadObjectID, bsontype: TypeObjectID}, err},
					{"ReadBoolean", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadBoolean, bsontype: TypeBoolean}, err},
					{"ReadDateTime", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadDateTime, bsontype: TypeDateTime}, err},
					{"ReadNull", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadNull, bsontype: TypeNull}, err},
					{"ReadRegex", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadRegex, bsontype: TypeRegex}, err},
					{"ReadDBPointer", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadDBPointer, bsontype: TypeDBPointer}, err},
					{"ReadJavascript", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadJavascript, bsontype: TypeJavaScript}, err},
					{"ReadSymbol", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadSymbol, bsontype: TypeSymbol}, err},
					{
						"ReadCodeWithScope (Lookup)", DecodeContext{Registry: NewEmptyRegistryBuilder().Build()},
						&llValueReaderWriter{t: t, err: err, errAfter: llvrwReadCodeWithScope, bsontype: TypeCodeWithScope},
						err,
					},
					{"ReadInt32", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadInt32, bsontype: TypeInt32}, err},
					{"ReadInt64", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadInt64, bsontype: TypeInt64}, err},
					{"ReadTimestamp", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadTimestamp, bsontype: TypeTimestamp}, err},
					{"ReadDecimal128", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadDecimal128, bsontype: TypeDecimal128}, err},
					{"ReadMinKey", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadMinKey, bsontype: TypeMinKey}, err},
					{"ReadMaxKey", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadMaxKey, bsontype: TypeMaxKey}, err},
					{"Invalid Type", dc, &llValueReaderWriter{t: t, bsontype: Type(0)}, fmt.Errorf("Cannot read unknown BSON type %s", Type(0))},
				}

				for _, tc := range testCases {
					t.Run(tc.name, func(t *testing.T) {
						err := (&ArrayCodec{}).DecodeValue(tc.dc, tc.llvrw, new(*Array))
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
				dc := DecodeContext{Registry: NewRegistryBuilder().Build()}

				dvr := newDocumentValueReader(NewDocument(EC.Array("", want)))
				dr, err := dvr.ReadDocument()
				noerr(t, err)
				_, vr, err := dr.ReadElement()
				noerr(t, err)

				var got *Array
				err = (&ArrayCodec{}).DecodeValue(dc, vr, &got)
				noerr(t, err)
				if !got.Equal(want) {
					t.Error("Documents do not match")
					t.Errorf("\ngot :%v\nwant:%v", got, want)
				}
			})
		})
	})
	t.Run("SliceCodec/DecodeValue/can't set slice", func(t *testing.T) {
		var val []string
		want := fmt.Errorf("%T can only be used to decode non-nil pointers to slice or array values, got %T", &SliceCodec{}, val)
		got := (&SliceCodec{}).DecodeValue(DecodeContext{}, nil, val)
		if !compareErrors(got, want) {
			t.Errorf("Errors do not match. got %v; want %v", got, want)
		}
	})
	t.Run("SliceCodec/DecodeValue/too many elements", func(t *testing.T) {
		dvr := newDocumentValueReader(NewDocument(EC.ArrayFromElements("foo", VC.String("foo"), VC.String("bar"))))
		dr, err := dvr.ReadDocument()
		noerr(t, err)
		_, vr, err := dr.ReadElement()
		noerr(t, err)
		var val [1]string
		want := fmt.Errorf("more elements returned in array than can fit inside %T", val)

		dc := DecodeContext{Registry: NewRegistryBuilder().Build()}
		got := (&SliceCodec{}).DecodeValue(dc, vr, &val)
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
				"map[string]*Element",
				map[string]*Element{"Z": EC.Int32("Z", 12345)},
				docToBytes(NewDocument(
					EC.Int32("Z", 12345),
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
					N  *Element
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
					N:  EC.Null("n"),
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
					EC.Null("n"),
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
				)),
				nil,
			},
		}

		t.Run("Encode", func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					b := make([]byte, 0, 512)
					vw := newValueWriterFromSlice(b)
					enc, err := NewEncoderv2(NewRegistryBuilder().Build(), vw)
					noerr(t, err)
					err = enc.Encode(tc.value)
					if err != tc.err {
						t.Errorf("Did not receive expected error. got %v; want %v", err, tc.err)
					}
					b = vw.buf
					if diff := cmp.Diff(b, tc.b); diff != "" {
						t.Errorf("Bytes written differ: (-got +want)\n%s", diff)
						t.Errorf("Bytes\ngot: %v\nwant:%v\n", b, tc.b)
						t.Errorf("Readers\ngot: %v\nwant:%v\n", Reader(b), Reader(tc.b))
					}
				})
			}
		})

		t.Run("Decode", func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					vr := newValueReader(tc.b)
					dec, err := NewDecoderv2(NewRegistryBuilder().Build(), vr)
					noerr(t, err)
					gotVal := reflect.New(reflect.TypeOf(tc.value))
					err = dec.Decode(gotVal.Interface())
					noerr(t, err)
					got := gotVal.Elem().Interface()
					want := tc.value
					if diff := cmp.Diff(
						got, want,
						cmp.Comparer(compareElements),
						cmp.Comparer(compareValues),
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

func compareValues(v1, v2 *Value) bool     { return v1.equal(v2) }
func compareElements(e1, e2 *Element) bool { return e1.equal(e2) }
func compareStrings(s1, s2 string) bool    { return s1 == s2 }

type noPrivateFields struct {
	a string
}

func compareNoPrivateFields(npf1, npf2 noPrivateFields) bool {
	return npf1.a != npf2.a // We don't want these to be equal
}
