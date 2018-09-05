package bsoncodec

import (
	"fmt"
	"reflect"
	"strings"
)

var ptBool = reflect.TypeOf((*bool)(nil))
var ptInt8 = reflect.TypeOf((*int8)(nil))
var ptInt16 = reflect.TypeOf((*int16)(nil))
var ptInt32 = reflect.TypeOf((*int32)(nil))
var ptInt64 = reflect.TypeOf((*int64)(nil))
var ptInt = reflect.TypeOf((*int)(nil))
var ptUint8 = reflect.TypeOf((*uint8)(nil))
var ptUint16 = reflect.TypeOf((*uint16)(nil))
var ptUint32 = reflect.TypeOf((*uint32)(nil))
var ptUint64 = reflect.TypeOf((*uint64)(nil))
var ptUint = reflect.TypeOf((*uint)(nil))
var ptFloat32 = reflect.TypeOf((*float32)(nil))
var ptFloat64 = reflect.TypeOf((*float64)(nil))
var ptString = reflect.TypeOf((*string)(nil))

// ValueEncoderError is an error returned from a ValueEncoder when the provided
// value can't be encoded by the ValueEncoder.
type ValueEncoderError struct {
	Name     string
	Types    []interface{}
	Received interface{}
}

func (vee ValueEncoderError) Error() string {
	types := make([]string, 0, len(vee.Types))
	for _, t := range vee.Types {
		types = append(types, fmt.Sprintf("%T", t))
	}
	return fmt.Sprintf("%s can only process %s, but got a %T", vee.Name, strings.Join(types, ", "), vee.Received)
}

// ValueDecoderError is an error returned from a ValueDecoder when the provided
// value can't be decoded by the ValueDecoder.
type ValueDecoderError struct {
	Name     string
	Types    []interface{}
	Received interface{}
}

func (vde ValueDecoderError) Error() string {
	types := make([]string, 0, len(vde.Types))
	for _, t := range vde.Types {
		types = append(types, fmt.Sprintf("%T", t))
	}
	return fmt.Sprintf("%s can only process %s, but got a %T", vde.Name, strings.Join(types, ", "), vde.Received)
}

// EncodeContext is the contextual information required for a Codec to encode a
// value.
type EncodeContext struct {
	*Registry
	MinSize bool
}

// DecodeContext is the contextual information required for a Codec to decode a
// value.
type DecodeContext struct {
	*Registry
	Truncate bool
}

// ValueCodec is the interface that groups the methods to encode and decode
// values.
type ValueCodec interface {
	ValueEncoder
	ValueDecoder
}

// ValueEncoder is the interface implemented by types that can handle the
// encoding of a value. Implementations must handle both values and
// pointers to values.
type ValueEncoder interface {
	EncodeValue(EncodeContext, ValueWriter, interface{}) error
}

// ValueEncoderFunc is an adapter function that allows a function with the
// correct signature to be used as a ValueEncoder.
type ValueEncoderFunc func(EncodeContext, ValueWriter, interface{}) error

// EncodeValue implements the ValueEncoder interface.
func (fn ValueEncoderFunc) EncodeValue(ec EncodeContext, vw ValueWriter, val interface{}) error {
	return fn(ec, vw, val)
}

// ValueDecoder is the interface implemented by types that can handle the
// decoding of a value. Implementations must handle pointers to values,
// including pointers to pointer values. The implementation may create a new
// value and assign it to the pointer if necessary.
type ValueDecoder interface {
	DecodeValue(DecodeContext, ValueReader, interface{}) error
}

// ValueDecoderFunc is an adapter function that allows a function with the
// correct signature to be used as a ValueDecoder.
type ValueDecoderFunc func(DecodeContext, ValueReader, interface{}) error

// DecodeValue implements the ValueDecoder interface.
func (fn ValueDecoderFunc) DecodeValue(dc DecodeContext, vr ValueReader, val interface{}) error {
	return fn(dc, vr, val)
}

// CodecZeroer is the interface implemented by Codecs that can also determine if
// a value of the type that would be encoded is zero.
type CodecZeroer interface {
	IsTypeZero(interface{}) bool
}
