package bson

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
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

	type enccase struct {
		name   string
		val    interface{}
		ectx   *EncodeContext
		llvrw  *llValueReaderWriter
		invoke interface{}
		err    error
	}
	type deccase struct {
		name   string
		val    interface{}
		dctx   *DecodeContext
		llvrw  *llValueReaderWriter
		invoke interface{}
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
					nil,
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
					nil,
					CodecDecodeError{Codec: &BooleanCodec{}, Types: []interface{}{bool(true)}, Received: &wrong},
				},
				{
					"type not boolean",
					bool(false),
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					nil,
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
					nil,
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
					nil,
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
					nil,
					fmt.Errorf("cannot decode %v into an integer type", TypeString),
				},
				{
					"ReadInt32 error",
					0,
					nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0), err: errors.New("ReadInt32 error")},
					llvrwReadInt32,
					errors.New("ReadInt32 error"),
				},
				{
					"ReadInt64 error",
					0,
					nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(0), err: errors.New("ReadInt64 error")},
					llvrwReadInt64,
					errors.New("ReadInt64 error"),
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
					nil,
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
					nil,
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
				{"uint64/fast path - overflow", uint64(1 << 63), nil, nil, nil, fmt.Errorf("%d overflows int64", uint(1<<63))},
				{"uint/fast path - overflow", uint(1 << 63), nil, nil, nil, fmt.Errorf("%d overflows int64", uint(1<<63))},
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
				{"uint64/reflection path - overflow", myuint64(1 << 63), nil, nil, nil, fmt.Errorf("%d overflows int64", uint(1<<63))},
				{"uint/reflection path - overflow", myuint(1 << 63), nil, nil, nil, fmt.Errorf("%d overflows int64", uint(1<<63))},
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
					nil,
					fmt.Errorf("cannot decode %v into an integer type", TypeString),
				},
				{
					"ReadInt32 error",
					0,
					nil,
					&llValueReaderWriter{bsontype: TypeInt32, readval: int32(0), err: errors.New("ReadInt32 error")},
					llvrwReadInt32,
					errors.New("ReadInt32 error"),
				},
				{
					"ReadInt64 error",
					0,
					nil,
					&llValueReaderWriter{bsontype: TypeInt64, readval: int64(0), err: errors.New("ReadInt64 error")},
					llvrwReadInt64,
					errors.New("ReadInt64 error"),
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
					nil,
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
					nil,
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
					nil,
					fmt.Errorf("cannot decode %v into a float32 or float64 type", TypeString),
				},
				{
					"ReadDouble error",
					0,
					nil,
					&llValueReaderWriter{bsontype: TypeDouble, readval: float64(0), err: errors.New("ReadDouble error")},
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
					nil,
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
					nil,
					CodecEncodeError{Codec: &StringCodec{}, Types: []interface{}{string("")}, Received: wrong},
				},
				{"fast path", string("foobar"), nil, nil, llvrwWriteString, nil},
				{"reflection path", mystring("foobarbaz"), nil, nil, llvrwWriteString, nil},
			},
			[]deccase{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					nil,
					CodecDecodeError{Codec: &StringCodec{}, Types: []interface{}{string("")}, Received: &wrong},
				},
				{
					"type not string",
					string(""),
					nil,
					&llValueReaderWriter{bsontype: TypeBoolean},
					nil,
					fmt.Errorf("cannot decode %v into a string", TypeBoolean),
				},
				{
					"fast path",
					string("foobar"),
					nil,
					&llValueReaderWriter{bsontype: TypeString, readval: string("foobar")},
					llvrwReadString,
					nil,
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
					&llValueReaderWriter{bsontype: TypeString, readval: string("foobarbazqux"), err: errors.New("ReadString Error")},
					llvrwReadString, errors.New("ReadString Error"),
				},
				{
					"can set false",
					cansetreflectiontest,
					nil,
					&llValueReaderWriter{bsontype: TypeString},
					nil,
					fmt.Errorf("%T can only be used to decode settable (non-nil) values", &StringCodec{}),
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
						var got interface{}
						var unwrap bool
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
						rtype := reflect.TypeOf(rc.val)
						if rtype.Kind() == reflect.Ptr {
							got = reflect.New(rtype).Elem().Interface()
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
}
