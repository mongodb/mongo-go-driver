package bson

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

func TestProvidedCodecs(t *testing.T) {
	var wrong func(string, string) string = func(string, string) string { return "wrong" }
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
					fmt.Errorf("%T can only convert float64 to an integer type when truncation is enabled", &IntCodec{}),
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
					fmt.Errorf("%T can only convert float64 to an integer type when truncation is enabled", &UintCodec{}),
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
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode non-nil *time.Time values, got %T", &TimeCodec{}, (*int64)(nil)),
				},
				{
					"*time.Time is nil",
					(*time.Time)(nil),
					nil,
					&llValueReaderWriter{bsontype: TypeDateTime, readval: int64(1234567890)},
					llvrwNothing,
					fmt.Errorf("%T can only be used to decode non-nil *time.Time values, got %T", &TimeCodec{}, (*time.Time)(nil)),
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
					"*time.Time",
					&now,
					nil,
					&llValueReaderWriter{bsontype: TypeDateTime, readval: int64(now.UnixNano() / int64(time.Millisecond))},
					llvrwReadDateTime,
					nil,
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
								got = val.Interface()
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
						if rc.err == nil && !cmp.Equal(got, want) {
							t.Errorf("Values do not match. got (%T)%v; want (%T)%v", got, got, want, want)
						}
					})
				}
			})
		})
	}

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
					// EC.ArrayFromElements("d", VC.Null()),
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
				want := CodecDecodeError{Codec: &DocumentCodec{}, Types: []interface{}{(*Document)(nil)}, Received: val}
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
				got := (&DocumentCodec{}).DecodeValue(DecodeContext{}, llvrw, NewDocument())
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
						err := (&DocumentCodec{}).decodeDocument(tc.dc, tc.llvrw, NewDocument())
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
					// EC.ArrayFromElements("d", VC.Null()),
					EC.BinaryWithSubtype("e", []byte{0x01, 0x02, 0x03}, 0xFF), EC.Undefined("f"),
					EC.ObjectID("g", oid), EC.Boolean("h", true), EC.DateTime("i", 1234567890), EC.Null("j"), EC.Regex("k", "foo", "bar"),
					EC.DBPointer("l", "foobar", oid), EC.JavaScript("m", "var hello = 'world';"), EC.Symbol("n", "bazqux"),
					EC.CodeWithScope("o", "var hello = 'world';", NewDocument(EC.Null("ab"))), EC.Int32("p", 12345),
					EC.Timestamp("q", 10, 20), EC.Int64("r", 1234567890), EC.Decimal128("s", d128), EC.MinKey("t"), EC.MaxKey("u"),
				)
				got := NewDocument()
				dc := DecodeContext{Registry: NewRegistryBuilder().Build()}
				err := (&DocumentCodec{}).DecodeValue(dc, newDocumentValueReader(want), got)
				noerr(t, err)
				if !got.Equal(want) {
					t.Error("Documents do not match")
					t.Errorf("\ngot :%v\nwant:%v", got, want)
				}
			})
		})
	})
}
