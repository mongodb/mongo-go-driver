package bsoncodec

import (
	"fmt"
	"reflect"

	"github.com/mongodb/mongo-go-driver/bson"
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
	case bson.TypeDouble:
		val = new(float64)
		rtype = tFloat64
		fn = func() { *target = *(val.(*float64)) }
	case bson.TypeString:
		val = new(string)
		rtype = tString
		fn = func() { *target = *(val.(*string)) }
	case bson.TypeEmbeddedDocument:
		val = bson.NewDocument()
		rtype = tDocument
		fn = func() { *target = val.(*bson.Document) }
	case bson.TypeArray:
		val = bson.NewArray()
		rtype = tArray
		fn = func() { *target = val.(*bson.Array) }
	case bson.TypeBinary:
		val = new(bson.Binary)
		rtype = tBinary
		fn = func() { *target = *(val.(*bson.Binary)) }
	case bson.TypeUndefined:
		val = new(bson.Undefinedv2)
		rtype = tUndefined
		fn = func() { *target = *(val.(*bson.Undefinedv2)) }
	case bson.TypeObjectID:
		val = new(objectid.ObjectID)
		rtype = tOID
		fn = func() { *target = *(val.(*objectid.ObjectID)) }
	case bson.TypeBoolean:
		val = new(bool)
		rtype = tBool
		fn = func() { *target = *(val.(*bool)) }
	case bson.TypeDateTime:
		val = new(bson.DateTime)
		rtype = tDateTime
		fn = func() { *target = *(val.(*bson.DateTime)) }
	case bson.TypeNull:
		val = new(bson.Nullv2)
		rtype = tNull
		fn = func() { *target = *(val.(*bson.Nullv2)) }
	case bson.TypeRegex:
		val = new(bson.Regex)
		rtype = tRegex
		fn = func() { *target = *(val.(*bson.Regex)) }
	case bson.TypeDBPointer:
		val = new(bson.DBPointer)
		rtype = tDBPointer
		fn = func() { *target = *(val.(*bson.DBPointer)) }
	case bson.TypeJavaScript:
		val = new(bson.JavaScriptCode)
		rtype = tJavaScriptCode
		fn = func() { *target = *(val.(*bson.JavaScriptCode)) }
	case bson.TypeSymbol:
		val = new(bson.Symbol)
		rtype = tSymbol
		fn = func() { *target = *(val.(*bson.Symbol)) }
	case bson.TypeCodeWithScope:
		val = new(bson.CodeWithScope)
		rtype = tCodeWithScope
		fn = func() { *target = *(val.(*bson.CodeWithScope)) }
	case bson.TypeInt32:
		val = new(int32)
		rtype = tInt32
		fn = func() { *target = *(val.(*int32)) }
	case bson.TypeInt64:
		val = new(int64)
		rtype = tInt64
		fn = func() { *target = *(val.(*int64)) }
	case bson.TypeTimestamp:
		val = new(bson.Timestamp)
		rtype = tTimestamp
		fn = func() { *target = *(val.(*bson.Timestamp)) }
	case bson.TypeDecimal128:
		val = new(decimal.Decimal128)
		rtype = tDecimal
		fn = func() { *target = *(val.(*decimal.Decimal128)) }
	case bson.TypeMinKey:
		val = new(bson.MinKeyv2)
		rtype = tMinKey
		fn = func() { *target = *(val.(*bson.MinKeyv2)) }
	case bson.TypeMaxKey:
		val = new(bson.MaxKeyv2)
		rtype = tMaxKey
		fn = func() { *target = *(val.(*bson.MaxKeyv2)) }
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
