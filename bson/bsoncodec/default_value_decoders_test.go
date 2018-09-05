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
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/internal/llbson"
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
		llvrw  *llValueReaderWriter
		invoke llvrwInvoked
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
					&llValueReaderWriter{bsontype: bson.TypeBoolean},
					llvrwNothing,
					ValueDecoderError{Name: "BooleanDecodeValue", Types: []interface{}{bool(true)}, Received: &wrong},
				},
				{
					"type not boolean",
					bool(false),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a boolean", bson.TypeString),
				},
				{
					"fast path",
					bool(true),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeBoolean, readval: bool(true)},
					llvrwReadBoolean,
					nil,
				},
				{
					"reflection path",
					mybool(true),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeBoolean, readval: bool(true)},
					llvrwReadBoolean,
					nil,
				},
				{
					"reflection path error",
					mybool(true),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeBoolean, readval: bool(true), err: errors.New("ReadBoolean Error")},
					llvrwReadBoolean, errors.New("ReadBoolean Error"),
				},
				{
					"can set false",
					cansetreflectiontest,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeBoolean},
					llvrwNothing,
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
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0)},
					llvrwReadInt32,
					ValueDecoderError{Name: "IntDecodeValue", Types: intAllowedDecodeTypes, Received: &wrong},
				},
				{
					"type not int32/int64",
					0,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into an integer type", bson.TypeString),
				},
				{
					"ReadInt32 error",
					0,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0), err: errors.New("ReadInt32 error"), errAfter: llvrwReadInt32},
					llvrwReadInt32,
					errors.New("ReadInt32 error"),
				},
				{
					"ReadInt64 error",
					0,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(0), err: errors.New("ReadInt64 error"), errAfter: llvrwReadInt64},
					llvrwReadInt64,
					errors.New("ReadInt64 error"),
				},
				{
					"ReadDouble error",
					0,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(0), err: errors.New("ReadDouble error"), errAfter: llvrwReadDouble},
					llvrwReadDouble,
					errors.New("ReadDouble error"),
				},
				{
					"ReadDouble", int64(3), &DecodeContext{},
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(3.00)}, llvrwReadDouble,
					nil,
				},
				{
					"ReadDouble (truncate)", int64(3), &DecodeContext{Truncate: true},
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(3.14)}, llvrwReadDouble,
					nil,
				},
				{
					"ReadDouble (no truncate)", int64(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(3.14)}, llvrwReadDouble,
					errors.New("IntDecodeValue can only truncate float64 to an integer type when truncation is enabled"),
				},
				{
					"ReadDouble overflows int64", int64(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: math.MaxFloat64}, llvrwReadDouble,
					fmt.Errorf("%g overflows int64", math.MaxFloat64),
				},
				{"int8/fast path", int8(127), nil, &llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(127)}, llvrwReadInt32, nil},
				{"int16/fast path", int16(32676), nil, &llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(32676)}, llvrwReadInt32, nil},
				{"int32/fast path", int32(1234), nil, &llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(1234)}, llvrwReadInt32, nil},
				{"int64/fast path", int64(1234), nil, &llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(1234)}, llvrwReadInt64, nil},
				{"int/fast path", int(1234), nil, &llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(1234)}, llvrwReadInt64, nil},
				{
					"int8/fast path - nil", (*int8)(nil), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0)}, llvrwReadInt32,
					errors.New("IntDecodeValue can only be used to decode non-nil *int8"),
				},
				{
					"int16/fast path - nil", (*int16)(nil), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0)}, llvrwReadInt32,
					errors.New("IntDecodeValue can only be used to decode non-nil *int16"),
				},
				{
					"int32/fast path - nil", (*int32)(nil), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0)}, llvrwReadInt32,
					errors.New("IntDecodeValue can only be used to decode non-nil *int32"),
				},
				{
					"int64/fast path - nil", (*int64)(nil), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0)}, llvrwReadInt32,
					errors.New("IntDecodeValue can only be used to decode non-nil *int64"),
				},
				{
					"int/fast path - nil", (*int)(nil), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0)}, llvrwReadInt32,
					errors.New("IntDecodeValue can only be used to decode non-nil *int"),
				},
				{
					"int8/fast path - overflow", int8(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(129)}, llvrwReadInt32,
					fmt.Errorf("%d overflows int8", 129),
				},
				{
					"int16/fast path - overflow", int16(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(32768)}, llvrwReadInt32,
					fmt.Errorf("%d overflows int16", 32768),
				},
				{
					"int32/fast path - overflow", int32(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(2147483648)}, llvrwReadInt64,
					fmt.Errorf("%d overflows int32", 2147483648),
				},
				{
					"int8/fast path - overflow (negative)", int8(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(-129)}, llvrwReadInt32,
					fmt.Errorf("%d overflows int8", -129),
				},
				{
					"int16/fast path - overflow (negative)", int16(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(-32769)}, llvrwReadInt32,
					fmt.Errorf("%d overflows int16", -32769),
				},
				{
					"int32/fast path - overflow (negative)", int32(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(-2147483649)}, llvrwReadInt64,
					fmt.Errorf("%d overflows int32", -2147483649),
				},
				{
					"int8/reflection path", myint8(127), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(127)}, llvrwReadInt32,
					nil,
				},
				{
					"int16/reflection path", myint16(255), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(255)}, llvrwReadInt32,
					nil,
				},
				{
					"int32/reflection path", myint32(511), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(511)}, llvrwReadInt32,
					nil,
				},
				{
					"int64/reflection path", myint64(1023), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(1023)}, llvrwReadInt32,
					nil,
				},
				{
					"int/reflection path", myint(2047), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(2047)}, llvrwReadInt32,
					nil,
				},
				{
					"int8/reflection path - overflow", myint8(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(129)}, llvrwReadInt32,
					fmt.Errorf("%d overflows int8", 129),
				},
				{
					"int16/reflection path - overflow", myint16(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(32768)}, llvrwReadInt32,
					fmt.Errorf("%d overflows int16", 32768),
				},
				{
					"int32/reflection path - overflow", myint32(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(2147483648)}, llvrwReadInt64,
					fmt.Errorf("%d overflows int32", 2147483648),
				},
				{
					"int8/reflection path - overflow (negative)", myint8(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(-129)}, llvrwReadInt32,
					fmt.Errorf("%d overflows int8", -129),
				},
				{
					"int16/reflection path - overflow (negative)", myint16(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(-32769)}, llvrwReadInt32,
					fmt.Errorf("%d overflows int16", -32769),
				},
				{
					"int32/reflection path - overflow (negative)", myint32(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(-2147483649)}, llvrwReadInt64,
					fmt.Errorf("%d overflows int32", -2147483649),
				},
				{
					"can set false",
					cansetreflectiontest,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0)},
					llvrwNothing,
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
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0)},
					llvrwReadInt32,
					ValueDecoderError{Name: "UintDecodeValue", Types: uintAllowedDecodeTypes, Received: &wrong},
				},
				{
					"type not int32/int64",
					0,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into an integer type", bson.TypeString),
				},
				{
					"ReadInt32 error",
					uint(0),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0), err: errors.New("ReadInt32 error"), errAfter: llvrwReadInt32},
					llvrwReadInt32,
					errors.New("ReadInt32 error"),
				},
				{
					"ReadInt64 error",
					uint(0),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(0), err: errors.New("ReadInt64 error"), errAfter: llvrwReadInt64},
					llvrwReadInt64,
					errors.New("ReadInt64 error"),
				},
				{
					"ReadDouble error",
					0,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(0), err: errors.New("ReadDouble error"), errAfter: llvrwReadDouble},
					llvrwReadDouble,
					errors.New("ReadDouble error"),
				},
				{
					"ReadDouble", uint64(3), &DecodeContext{},
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(3.00)}, llvrwReadDouble,
					nil,
				},
				{
					"ReadDouble (truncate)", uint64(3), &DecodeContext{Truncate: true},
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(3.14)}, llvrwReadDouble,
					nil,
				},
				{
					"ReadDouble (no truncate)", uint64(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(3.14)}, llvrwReadDouble,
					errors.New("UintDecodeValue can only truncate float64 to an integer type when truncation is enabled"),
				},
				{
					"ReadDouble overflows int64", uint64(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: math.MaxFloat64}, llvrwReadDouble,
					fmt.Errorf("%g overflows int64", math.MaxFloat64),
				},
				{"uint8/fast path", uint8(127), nil, &llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(127)}, llvrwReadInt32, nil},
				{"uint16/fast path", uint16(255), nil, &llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(255)}, llvrwReadInt32, nil},
				{"uint32/fast path", uint32(1234), nil, &llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(1234)}, llvrwReadInt32, nil},
				{"uint64/fast path", uint64(1234), nil, &llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(1234)}, llvrwReadInt64, nil},
				{"uint/fast path", uint(1234), nil, &llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(1234)}, llvrwReadInt64, nil},
				{
					"uint8/fast path - nil", (*uint8)(nil), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0)}, llvrwReadInt32,
					errors.New("UintDecodeValue can only be used to decode non-nil *uint8"),
				},
				{
					"uint16/fast path - nil", (*uint16)(nil), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0)}, llvrwReadInt32,
					errors.New("UintDecodeValue can only be used to decode non-nil *uint16"),
				},
				{
					"uint32/fast path - nil", (*uint32)(nil), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0)}, llvrwReadInt32,
					errors.New("UintDecodeValue can only be used to decode non-nil *uint32"),
				},
				{
					"uint64/fast path - nil", (*uint64)(nil), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0)}, llvrwReadInt32,
					errors.New("UintDecodeValue can only be used to decode non-nil *uint64"),
				},
				{
					"uint/fast path - nil", (*uint)(nil), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0)}, llvrwReadInt32,
					errors.New("UintDecodeValue can only be used to decode non-nil *uint"),
				},
				{
					"uint8/fast path - overflow", uint8(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(1 << 8)}, llvrwReadInt32,
					fmt.Errorf("%d overflows uint8", 1<<8),
				},
				{
					"uint16/fast path - overflow", uint16(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(1 << 16)}, llvrwReadInt32,
					fmt.Errorf("%d overflows uint16", 1<<16),
				},
				{
					"uint32/fast path - overflow", uint32(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(1 << 32)}, llvrwReadInt64,
					fmt.Errorf("%d overflows uint32", 1<<32),
				},
				{
					"uint8/fast path - overflow (negative)", uint8(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(-1)}, llvrwReadInt32,
					fmt.Errorf("%d overflows uint8", -1),
				},
				{
					"uint16/fast path - overflow (negative)", uint16(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(-1)}, llvrwReadInt32,
					fmt.Errorf("%d overflows uint16", -1),
				},
				{
					"uint32/fast path - overflow (negative)", uint32(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(-1)}, llvrwReadInt64,
					fmt.Errorf("%d overflows uint32", -1),
				},
				{
					"uint64/fast path - overflow (negative)", uint64(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(-1)}, llvrwReadInt64,
					fmt.Errorf("%d overflows uint64", -1),
				},
				{
					"uint/fast path - overflow (negative)", uint(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(-1)}, llvrwReadInt64,
					fmt.Errorf("%d overflows uint", -1),
				},
				{
					"uint8/reflection path", myuint8(127), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(127)}, llvrwReadInt32,
					nil,
				},
				{
					"uint16/reflection path", myuint16(255), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(255)}, llvrwReadInt32,
					nil,
				},
				{
					"uint32/reflection path", myuint32(511), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(511)}, llvrwReadInt32,
					nil,
				},
				{
					"uint64/reflection path", myuint64(1023), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(1023)}, llvrwReadInt32,
					nil,
				},
				{
					"uint/reflection path", myuint(2047), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(2047)}, llvrwReadInt32,
					nil,
				},
				{
					"uint8/reflection path - overflow", myuint8(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(1 << 8)}, llvrwReadInt32,
					fmt.Errorf("%d overflows uint8", 1<<8),
				},
				{
					"uint16/reflection path - overflow", myuint16(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(1 << 16)}, llvrwReadInt32,
					fmt.Errorf("%d overflows uint16", 1<<16),
				},
				{
					"uint32/reflection path - overflow", myuint32(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(1 << 32)}, llvrwReadInt64,
					fmt.Errorf("%d overflows uint32", 1<<32),
				},
				{
					"uint8/reflection path - overflow (negative)", myuint8(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(-1)}, llvrwReadInt32,
					fmt.Errorf("%d overflows uint8", -1),
				},
				{
					"uint16/reflection path - overflow (negative)", myuint16(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(-1)}, llvrwReadInt32,
					fmt.Errorf("%d overflows uint16", -1),
				},
				{
					"uint32/reflection path - overflow (negative)", myuint32(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(-1)}, llvrwReadInt64,
					fmt.Errorf("%d overflows uint32", -1),
				},
				{
					"uint64/reflection path - overflow (negative)", myuint64(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(-1)}, llvrwReadInt64,
					fmt.Errorf("%d overflows uint64", -1),
				},
				{
					"uint/reflection path - overflow (negative)", myuint(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(-1)}, llvrwReadInt64,
					fmt.Errorf("%d overflows uint", -1),
				},
				{
					"can set false",
					cansetreflectiontest,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0)},
					llvrwNothing,
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
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(0)},
					llvrwReadDouble,
					ValueDecoderError{Name: "FloatDecodeValue", Types: []interface{}{(*float32)(nil), (*float64)(nil)}, Received: &wrong},
				},
				{
					"type not double",
					0,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a float32 or float64 type", bson.TypeString),
				},
				{
					"ReadDouble error",
					float64(0),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(0), err: errors.New("ReadDouble error"), errAfter: llvrwReadDouble},
					llvrwReadDouble,
					errors.New("ReadDouble error"),
				},
				{
					"ReadInt32 error",
					float64(0),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0), err: errors.New("ReadInt32 error"), errAfter: llvrwReadInt32},
					llvrwReadInt32,
					errors.New("ReadInt32 error"),
				},
				{
					"ReadInt64 error",
					float64(0),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(0), err: errors.New("ReadInt64 error"), errAfter: llvrwReadInt64},
					llvrwReadInt64,
					errors.New("ReadInt64 error"),
				},
				{
					"float64/int32", float32(32.0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(32)}, llvrwReadInt32,
					nil,
				},
				{
					"float64/int64", float32(64.0), nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(64)}, llvrwReadInt64,
					nil,
				},
				{
					"float32/fast path (equal)", float32(3.0), nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(3.0)}, llvrwReadDouble,
					nil,
				},
				{
					"float64/fast path", float64(3.14159), nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(3.14159)}, llvrwReadDouble,
					nil,
				},
				{
					"float32/fast path (truncate)", float32(3.14), &DecodeContext{Truncate: true},
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(3.14)}, llvrwReadDouble,
					nil,
				},
				{
					"float32/fast path (no truncate)", float32(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(3.14)}, llvrwReadDouble,
					errors.New("FloatDecodeValue can only convert float64 to float32 when truncation is allowed"),
				},
				{
					"float32/fast path - nil", (*float32)(nil), nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(0)}, llvrwReadDouble,
					errors.New("FloatDecodeValue can only be used to decode non-nil *float32"),
				},
				{
					"float64/fast path - nil", (*float64)(nil), nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(0)}, llvrwReadDouble,
					errors.New("FloatDecodeValue can only be used to decode non-nil *float64"),
				},
				{
					"float32/reflection path (equal)", myfloat32(3.0), nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(3.0)}, llvrwReadDouble,
					nil,
				},
				{
					"float64/reflection path", myfloat64(3.14159), nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(3.14159)}, llvrwReadDouble,
					nil,
				},
				{
					"float32/reflection path (truncate)", myfloat32(3.14), &DecodeContext{Truncate: true},
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(3.14)}, llvrwReadDouble,
					nil,
				},
				{
					"float32/reflection path (no truncate)", myfloat32(0), nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(3.14)}, llvrwReadDouble,
					errors.New("FloatDecodeValue can only convert float64 to float32 when truncation is allowed"),
				},
				{
					"can set false",
					cansetreflectiontest,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(0)},
					llvrwNothing,
					errors.New("FloatDecodeValue can only be used to decode settable (non-nil) values"),
				},
			},
		},
		{
			"StringDecodeValue",
			ValueDecoderFunc(dvd.StringDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString, readval: string("")},
					llvrwReadString,
					ValueDecoderError{Name: "StringDecodeValue", Types: []interface{}{(*string)(nil), (*bson.JavaScriptCode)(nil), (*bson.Symbol)(nil)}, Received: &wrong},
				},
				{
					"type not string",
					string(""),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeBoolean},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a string type", bson.TypeBoolean),
				},
				{
					"ReadString error",
					string(""),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString, readval: string(""), err: errors.New("ReadString error"), errAfter: llvrwReadString},
					llvrwReadString,
					errors.New("ReadString error"),
				},
				{
					"ReadJavaScript error",
					string(""),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeJavaScript, readval: string(""), err: errors.New("ReadJS error"), errAfter: llvrwReadJavascript},
					llvrwReadJavascript,
					errors.New("ReadJS error"),
				},
				{
					"ReadSymbol error",
					string(""),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeSymbol, readval: string(""), err: errors.New("ReadSymbol error"), errAfter: llvrwReadSymbol},
					llvrwReadSymbol,
					errors.New("ReadSymbol error"),
				},
				{
					"string/fast path",
					string("foobar"),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString, readval: string("foobar")},
					llvrwReadString,
					nil,
				},
				{
					"JavaScript/fast path",
					bson.JavaScriptCode("var hello = 'world';"),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeJavaScript, readval: string("var hello = 'world';")},
					llvrwReadJavascript,
					nil,
				},
				{
					"Symbol/fast path",
					bson.Symbol("foobarbaz"),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeSymbol, readval: bson.Symbol("foobarbaz")},
					llvrwReadSymbol,
					nil,
				},
				{
					"string/fast path - nil", (*string)(nil), nil,
					&llValueReaderWriter{bsontype: bson.TypeString, readval: string("")}, llvrwReadString,
					errors.New("StringDecodeValue can only be used to decode non-nil *string"),
				},
				{
					"JavaScript/fast path - nil", (*bson.JavaScriptCode)(nil), nil,
					&llValueReaderWriter{bsontype: bson.TypeJavaScript, readval: string("")}, llvrwReadJavascript,
					errors.New("StringDecodeValue can only be used to decode non-nil *JavaScriptCode"),
				},
				{
					"Symbol/fast path - nil", (*bson.Symbol)(nil), nil,
					&llValueReaderWriter{bsontype: bson.TypeSymbol, readval: bson.Symbol("")}, llvrwReadSymbol,
					errors.New("StringDecodeValue can only be used to decode non-nil *Symbol"),
				},
				{
					"reflection path",
					mystring("foobarbaz"),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString, readval: string("foobarbaz")},
					llvrwReadString,
					nil,
				},
				{
					"reflection path error",
					mystring("foobarbazqux"),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString, readval: string("foobarbazqux"), err: errors.New("ReadString Error"), errAfter: llvrwReadString},
					llvrwReadString, errors.New("ReadString Error"),
				},
				{
					"can set false",
					cansetreflectiontest,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString, readval: string("")},
					llvrwNothing,
					errors.New("StringDecodeValue can only be used to decode settable (non-nil) values"),
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
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(0)},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a time.Time", bson.TypeInt32),
				},
				{
					"type not *time.Time",
					int64(0),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeDateTime, readval: int64(1234567890)},
					llvrwReadDateTime,
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
					&llValueReaderWriter{bsontype: bson.TypeDateTime, readval: int64(0), err: errors.New("ReadDateTime error"), errAfter: llvrwReadDateTime},
					llvrwReadDateTime,
					errors.New("ReadDateTime error"),
				},
				{
					"time.Time",
					now,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeDateTime, readval: int64(now.UnixNano() / int64(time.Millisecond))},
					llvrwReadDateTime,
					nil,
				},
				{
					"*time.Time",
					&now,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeDateTime, readval: int64(now.UnixNano() / int64(time.Millisecond))},
					llvrwReadDateTime,
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
					&llValueReaderWriter{},
					llvrwNothing,
					errors.New("MapDecodeValue can only decode settable maps with string keys"),
				},
				{
					"wrong kind (non-string key)",
					map[int]interface{}{},
					nil,
					&llValueReaderWriter{},
					llvrwNothing,
					errors.New("MapDecodeValue can only decode settable maps with string keys"),
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
					ErrNoDecoder{Type: reflect.TypeOf(string(""))},
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
					&llValueReaderWriter{bsontype: bson.TypeString, err: errors.New("dv error"), errAfter: llvrwReadString},
					llvrwReadString,
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
					&llValueReaderWriter{},
					llvrwNothing,
					fmt.Errorf("SliceDecodeValue can only decode settable slice and array values, got %T", &wrong),
				},
				{
					"can set false",
					(*[]string)(nil),
					nil,
					&llValueReaderWriter{},
					llvrwNothing,
					fmt.Errorf("SliceDecodeValue can only be used to decode non-nil pointers to slice or array values, got %T", (*[]string)(nil)),
				},
				{
					"Not Type Array",
					[]interface{}{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					errors.New("cannot decode string into a slice"),
				},
				{
					"ReadArray Error",
					[]interface{}{},
					nil,
					&llValueReaderWriter{err: errors.New("ra error"), errAfter: llvrwReadArray, bsontype: bson.TypeArray},
					llvrwReadArray,
					errors.New("ra error"),
				},
				{
					"Lookup Error",
					[]string{},
					&DecodeContext{Registry: NewEmptyRegistryBuilder().Build()},
					&llValueReaderWriter{bsontype: bson.TypeArray},
					llvrwReadArray,
					ErrNoDecoder{Type: reflect.TypeOf(string(""))},
				},
				{
					"ReadValue Error",
					[]string{},
					&DecodeContext{Registry: NewRegistryBuilder().Build()},
					&llValueReaderWriter{err: errors.New("rv error"), errAfter: llvrwReadValue, bsontype: bson.TypeArray},
					llvrwReadValue,
					errors.New("rv error"),
				},
				{
					"DecodeValue Error",
					[]string{},
					&DecodeContext{Registry: NewRegistryBuilder().Build()},
					&llValueReaderWriter{bsontype: bson.TypeArray},
					llvrwReadValue,
					errors.New("cannot decode array into a string type"),
				},
			},
		},
		{
			"BinaryDecodeValue",
			ValueDecoderFunc(dvd.BinaryDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeBinary, readval: bson.Binary{}},
					llvrwReadBinary,
					ValueDecoderError{Name: "BinaryDecodeValue", Types: []interface{}{(*bson.Binary)(nil)}, Received: &wrong},
				},
				{
					"type not binary",
					bson.Binary{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a Binary", bson.TypeString),
				},
				{
					"ReadBinary Error",
					bson.Binary{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeBinary, err: errors.New("rb error"), errAfter: llvrwReadBinary},
					llvrwReadBinary,
					errors.New("rb error"),
				},
				{
					"Binary/success",
					bson.Binary{Data: []byte{0x01, 0x02, 0x03}, Subtype: 0xFF},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeBinary, readval: bson.Binary{Data: []byte{0x01, 0x02, 0x03}, Subtype: 0xFF}},
					llvrwReadBinary,
					nil,
				},
				{
					"*Binary/success",
					&bson.Binary{Data: []byte{0x01, 0x02, 0x03}, Subtype: 0xFF},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeBinary, readval: bson.Binary{Data: []byte{0x01, 0x02, 0x03}, Subtype: 0xFF}},
					llvrwReadBinary,
					nil,
				},
			},
		},
		{
			"UndefinedDecodeValue",
			ValueDecoderFunc(dvd.UndefinedDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeUndefined},
					llvrwNothing,
					ValueDecoderError{Name: "UndefinedDecodeValue", Types: []interface{}{(*bson.Undefinedv2)(nil)}, Received: &wrong},
				},
				{
					"type not undefined",
					bson.Undefinedv2{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into an Undefined", bson.TypeString),
				},
				{
					"ReadUndefined Error",
					bson.Undefinedv2{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeUndefined, err: errors.New("ru error"), errAfter: llvrwReadUndefined},
					llvrwReadUndefined,
					errors.New("ru error"),
				},
				{
					"ReadUndefined/success",
					bson.Undefinedv2{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeUndefined},
					llvrwReadUndefined,
					nil,
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
					&llValueReaderWriter{bsontype: bson.TypeObjectID},
					llvrwNothing,
					ValueDecoderError{Name: "ObjectIDDecodeValue", Types: []interface{}{(*objectid.ObjectID)(nil)}, Received: &wrong},
				},
				{
					"type not objectID",
					objectid.ObjectID{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into an ObjectID", bson.TypeString),
				},
				{
					"ReadObjectID Error",
					objectid.ObjectID{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeObjectID, err: errors.New("roid error"), errAfter: llvrwReadObjectID},
					llvrwReadObjectID,
					errors.New("roid error"),
				},
				{
					"success",
					objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					nil,
					&llValueReaderWriter{
						bsontype: bson.TypeObjectID,
						readval:  objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					},
					llvrwReadObjectID,
					nil,
				},
			},
		},
		{
			"DateTimeDecodeValue",
			ValueDecoderFunc(dvd.DateTimeDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeDateTime},
					llvrwNothing,
					ValueDecoderError{Name: "DateTimeDecodeValue", Types: []interface{}{(*bson.DateTime)(nil)}, Received: &wrong},
				},
				{
					"type not datetime",
					bson.DateTime(0),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a DateTime", bson.TypeString),
				},
				{
					"ReadDateTime Error",
					bson.DateTime(0),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeDateTime, err: errors.New("rdt error"), errAfter: llvrwReadDateTime},
					llvrwReadDateTime,
					errors.New("rdt error"),
				},
				{
					"success",
					bson.DateTime(1234567890),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeDateTime, readval: int64(1234567890)},
					llvrwReadDateTime,
					nil,
				},
			},
		},
		{
			"NullDecodeValue",
			ValueDecoderFunc(dvd.NullDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeNull},
					llvrwNothing,
					ValueDecoderError{Name: "NullDecodeValue", Types: []interface{}{(*bson.Nullv2)(nil)}, Received: &wrong},
				},
				{
					"type not null",
					bson.Nullv2{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a Null", bson.TypeString),
				},
				{
					"ReadNull Error",
					bson.Nullv2{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeNull, err: errors.New("rn error"), errAfter: llvrwReadNull},
					llvrwReadNull,
					errors.New("rn error"),
				},
				{
					"success",
					bson.Nullv2{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeNull},
					llvrwReadNull,
					nil,
				},
			},
		},
		{
			"RegexDecodeValue",
			ValueDecoderFunc(dvd.RegexDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeRegex},
					llvrwNothing,
					ValueDecoderError{Name: "RegexDecodeValue", Types: []interface{}{(*bson.Regex)(nil)}, Received: &wrong},
				},
				{
					"type not regex",
					bson.Regex{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a Regex", bson.TypeString),
				},
				{
					"ReadRegex Error",
					bson.Regex{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeRegex, err: errors.New("rr error"), errAfter: llvrwReadRegex},
					llvrwReadRegex,
					errors.New("rr error"),
				},
				{
					"success",
					bson.Regex{Pattern: "foo", Options: "bar"},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeRegex, readval: bson.Regex{Pattern: "foo", Options: "bar"}},
					llvrwReadRegex,
					nil,
				},
			},
		},
		{
			"DBPointerDecodeValue",
			ValueDecoderFunc(dvd.DBPointerDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeDBPointer},
					llvrwNothing,
					ValueDecoderError{Name: "DBPointerDecodeValue", Types: []interface{}{(*bson.DBPointer)(nil)}, Received: &wrong},
				},
				{
					"type not dbpointer",
					bson.DBPointer{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a DBPointer", bson.TypeString),
				},
				{
					"ReadDBPointer Error",
					bson.DBPointer{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeDBPointer, err: errors.New("rdbp error"), errAfter: llvrwReadDBPointer},
					llvrwReadDBPointer,
					errors.New("rdbp error"),
				},
				{
					"success",
					bson.DBPointer{
						DB:      "foobar",
						Pointer: objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					},
					nil,
					&llValueReaderWriter{
						bsontype: bson.TypeDBPointer,
						readval: bson.DBPointer{
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
			"CodeWithScopeDecodeValue",
			ValueDecoderFunc(dvd.CodeWithScopeDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeCodeWithScope},
					llvrwNothing,
					ValueDecoderError{
						Name:     "CodeWithScopeDecodeValue",
						Types:    []interface{}{(*bson.CodeWithScope)(nil)},
						Received: &wrong,
					},
				},
				{
					"type not codewithscope",
					bson.CodeWithScope{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a CodeWithScope", bson.TypeString),
				},
				{
					"ReadCodeWithScope Error",
					bson.CodeWithScope{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeCodeWithScope, err: errors.New("rcws error"), errAfter: llvrwReadCodeWithScope},
					llvrwReadCodeWithScope,
					errors.New("rcws error"),
				},
				{
					"decodeDocument Error",
					bson.CodeWithScope{
						Code:  "var hello = 'world';",
						Scope: bson.NewDocument(bson.EC.Null("foo")),
					},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeCodeWithScope, err: errors.New("dd error"), errAfter: llvrwReadElement},
					llvrwReadElement,
					errors.New("dd error"),
				},
			},
		},
		{
			"TimestampDecodeValue",
			ValueDecoderFunc(dvd.TimestampDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeTimestamp},
					llvrwNothing,
					ValueDecoderError{Name: "TimestampDecodeValue", Types: []interface{}{(*bson.Timestamp)(nil)}, Received: &wrong},
				},
				{
					"type not timestamp",
					bson.Timestamp{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a Timestamp", bson.TypeString),
				},
				{
					"ReadTimestamp Error",
					bson.Timestamp{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeTimestamp, err: errors.New("rt error"), errAfter: llvrwReadTimestamp},
					llvrwReadTimestamp,
					errors.New("rt error"),
				},
				{
					"success",
					bson.Timestamp{T: 12345, I: 67890},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeTimestamp, readval: bson.Timestamp{T: 12345, I: 67890}},
					llvrwReadTimestamp,
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
					&llValueReaderWriter{bsontype: bson.TypeDecimal128},
					llvrwNothing,
					ValueDecoderError{Name: "Decimal128DecodeValue", Types: []interface{}{(*decimal.Decimal128)(nil)}, Received: &wrong},
				},
				{
					"type not decimal128",
					decimal.Decimal128{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a decimal.Decimal128", bson.TypeString),
				},
				{
					"ReadDecimal128 Error",
					decimal.Decimal128{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeDecimal128, err: errors.New("rd128 error"), errAfter: llvrwReadDecimal128},
					llvrwReadDecimal128,
					errors.New("rd128 error"),
				},
				{
					"success",
					d128,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeDecimal128, readval: d128},
					llvrwReadDecimal128,
					nil,
				},
			},
		},
		{
			"MinKeyDecodeValue",
			ValueDecoderFunc(dvd.MinKeyDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeMinKey},
					llvrwNothing,
					ValueDecoderError{Name: "MinKeyDecodeValue", Types: []interface{}{(*bson.MinKeyv2)(nil)}, Received: &wrong},
				},
				{
					"type not null",
					bson.MinKeyv2{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a MinKey", bson.TypeString),
				},
				{
					"ReadMinKey Error",
					bson.MinKeyv2{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeMinKey, err: errors.New("rn error"), errAfter: llvrwReadMinKey},
					llvrwReadMinKey,
					errors.New("rn error"),
				},
				{
					"success",
					bson.MinKeyv2{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeMinKey},
					llvrwReadMinKey,
					nil,
				},
			},
		},
		{
			"MaxKeyDecodeValue",
			ValueDecoderFunc(dvd.MaxKeyDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeMaxKey},
					llvrwNothing,
					ValueDecoderError{Name: "MaxKeyDecodeValue", Types: []interface{}{(*bson.MaxKeyv2)(nil)}, Received: &wrong},
				},
				{
					"type not null",
					bson.MaxKeyv2{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a MaxKey", bson.TypeString),
				},
				{
					"ReadMaxKey Error",
					bson.MaxKeyv2{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeMaxKey, err: errors.New("rn error"), errAfter: llvrwReadMaxKey},
					llvrwReadMaxKey,
					errors.New("rn error"),
				},
				{
					"success",
					bson.MaxKeyv2{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeMaxKey},
					llvrwReadMaxKey,
					nil,
				},
			},
		},
		{
			"ValueDecodeValue",
			ValueDecoderFunc(dvd.ValueDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					nil,
					llvrwNothing,
					ValueDecoderError{Name: "ValueDecodeValue", Types: []interface{}{(**bson.Value)(nil)}, Received: &wrong},
				},
				{"invalid value", (**bson.Value)(nil), nil, nil, llvrwNothing, errors.New("ValueDecodeValue can only be used to decode non-nil **Value")},
				{
					"success",
					bson.VC.Double(3.14159),
					&DecodeContext{Registry: NewRegistryBuilder().Build()},
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(3.14159)},
					llvrwReadDouble,
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
					&llValueReaderWriter{bsontype: bson.TypeObjectID},
					llvrwNothing,
					ValueDecoderError{Name: "JSONNumberDecodeValue", Types: []interface{}{(*json.Number)(nil)}, Received: &wrong},
				},
				{
					"type not double/int32/int64",
					json.Number(""),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a json.Number", bson.TypeString),
				},
				{
					"ReadDouble Error",
					json.Number(""),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, err: errors.New("rd error"), errAfter: llvrwReadDouble},
					llvrwReadDouble,
					errors.New("rd error"),
				},
				{
					"ReadInt32 Error",
					json.Number(""),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, err: errors.New("ri32 error"), errAfter: llvrwReadInt32},
					llvrwReadInt32,
					errors.New("ri32 error"),
				},
				{
					"ReadInt64 Error",
					json.Number(""),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, err: errors.New("ri64 error"), errAfter: llvrwReadInt64},
					llvrwReadInt64,
					errors.New("ri64 error"),
				},
				{
					"success/double",
					json.Number("3.14159"),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeDouble, readval: float64(3.14159)},
					llvrwReadDouble,
					nil,
				},
				{
					"success/int32",
					json.Number("12345"),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeInt32, readval: int32(12345)},
					llvrwReadInt32,
					nil,
				},
				{
					"success/int64",
					json.Number("1234567890"),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeInt64, readval: int64(1234567890)},
					llvrwReadInt64,
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
					&llValueReaderWriter{bsontype: bson.TypeInt32},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a *url.URL", bson.TypeInt32),
				},
				{
					"type not *url.URL",
					int64(0),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString, readval: string("http://example.com")},
					llvrwReadString,
					ValueDecoderError{Name: "URLDecodeValue", Types: []interface{}{(*url.URL)(nil), (**url.URL)(nil)}, Received: (*int64)(nil)},
				},
				{
					"ReadString error",
					url.URL{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString, err: errors.New("rs error"), errAfter: llvrwReadString},
					llvrwReadString,
					errors.New("rs error"),
				},
				{
					"url.Parse error",
					url.URL{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString, readval: string("not-valid-%%%%://")},
					llvrwReadString,
					errors.New("parse not-valid-%%%%://: first path segment in URL cannot contain colon"),
				},
				{
					"nil *url.URL",
					(*url.URL)(nil),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString, readval: string("http://example.com")},
					llvrwReadString,
					ValueDecoderError{Name: "URLDecodeValue", Types: []interface{}{(*url.URL)(nil), (**url.URL)(nil)}, Received: (*url.URL)(nil)},
				},
				{
					"nil **url.URL",
					(**url.URL)(nil),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString, readval: string("http://example.com")},
					llvrwReadString,
					ValueDecoderError{Name: "URLDecodeValue", Types: []interface{}{(*url.URL)(nil), (**url.URL)(nil)}, Received: (**url.URL)(nil)},
				},
				{
					"url.URL",
					url.URL{Scheme: "http", Host: "example.com"},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString, readval: string("http://example.com")},
					llvrwReadString,
					nil,
				},
				{
					"*url.URL",
					&url.URL{Scheme: "http", Host: "example.com"},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString, readval: string("http://example.com")},
					llvrwReadString,
					nil,
				},
			},
		},
		{
			"ReaderDecodeValue",
			ValueDecoderFunc(dvd.ReaderDecodeValue),
			[]subtest{
				{
					"wrong type",
					wrong,
					nil,
					&llValueReaderWriter{},
					llvrwNothing,
					ValueDecoderError{Name: "ReaderDecodeValue", Types: []interface{}{(*bson.Reader)(nil)}, Received: &wrong},
				},
				{
					"*Reader is nil",
					(*bson.Reader)(nil),
					nil,
					nil,
					llvrwNothing,
					errors.New("ReaderDecodeValue can only be used to decode non-nil *Reader"),
				},
				{
					"Copy error",
					bson.Reader{},
					nil,
					&llValueReaderWriter{err: errors.New("copy error"), errAfter: llvrwReadDocument},
					llvrwReadDocument,
					errors.New("copy error"),
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
					&llValueReaderWriter{bsontype: bson.TypeInt32},
					llvrwNothing,
					fmt.Errorf("cannot decode %v into a *[]byte", bson.TypeInt32),
				},
				{
					"type not *[]byte",
					int64(0),
					nil,
					&llValueReaderWriter{bsontype: bson.TypeBinary, readval: bson.Binary{}},
					llvrwNothing,
					ValueDecoderError{Name: "ByteSliceDecodeValue", Types: []interface{}{(*[]byte)(nil)}, Received: (*int64)(nil)},
				},
				{
					"ReadBinary error",
					[]byte{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeBinary, err: errors.New("rb error"), errAfter: llvrwReadBinary},
					llvrwReadBinary,
					errors.New("rb error"),
				},
				{
					"incorrect subtype",
					[]byte{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeBinary, readval: bson.Binary{Data: []byte{0x01, 0x02, 0x03}, Subtype: 0xFF}},
					llvrwReadBinary,
					fmt.Errorf("ByteSliceDecodeValue can only be used to decode subtype 0x00 for %s, got %v", bson.TypeBinary, byte(0xFF)),
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
					llvrwNothing,
					fmt.Errorf("ValueUnmarshalerDecodeValue can only handle types or pointers to types that are a ValueUnmarshaler, got %T", &wrong),
				},
				{
					"copy error",
					testValueUnmarshaler{},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString, err: errors.New("copy error"), errAfter: llvrwReadString},
					llvrwReadString,
					errors.New("copy error"),
				},
				{
					"ValueUnmarshaler",
					testValueUnmarshaler{t: bson.TypeString, val: llbson.AppendString(nil, "hello, world")},
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString, readval: string("hello, world")},
					llvrwReadString,
					nil,
				},
				{
					"nil pointer to ValueUnmarshaler",
					ptrPtrValueUnmarshaler,
					nil,
					&llValueReaderWriter{bsontype: bson.TypeString, readval: string("hello, world")},
					llvrwNothing,
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
					llvrw := new(llValueReaderWriter)
					if rc.llvrw != nil {
						llvrw = rc.llvrw
					}
					llvrw.t = t
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
	}

	t.Run("ValueUnmarshalerDecodeValue/UnmarshalBSONValue error", func(t *testing.T) {
		var dc DecodeContext
		llvrw := &llValueReaderWriter{bsontype: bson.TypeString, readval: string("hello, world!")}
		llvrw.t = t

		want := errors.New("ubsonv error")
		valUnmarshaler := &testValueUnmarshaler{err: want}
		got := dvd.ValueUnmarshalerDecodeValue(dc, llvrw, valUnmarshaler)
		if !compareErrors(got, want) {
			t.Errorf("Errors do not match. got %v; want %v", got, want)
		}
	})
	t.Run("ValueUnmarshalerDecodeValue/Pointer to ValueUnmarshaler", func(t *testing.T) {
		var dc DecodeContext
		llvrw := &llValueReaderWriter{bsontype: bson.TypeString, readval: string("hello, world!")}
		llvrw.t = t

		var want = new(*testValueUnmarshaler)
		var got = new(*testValueUnmarshaler)
		*want = &testValueUnmarshaler{t: bson.TypeString, val: llbson.AppendString(nil, "hello, world!")}
		*got = new(testValueUnmarshaler)

		err := dvd.ValueUnmarshalerDecodeValue(dc, llvrw, got)
		noerr(t, err)
		if !cmp.Equal(*got, *want) {
			t.Errorf("Unmarshaled values do not match. got %v; want %v", *got, *want)
		}
	})
	t.Run("ValueUnmarshalerDecodeValue/nil pointer inside non-nil pointer", func(t *testing.T) {
		var dc DecodeContext
		llvrw := &llValueReaderWriter{bsontype: bson.TypeString, readval: string("hello, world!")}
		llvrw.t = t

		var got = new(*testValueUnmarshaler)
		var want = new(*testValueUnmarshaler)
		*want = &testValueUnmarshaler{t: bson.TypeString, val: llbson.AppendString(nil, "hello, world!")}

		err := dvd.ValueUnmarshalerDecodeValue(dc, llvrw, got)
		noerr(t, err)
		if !cmp.Equal(got, want) {
			t.Errorf("Results do not match. got %v; want %v", got, want)
		}
	})
	t.Run("MapCodec/DecodeValue/non-settable", func(t *testing.T) {
		var dc DecodeContext
		llvrw := new(llValueReaderWriter)
		llvrw.t = t

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

	t.Run("CodeWithScopeCodec/DecodeValue/success", func(t *testing.T) {
		dc := DecodeContext{Registry: NewRegistryBuilder().Build()}
		dvr := newDocumentValueReader(bson.NewDocument(bson.EC.CodeWithScope("foo", "var hello = 'world';", bson.NewDocument(bson.EC.Null("bar")))))
		dr, err := dvr.ReadDocument()
		noerr(t, err)
		_, vr, err := dr.ReadElement()
		noerr(t, err)

		want := bson.CodeWithScope{
			Code:  "var hello = 'world';",
			Scope: bson.NewDocument(bson.EC.Null("bar")),
		}
		var got bson.CodeWithScope
		err = dvd.CodeWithScopeDecodeValue(dc, vr, &got)
		noerr(t, err)

		if !cmp.Equal(got, want) {
			t.Errorf("CodeWithScopes do not match. got %v; want %v", got, want)
		}
	})
	t.Run("DocumentDecodeValue", func(t *testing.T) {
		t.Run("CodecDecodeError", func(t *testing.T) {
			val := bool(true)
			want := ValueDecoderError{Name: "DocumentDecodeValue", Types: []interface{}{(**bson.Document)(nil)}, Received: val}
			got := dvd.DocumentDecodeValue(DecodeContext{}, &llValueReaderWriter{bsontype: bson.TypeEmbeddedDocument}, val)
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
				bsontype: bson.TypeEmbeddedDocument,
			}
			got := dvd.DocumentDecodeValue(DecodeContext{}, llvrw, new(*bson.Document))
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("decodeDocument errors", func(t *testing.T) {
			dc := DecodeContext{}
			err := errors.New("decodeDocument error")
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
				{"ReadDouble", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadDouble, bsontype: bson.TypeDouble}, err},
				{"ReadString", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadString, bsontype: bson.TypeString}, err},
				{
					"ReadDocument (Lookup)", DecodeContext{Registry: NewEmptyRegistryBuilder().Build()},
					&llValueReaderWriter{t: t, bsontype: bson.TypeEmbeddedDocument},
					ErrNoDecoder{Type: tDocument},
				},
				{
					"ReadArray (Lookup)", DecodeContext{Registry: NewEmptyRegistryBuilder().Build()},
					&llValueReaderWriter{t: t, bsontype: bson.TypeArray},
					ErrNoDecoder{Type: tArray},
				},
				{"ReadBinary", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadBinary, bsontype: bson.TypeBinary}, err},
				{"ReadUndefined", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadUndefined, bsontype: bson.TypeUndefined}, err},
				{"ReadObjectID", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadObjectID, bsontype: bson.TypeObjectID}, err},
				{"ReadBoolean", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadBoolean, bsontype: bson.TypeBoolean}, err},
				{"ReadDateTime", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadDateTime, bsontype: bson.TypeDateTime}, err},
				{"ReadNull", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadNull, bsontype: bson.TypeNull}, err},
				{"ReadRegex", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadRegex, bsontype: bson.TypeRegex}, err},
				{"ReadDBPointer", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadDBPointer, bsontype: bson.TypeDBPointer}, err},
				{"ReadJavascript", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadJavascript, bsontype: bson.TypeJavaScript}, err},
				{"ReadSymbol", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadSymbol, bsontype: bson.TypeSymbol}, err},
				{
					"ReadCodeWithScope (Lookup)", DecodeContext{Registry: NewEmptyRegistryBuilder().Build()},
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwReadCodeWithScope, bsontype: bson.TypeCodeWithScope},
					err,
				},
				{"ReadInt32", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadInt32, bsontype: bson.TypeInt32}, err},
				{"ReadInt64", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadInt64, bsontype: bson.TypeInt64}, err},
				{"ReadTimestamp", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadTimestamp, bsontype: bson.TypeTimestamp}, err},
				{"ReadDecimal128", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadDecimal128, bsontype: bson.TypeDecimal128}, err},
				{"ReadMinKey", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadMinKey, bsontype: bson.TypeMinKey}, err},
				{"ReadMaxKey", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadMaxKey, bsontype: bson.TypeMaxKey}, err},
				{"Invalid Type", dc, &llValueReaderWriter{t: t, bsontype: bson.Type(0)}, fmt.Errorf("Cannot read unknown BSON type %s", bson.Type(0))},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					err := dvd.decodeDocument(tc.dc, tc.llvrw, new(*bson.Document))
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
			var got *bson.Document
			dc := DecodeContext{Registry: NewRegistryBuilder().Build()}
			err := dvd.DocumentDecodeValue(dc, newDocumentValueReader(want), &got)
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
			want := ValueDecoderError{Name: "ArrayDecodeValue", Types: []interface{}{(**bson.Array)(nil)}, Received: val}
			got := dvd.ArrayDecodeValue(DecodeContext{}, &llValueReaderWriter{bsontype: bson.TypeArray}, val)
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
				bsontype: bson.TypeArray,
			}
			got := dvd.ArrayDecodeValue(DecodeContext{}, llvrw, new(*bson.Array))
			if !compareErrors(got, want) {
				t.Errorf("Errors do not match. got %v; want %v", got, want)
			}
		})
		t.Run("decode array errors", func(t *testing.T) {
			dc := DecodeContext{}
			err := errors.New("decode array error")
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
				{"ReadDouble", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadDouble, bsontype: bson.TypeDouble}, err},
				{"ReadString", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadString, bsontype: bson.TypeString}, err},
				{
					"ReadDocument (Lookup)", DecodeContext{Registry: NewEmptyRegistryBuilder().Build()},
					&llValueReaderWriter{t: t, bsontype: bson.TypeEmbeddedDocument},
					ErrNoDecoder{Type: tDocument},
				},
				{
					"ReadArray (Lookup)", DecodeContext{Registry: NewEmptyRegistryBuilder().Build()},
					&llValueReaderWriter{t: t, bsontype: bson.TypeArray},
					ErrNoDecoder{Type: tArray},
				},
				{"ReadBinary", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadBinary, bsontype: bson.TypeBinary}, err},
				{"ReadUndefined", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadUndefined, bsontype: bson.TypeUndefined}, err},
				{"ReadObjectID", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadObjectID, bsontype: bson.TypeObjectID}, err},
				{"ReadBoolean", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadBoolean, bsontype: bson.TypeBoolean}, err},
				{"ReadDateTime", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadDateTime, bsontype: bson.TypeDateTime}, err},
				{"ReadNull", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadNull, bsontype: bson.TypeNull}, err},
				{"ReadRegex", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadRegex, bsontype: bson.TypeRegex}, err},
				{"ReadDBPointer", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadDBPointer, bsontype: bson.TypeDBPointer}, err},
				{"ReadJavascript", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadJavascript, bsontype: bson.TypeJavaScript}, err},
				{"ReadSymbol", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadSymbol, bsontype: bson.TypeSymbol}, err},
				{
					"ReadCodeWithScope (Lookup)", DecodeContext{Registry: NewEmptyRegistryBuilder().Build()},
					&llValueReaderWriter{t: t, err: err, errAfter: llvrwReadCodeWithScope, bsontype: bson.TypeCodeWithScope},
					err,
				},
				{"ReadInt32", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadInt32, bsontype: bson.TypeInt32}, err},
				{"ReadInt64", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadInt64, bsontype: bson.TypeInt64}, err},
				{"ReadTimestamp", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadTimestamp, bsontype: bson.TypeTimestamp}, err},
				{"ReadDecimal128", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadDecimal128, bsontype: bson.TypeDecimal128}, err},
				{"ReadMinKey", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadMinKey, bsontype: bson.TypeMinKey}, err},
				{"ReadMaxKey", dc, &llValueReaderWriter{t: t, err: err, errAfter: llvrwReadMaxKey, bsontype: bson.TypeMaxKey}, err},
				{"Invalid Type", dc, &llValueReaderWriter{t: t, bsontype: bson.Type(0)}, fmt.Errorf("Cannot read unknown BSON type %s", bson.Type(0))},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					err := dvd.ArrayDecodeValue(tc.dc, tc.llvrw, new(*bson.Array))
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
			dc := DecodeContext{Registry: NewRegistryBuilder().Build()}

			dvr := newDocumentValueReader(bson.NewDocument(bson.EC.Array("", want)))
			dr, err := dvr.ReadDocument()
			noerr(t, err)
			_, vr, err := dr.ReadElement()
			noerr(t, err)

			var got *bson.Array
			err = dvd.ArrayDecodeValue(dc, vr, &got)
			noerr(t, err)
			if !got.Equal(want) {
				t.Error("Documents do not match")
				t.Errorf("\ngot :%v\nwant:%v", got, want)
			}
		})
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
		dvr := newDocumentValueReader(bson.NewDocument(bson.EC.ArrayFromElements("foo", bson.VC.String("foo"), bson.VC.String("bar"))))
		dr, err := dvr.ReadDocument()
		noerr(t, err)
		_, vr, err := dr.ReadElement()
		noerr(t, err)
		var val [1]string
		want := fmt.Errorf("more elements returned in array than can fit inside %T", val)

		dc := DecodeContext{Registry: NewRegistryBuilder().Build()}
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
					AE: &testValueUnmarshaler{t: bson.TypeString, val: llbson.AppendString(nil, "hello, world!")},
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
					bson.EC.String("ae", "hello, world!"),
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
					AE: []*testValueUnmarshaler{
						{t: bson.TypeString, val: llbson.AppendString(nil, "hello")},
						{t: bson.TypeString, val: llbson.AppendString(nil, "world")},
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

		t.Run("Decode", func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					vr := newValueReader(tc.b)
					dec, err := NewDecoder(NewRegistryBuilder().Build(), vr)
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

	t.Run("EmptyInterfaceDecodeValue", func(t *testing.T) {
		t.Run("DecodeValue", func(t *testing.T) {
			testCases := []struct {
				name     string
				val      interface{}
				bsontype bson.Type
			}{
				{
					"Double - float64",
					float64(3.14159),
					bson.TypeDouble,
				},
				{
					"String - string",
					string("foo bar baz"),
					bson.TypeString,
				},
				{
					"Embedded Document - *Document",
					bson.NewDocument(bson.EC.Null("foo")),
					bson.TypeEmbeddedDocument,
				},
				{
					"Array - *Array",
					bson.NewArray(bson.VC.Double(3.14159)),
					bson.TypeArray,
				},
				{
					"Binary - Binary",
					bson.Binary{Subtype: 0xFF, Data: []byte{0x01, 0x02, 0x03}},
					bson.TypeBinary,
				},
				{
					"Undefined - Undefined",
					bson.Undefinedv2{},
					bson.TypeUndefined,
				},
				{
					"ObjectID - objectid.ObjectID",
					objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					bson.TypeObjectID,
				},
				{
					"Boolean - bool",
					bool(true),
					bson.TypeBoolean,
				},
				{
					"DateTime - DateTime",
					bson.DateTime(1234567890),
					bson.TypeDateTime,
				},
				{
					"Null - Null",
					bson.Nullv2{},
					bson.TypeNull,
				},
				{
					"Regex - Regex",
					bson.Regex{Pattern: "foo", Options: "bar"},
					bson.TypeRegex,
				},
				{
					"DBPointer - DBPointer",
					bson.DBPointer{
						DB:      "foobar",
						Pointer: objectid.ObjectID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
					},
					bson.TypeDBPointer,
				},
				{
					"JavaScript - JavaScriptCode",
					bson.JavaScriptCode("var foo = 'bar';"),
					bson.TypeJavaScript,
				},
				{
					"Symbol - Symbol",
					bson.Symbol("foobarbazlolz"),
					bson.TypeSymbol,
				},
				{
					"CodeWithScope - CodeWithScope",
					bson.CodeWithScope{
						Code:  "var foo = 'bar';",
						Scope: bson.NewDocument(bson.EC.Double("foo", 3.14159)),
					},
					bson.TypeCodeWithScope,
				},
				{
					"Int32 - int32",
					int32(123456),
					bson.TypeInt32,
				},
				{
					"Int64 - int64",
					int64(1234567890),
					bson.TypeInt64,
				},
				{
					"Timestamp - Timestamp",
					bson.Timestamp{T: 12345, I: 67890},
					bson.TypeTimestamp,
				},
				{
					"Decimal128 - decimal.Decimal128",
					decimal.NewDecimal128(12345, 67890),
					bson.TypeDecimal128,
				},
				{
					"MinKey - MinKey",
					bson.MinKeyv2{},
					bson.TypeMinKey,
				},
				{
					"MaxKey - MaxKey",
					bson.MaxKeyv2{},
					bson.TypeMaxKey,
				},
			}
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					llvr := &llValueReaderWriter{bsontype: tc.bsontype}

					t.Run("Lookup failure", func(t *testing.T) {
						val := new(interface{})
						dc := DecodeContext{Registry: NewEmptyRegistryBuilder().Build()}
						want := ErrNoDecoder{Type: reflect.TypeOf(tc.val)}
						got := dvd.EmptyInterfaceDecodeValue(dc, llvr, val)
						if !compareErrors(got, want) {
							t.Errorf("Errors are not equal. got %v; want %v", got, want)
						}
					})

					t.Run("DecodeValue failure", func(t *testing.T) {
						want := errors.New("DecodeValue failure error")
						llc := &llCodec{t: t, err: want}
						dc := DecodeContext{
							Registry: NewEmptyRegistryBuilder().RegisterDecoder(reflect.TypeOf(tc.val), llc).Build(),
						}
						got := dvd.EmptyInterfaceDecodeValue(dc, llvr, new(interface{}))
						if !compareErrors(got, want) {
							t.Errorf("Errors are not equal. got %v; want %v", got, want)
						}
					})

					t.Run("Success", func(t *testing.T) {
						want := tc.val
						llc := &llCodec{t: t, decodeval: tc.val}
						dc := DecodeContext{
							Registry: NewEmptyRegistryBuilder().RegisterDecoder(reflect.TypeOf(tc.val), llc).Build(),
						}
						got := new(interface{})
						err := dvd.EmptyInterfaceDecodeValue(dc, llvr, got)
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
			got := dvd.EmptyInterfaceDecodeValue(DecodeContext{}, nil, &val)
			if !compareErrors(got, want) {
				t.Errorf("Errors are not equal. got %v; want %v", got, want)
			}
		})

		t.Run("nil *interface{}", func(t *testing.T) {
			var val *interface{}
			want := fmt.Errorf("EmptyInterfaceDecodeValue can only be used to decode non-nil *interface{} values, provided type if %T", val)
			got := dvd.EmptyInterfaceDecodeValue(DecodeContext{}, nil, val)
			if !compareErrors(got, want) {
				t.Errorf("Errors are not equal. got %v; want %v", got, want)
			}
		})

		t.Run("unknown BSON type", func(t *testing.T) {
			llvr := &llValueReaderWriter{bsontype: bson.Type(0)}
			want := fmt.Errorf("Type %s is not a valid BSON type and has no default Go type to decode into", bson.Type(0))
			got := dvd.EmptyInterfaceDecodeValue(DecodeContext{}, llvr, new(interface{}))
			if !compareErrors(got, want) {
				t.Errorf("Errors are not equal. got %v; want %v", got, want)
			}
		})
	})

}

type testValueUnmarshaler struct {
	t   bson.Type
	val []byte
	err error
}

func (tvu *testValueUnmarshaler) UnmarshalBSONValue(t bson.Type, val []byte) error {
	tvu.t, tvu.val = t, val
	return tvu.err
}
func (tvu testValueUnmarshaler) Equal(tvu2 testValueUnmarshaler) bool {
	return tvu.t == tvu2.t && bytes.Equal(tvu.val, tvu2.val)
}
