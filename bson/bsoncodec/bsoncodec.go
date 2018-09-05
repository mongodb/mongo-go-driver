package bsoncodec

import (
	"errors"
	"fmt"
	"math"

	"github.com/mongodb/mongo-go-driver/bson"
)

// ConstructElement will attempt to turn the provided key and value into an Element.
// For common types, type casting is used, if the type is more complex, such as
// a map or struct, reflection is used. If the value cannot be converted either
// by typecasting or through reflection, a null Element is constructed with the
// key. This method will never return a nil *Element. If an error turning the
// value into an Element is desired, use the InterfaceErr method.
func ConstructElement(key string, value interface{}) *bson.Element {
	var elem *bson.Element
	switch t := value.(type) {
	case bool:
		elem = bson.EC.Boolean(key, t)
	case int8:
		elem = bson.EC.Int32(key, int32(t))
	case int16:
		elem = bson.EC.Int32(key, int32(t))
	case int32:
		elem = bson.EC.Int32(key, int32(t))
	case int:
		if t < math.MaxInt32 {
			elem = bson.EC.Int32(key, int32(t))
		}
		elem = bson.EC.Int64(key, int64(t))
	case int64:
		if t < math.MaxInt32 {
			elem = bson.EC.Int32(key, int32(t))
		}
		elem = bson.EC.Int64(key, int64(t))
	case uint8:
		elem = bson.EC.Int32(key, int32(t))
	case uint16:
		elem = bson.EC.Int32(key, int32(t))
	case uint:
		switch {
		case t < math.MaxInt32:
			elem = bson.EC.Int32(key, int32(t))
		case uint64(t) > math.MaxInt64:
			elem = bson.EC.Null(key)
		default:
			elem = bson.EC.Int64(key, int64(t))
		}
	case uint32:
		if t < math.MaxInt32 {
			elem = bson.EC.Int32(key, int32(t))
		}
		elem = bson.EC.Int64(key, int64(t))
	case uint64:
		switch {
		case t < math.MaxInt32:
			elem = bson.EC.Int32(key, int32(t))
		case t > math.MaxInt64:
			elem = bson.EC.Null(key)
		default:
			elem = bson.EC.Int64(key, int64(t))
		}
	case float32:
		elem = bson.EC.Double(key, float64(t))
	case float64:
		elem = bson.EC.Double(key, t)
	case string:
		elem = bson.EC.String(key, t)
	case *bson.Element:
		elem = t
	case *bson.Document:
		elem = bson.EC.SubDocument(key, t)
	case bson.Reader:
		elem = bson.EC.SubDocumentFromReader(key, t)
	case *bson.Value:
		elem = bson.EC.FromValue(key, t)
		if elem == nil {
			elem = bson.EC.Null(key)
		}
	default:
		// TODO(skriptble): Allow users to provide registry
		// TODO(skriptble): Use a pool of []byte
		buf, err := marshalElement(defaultRegistry, nil, key, t)
		if err != nil {
			elem = bson.EC.Null(key)
		}
		elem, err = bson.EC.FromBytesErr(buf)
		if err != nil {
			elem = bson.EC.Null(key)
		}
	}

	return elem
}

// ConstructElementErr does the same thing as ConstructElement but returns an
// error instead of returning a BSON Null element.
func ConstructElementErr(key string, value interface{}) (*bson.Element, error) {
	var elem *bson.Element
	var err error
	switch t := value.(type) {
	case bool, int8, int16, int32, int, int64, uint8, uint16,
		uint32, float32, float64, string, *bson.Element, *bson.Document, bson.Reader:
		elem = ConstructElement(key, value)
	case uint:
		switch {
		case t < math.MaxInt32:
			elem = bson.EC.Int32(key, int32(t))
		case uint64(t) > math.MaxInt64:
			err = fmt.Errorf("BSON only has signed integer types and %d overflows an int64", t)
		default:
			elem = bson.EC.Int64(key, int64(t))
		}
	case uint64:
		switch {
		case t < math.MaxInt32:
			elem = bson.EC.Int32(key, int32(t))
		case uint64(t) > math.MaxInt64:
			err = fmt.Errorf("BSON only has signed integer types and %d overflows an int64", t)
		default:
			elem = bson.EC.Int64(key, int64(t))
		}
	case *bson.Value:
		elem = bson.EC.FromValue(key, t)
		if elem == nil {
			err = errors.New("invalid *Value provided, cannot convert to *Element")
		}
	default:
		// TODO(skriptble): Allow users to provide registry
		// TODO(skriptble): Use a pool of []byte
		var buf []byte
		buf, err = marshalElement(defaultRegistry, nil, key, t)
		if err != nil {
			break
		}
		elem, err = bson.EC.FromBytesErr(buf)
	}

	if err != nil {
		return nil, err
	}

	return elem, nil
}
