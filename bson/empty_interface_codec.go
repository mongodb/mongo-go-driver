package bson

import (
	"fmt"
	"reflect"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

var defaultEmptyInterfaceCodec = &EmptyInterfaceCodec{}

// EmptyInterfaceCodec is the Codec used for empty interface (interface{})
// values.
type EmptyInterfaceCodec struct{}

var _ Codec = &EmptyInterfaceCodec{}

// EncodeValue implements the Codec interface.
func (eic *EmptyInterfaceCodec) EncodeValue(ec EncodeContext, vw ValueWriter, i interface{}) error {
	codec, err := ec.Lookup(reflect.TypeOf(i))
	if err != nil {
		return err
	}

	return codec.EncodeValue(ec, vw, i)
}

// DecodeValue implements the Codec interface.
func (eic *EmptyInterfaceCodec) DecodeValue(dc DecodeContext, vr ValueReader, i interface{}) error {
	target, ok := i.(*interface{})
	if !ok || target == nil {
		return fmt.Errorf("%T can only be used to decode non-nil *interface{} values, provided type if %T", eic, i)
	}

	// fn is a function we call to assign val back to the target, we do this so
	// we can keep down on the repeated code in this method. In all of the
	// implementations this is a closure, so we don't need to provide the
	// target as a parameter.
	var fn func()
	var val interface{}
	var rtype reflect.Type

	switch vr.Type() {
	case TypeDouble:
		val = new(float64)
		rtype = tFloat64
		fn = func() { *target = *(val.(*float64)) }
	case TypeString:
		val = new(string)
		rtype = tString
		fn = func() { *target = *(val.(*string)) }
	case TypeEmbeddedDocument:
		val = NewDocument()
		rtype = tDocument
		fn = func() { *target = val.(*Document) }
	case TypeArray:
		val = NewArray()
		rtype = tArray
		fn = func() { *target = val.(*Array) }
	case TypeBinary:
		val = new(Binary)
		rtype = tBinary
		fn = func() { *target = *(val.(*Binary)) }
	case TypeUndefined:
		val = new(Undefinedv2)
		rtype = tUndefined
		fn = func() { *target = *(val.(*Undefinedv2)) }
	case TypeObjectID:
		val = new(objectid.ObjectID)
		rtype = tOID
		fn = func() { *target = *(val.(*objectid.ObjectID)) }
	case TypeBoolean:
		val = new(bool)
		rtype = tBool
		fn = func() { *target = *(val.(*bool)) }
	case TypeDateTime:
		val = new(DateTime)
		rtype = tDateTime
		fn = func() { *target = *(val.(*DateTime)) }
	case TypeNull:
		val = new(Nullv2)
		rtype = tNull
		fn = func() { *target = *(val.(*Nullv2)) }
	case TypeRegex:
		val = new(Regex)
		rtype = tRegex
		fn = func() { *target = *(val.(*Regex)) }
	case TypeDBPointer:
		val = new(DBPointer)
		rtype = tDBPointer
		fn = func() { *target = *(val.(*DBPointer)) }
	case TypeJavaScript:
		val = new(JavaScriptCode)
		rtype = tJavaScriptCode
		fn = func() { *target = *(val.(*JavaScriptCode)) }
	case TypeSymbol:
		val = new(Symbol)
		rtype = tSymbol
		fn = func() { *target = *(val.(*Symbol)) }
	case TypeCodeWithScope:
		val = new(CodeWithScope)
		rtype = tCodeWithScope
		fn = func() { *target = *(val.(*CodeWithScope)) }
	case TypeInt32:
		val = new(int32)
		rtype = tInt32
		fn = func() { *target = *(val.(*int32)) }
	case TypeInt64:
		val = new(int64)
		rtype = tInt64
		fn = func() { *target = *(val.(*int64)) }
	case TypeTimestamp:
		val = new(Timestamp)
		rtype = tTimestamp
		fn = func() { *target = *(val.(*Timestamp)) }
	case TypeDecimal128:
		val = new(decimal.Decimal128)
		rtype = tDecimal
		fn = func() { *target = *(val.(*decimal.Decimal128)) }
	case TypeMinKey:
		val = new(MinKeyv2)
		rtype = tMinKey
		fn = func() { *target = *(val.(*MinKeyv2)) }
	case TypeMaxKey:
		val = new(MaxKeyv2)
		rtype = tMaxKey
		fn = func() { *target = *(val.(*MaxKeyv2)) }
	default:
		return fmt.Errorf("Type %s is not a valid BSON type and has no default Go type to decode into", vr.Type())
	}

	codec, err := dc.Lookup(rtype)
	if err != nil {
		return err
	}
	err = codec.DecodeValue(dc, vr, val)
	if err != nil {
		return err
	}

	fn()
	return nil
}
