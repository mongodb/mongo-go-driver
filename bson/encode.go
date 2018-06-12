// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// ErrEncoderNilWriter indicates that encoder.Encode was called with a nil argument.
var ErrEncoderNilWriter = errors.New("encoder.Encode called on Encoder with nil io.Writer")

var tByteSlice = reflect.TypeOf(([]byte)(nil))
var tByte = reflect.TypeOf(byte(0x00))
var tElement = reflect.TypeOf((*Element)(nil))
var tURL = reflect.TypeOf(url.URL{})
var tJSONNumber = reflect.TypeOf(json.Number(""))

// Marshaler describes a type that can marshal a BSON representation of itself into bytes.
type Marshaler interface {
	MarshalBSON() ([]byte, error)
}

// DocumentMarshaler describes a type that can marshal itself into a bson.Document.
type DocumentMarshaler interface {
	MarshalBSONDocument() (*Document, error)
}

// ElementMarshaler describes a type that can marshal itself into a bson.Element.
type ElementMarshaler interface {
	MarshalBSONElement() (*Element, error)
}

// ValueMarshaler describes a type that can marshal itself into a bson.Value.
type ValueMarshaler interface {
	MarshalBSONValue() (*Value, error)
}

// Encoder describes a type that can encode itself into a value.
type Encoder interface {
	// Encode encodes a value from an io.Writer into the given value.
	//
	// The value can be any one of the following types:
	//
	//   - bson.Marshaler
	//   - io.Reader
	//   - []byte
	//   - bson.Reader
	//   - any map with string keys
	//   - a struct (possibly with tags)
	//
	// In the case of a struct, the lowercased field name is used as the key for each exported
	// field but this behavior may be changed using a struct tag. The tag may also contain flags to
	// adjust the marshalling behavior for the field. The tag formats accepted are:
	//
	//     "[<key>][,<flag1>[,<flag2>]]"
	//
	//     `(...) bson:"[<key>][,<flag1>[,<flag2>]]" (...)`
	//
	// The following flags are currently supported:
	//
	//     omitempty  Only include the field if it's not set to the zero value for the type or to
	//                empty slices or maps.
	//
	//     minsize    Marshal an integer of a type larger than 32 bits value as an int32, if that's
	//                feasible while preserving the numeric value.
	//
	//     inline     Inline the field, which must be a struct or a map, causing all of its fields
	//                or keys to be processed as if they were part of the outer struct.
	//
	// An example:
	//
	//     type T struct {
	//         A bool
	//         B int    "myb"
	//         C string "myc,omitempty"
	//         D string `bson:",omitempty" json:"jsonkey"`
	//         E int64  ",minsize"
	//         F int64  "myf,omitempty,minsize"
	//     }
	Encode(interface{}) error
}

// DocumentEncoder describes a type that can marshal itself into a value and return the bson.Document it represents.
type DocumentEncoder interface {
	EncodeDocument(interface{}) (*Document, error)
}

// Zeroer allows custom struct types to implement a report of zero
// state. All struct types that don't implement Zeroer or where IsZero
// returns false are considered to be not zero.
type Zeroer interface {
	IsZero() bool
}

type encoder struct {
	w io.Writer
}

// NewEncoder creates an encoder that writes to w.
func NewEncoder(w io.Writer) Encoder {
	return &encoder{w: w}
}

// NewDocumentEncoder creates an encoder that encodes into a *Document.
func NewDocumentEncoder() DocumentEncoder {
	return &encoder{}
}

func convertTimeToInt64(t time.Time) int64 {
	return t.Unix()*1000 + int64(t.Nanosecond()/1e6)
}

func (e *encoder) Encode(v interface{}) error {
	var err error

	if e.w == nil {
		return ErrEncoderNilWriter
	}

	switch t := v.(type) {
	case Marshaler:
		var b []byte
		b, err = t.MarshalBSON()
		if err != nil {
			return err
		}
		_, err = Reader(b).Validate()
		if err != nil {
			return err
		}
		_, err = e.w.Write(b)
		return err
	case io.Reader:
		var r Reader
		r, err = NewFromIOReader(t)
		if err != nil {
			return err
		}

		_, err = r.Validate()
		if err != nil {
			return err
		}
		_, err = e.w.Write(r)
	case []byte:
		_, err = Reader(t).Validate()
		if err != nil {
			return err
		}
		_, err = e.w.Write(t)

	case Reader:
		_, err = t.Validate()
		if err != nil {
			return err
		}
		_, err = e.w.Write(t)
	default:
		var elems []*Element
		rval := reflect.ValueOf(v)
		elems, err = e.reflectEncode(rval)
		if err != nil {
			return err
		}
		_, err = NewDocument(elems...).WriteTo(e.w)
	}

	return err
}

// EncodeDocument encodes a value from an io.Writer into the given value and returns the document
// it represents.
//
// EncodeDocument accepts the same types as Encoder.Encode.
func (e *encoder) EncodeDocument(v interface{}) (*Document, error) {
	var err error

	d := NewDocument()

	switch t := v.(type) {
	case *Document:
		err = d.Concat(t)
	case Marshaler:
		var b []byte
		b, err = t.MarshalBSON()
		if err != nil {
			return nil, err
		}
		_, err = Reader(b).Validate()
		if err != nil {
			return nil, err
		}
		err = d.Concat(b)
	case io.Reader:
		var r Reader
		r, err = NewFromIOReader(t)
		if err != nil {
			return nil, err
		}

		_, err = r.Validate()
		if err != nil {
			return nil, err
		}
		err = d.Concat(r)
	case []byte:
		_, err = Reader(t).Validate()
		if err != nil {
			return nil, err
		}
		err = d.Concat(t)
	case Reader:
		_, err = t.Validate()
		if err != nil {
			return nil, err
		}
		err = d.Concat(t)
	default:
		var elems []*Element
		rval := reflect.ValueOf(v)
		elems, err = e.reflectEncode(rval)
		if err != nil {
			return nil, err
		}
		d.Append(elems...)
	}

	if err != nil {
		return nil, err
	}

	return d, nil
}

// underlyingVal will unwrap the given reflect.Value until it is not a pointer
// nor an interface.
func (e *encoder) underlyingVal(val reflect.Value) reflect.Value {
	if val.Kind() != reflect.Ptr && val.Kind() != reflect.Interface {
		return val
	}
	if val.IsNil() {
		return val
	}
	return e.underlyingVal(val.Elem())
}

func (e *encoder) reflectEncode(val reflect.Value) ([]*Element, error) {
	val = e.underlyingVal(val)

	var elems []*Element
	var err error
	switch val.Kind() {
	case reflect.Map:
		elems, err = e.encodeMap(val)
	case reflect.Slice, reflect.Array:
		elems, err = e.encodeSlice(val)
	case reflect.Struct:
		elems, err = e.encodeStruct(val)
	default:
		err = fmt.Errorf("Cannot encode type %s as a BSON Document", val.Type())
	}

	if err != nil {
		return nil, err
	}

	return elems, nil
}

func (e *encoder) encodeMap(val reflect.Value) ([]*Element, error) {
	mapkeys := val.MapKeys()
	elems := make([]*Element, 0, val.Len())
	for _, rkey := range mapkeys {
		orig := rkey
		rkey = e.underlyingVal(rkey)

		var key string
		switch rkey.Kind() {
		case reflect.Bool:
			key = strconv.FormatBool(rkey.Bool())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			key = strconv.FormatInt(rkey.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			key = strconv.FormatUint(rkey.Uint(), 10)
		case reflect.Float32:
			key = strconv.FormatFloat(rkey.Float(), 'g', -1, 32)
		case reflect.Float64:
			key = strconv.FormatFloat(rkey.Float(), 'g', -1, 64)
		case reflect.Complex64, reflect.Complex128:
			key = fmt.Sprintf("%g", rkey.Complex())
		case reflect.String:
			key = rkey.String()
		default:
			switch rkey.Type() {
			case tOID:
				key = fmt.Sprintf("%s", rkey.Interface())
			case tURL:
				rkey = orig
				key = fmt.Sprintf("%s", rkey.Interface())
			case tDecimal:
				key = fmt.Sprintf("%s", rkey.Interface())
			default:
				return nil, fmt.Errorf("Unsupported map key type %s", rkey.Kind())
			}
		}

		rval := val.MapIndex(rkey)

		switch t := rval.Interface().(type) {
		case *Element:
			elems = append(elems, t)
			continue
		case *Document:
			elems = append(elems, EC.SubDocument(key, t))
			continue
		case Reader:
			elems = append(elems, EC.SubDocumentFromReader(key, t))
			continue
		case json.Number:
			// We try to do an int first
			if i64, err := t.Int64(); err == nil {
				elems = append(elems, EC.Int64(key, i64))
				continue
			}
			f64, err := t.Float64()
			if err != nil {
				return nil, fmt.Errorf("Invalid json.Number used as map value: %s", err)
			}
			elems = append(elems, EC.Double(key, f64))
			continue
		case *url.URL:
			elems = append(elems, EC.String(key, t.String()))
			continue
		case decimal.Decimal128:
			elems = append(elems, EC.Decimal128(key, t))
			continue
		}
		rval = e.underlyingVal(rval)

		elem, err := e.elemFromValue(key, rval, false)
		if err != nil {
			return nil, err
		}
		elems = append(elems, elem)
	}
	return elems, nil
}

func (e *encoder) encodeSlice(val reflect.Value) ([]*Element, error) {
	elems := make([]*Element, 0, val.Len())
	for i := 0; i < val.Len(); i++ {
		sval := val.Index(i)
		key := strconv.Itoa(i)
		switch t := sval.Interface().(type) {
		case *Element:
			elems = append(elems, t)
			continue
		case *Document:
			elems = append(elems, EC.SubDocument(key, t))
			continue
		case Reader:
			elems = append(elems, EC.SubDocumentFromReader(key, t))
			continue
		case json.Number:
			// We try to do an int first
			if i64, err := t.Int64(); err == nil {
				elems = append(elems, EC.Int64(key, i64))
				continue
			}
			f64, err := t.Float64()
			if err != nil {
				return nil, fmt.Errorf("Invalid json.Number used as map value: %s", err)
			}
			elems = append(elems, EC.Double(key, f64))
			continue
		case *url.URL:
			elems = append(elems, EC.String(key, t.String()))
			continue
		case decimal.Decimal128:
			elems = append(elems, EC.Decimal128(key, t))
			continue
		}
		sval = e.underlyingVal(sval)
		elem, err := e.elemFromValue(key, sval, false)
		if err != nil {
			return nil, err
		}
		elems = append(elems, elem)
	}
	return elems, nil
}

func (e *encoder) encodeSliceAsArray(rval reflect.Value, minsize bool) ([]*Value, error) {
	vals := make([]*Value, 0, rval.Len())
	for i := 0; i < rval.Len(); i++ {
		sval := rval.Index(i)
		switch t := sval.Interface().(type) {
		case *Element:
			vals = append(vals, t.value)
			continue
		case *Value:
			vals = append(vals, t)
			continue
		case *Document:
			vals = append(vals, VC.Document(t))
			continue
		case Reader:
			vals = append(vals, VC.DocumentFromReader(t))
			continue
		case json.Number:
			// We try to do an int first
			if i64, err := t.Int64(); err == nil {
				vals = append(vals, VC.Int64(i64))
				continue
			}
			f64, err := t.Float64()
			if err != nil {
				return nil, fmt.Errorf("Invalid json.Number used as map value: %s", err)
			}
			vals = append(vals, VC.Double(f64))
			continue
		case *url.URL:
			vals = append(vals, VC.String(t.String()))
			continue
		case decimal.Decimal128:
			vals = append(vals, VC.Decimal128(t))
			continue
		case time.Time:
			vals = append(vals, VC.DateTime(convertTimeToInt64(t)))
			continue
		case *time.Time:
			if t == nil {
				vals = append(vals, VC.Null())
			} else {
				vals = append(vals, VC.DateTime(convertTimeToInt64(*t)))
			}
			continue
		}

		sval = e.underlyingVal(sval)
		val, err := e.valueFromValue(sval, minsize)
		if err != nil {
			return nil, err
		}
		vals = append(vals, val)
	}
	return vals, nil
}

func (e *encoder) encodeStruct(val reflect.Value) ([]*Element, error) {
	elems := make([]*Element, 0, val.NumField())
	sType := val.Type()

	for i := 0; i < val.NumField(); i++ {
		sf := sType.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		key := strings.ToLower(sf.Name)
		tag, ok := sf.Tag.Lookup("bson")
		var omitempty, minsize, inline = false, false, false
		switch {
		case ok:
			if tag == "-" {
				continue
			}
			for idx, str := range strings.Split(tag, ",") {
				if idx == 0 && str != "" {
					key = str
				}
				switch str {
				case "omitempty":
					omitempty = true
				case "minsize":
					minsize = true
				case "inline":
					inline = true
				}
			}
		case !ok && !strings.Contains(string(sf.Tag), ":") && len(sf.Tag) > 0:
			key = string(sf.Tag)
		}

		field := val.Field(i)

		if omitempty && e.isZero(field) {
			continue
		}

		switch t := field.Interface().(type) {
		case *Element:
			elems = append(elems, t)
			continue
		case *Document:
			elems = append(elems, EC.SubDocument(key, t))
			continue
		case Reader:
			elems = append(elems, EC.SubDocumentFromReader(key, t))
			continue
		case json.Number:
			// We try to do an int first
			if i64, err := t.Int64(); err == nil {
				elems = append(elems, EC.Int64(key, i64))
				continue
			}
			f64, err := t.Float64()
			if err != nil {
				return nil, fmt.Errorf("Invalid json.Number used as map value: %s", err)
			}
			elems = append(elems, EC.Double(key, f64))
			continue
		case *url.URL:
			elems = append(elems, EC.String(key, t.String()))
			continue
		case decimal.Decimal128:
			elems = append(elems, EC.Decimal128(key, t))
			continue
		case time.Time:
			elems = append(elems, EC.DateTime(key, convertTimeToInt64(t)))
			continue
		case *time.Time:
			if t == nil {
				elems = append(elems, EC.Null(key))
			} else {
				elems = append(elems, EC.DateTime(key, convertTimeToInt64(*t)))
			}
			continue
		}
		field = e.underlyingVal(field)

		if inline {
			switch sf.Type.Kind() {
			case reflect.Map:
				melems, err := e.encodeMap(field)
				if err != nil {
					return nil, err
				}
				elems = append(elems, melems...)
				continue
			case reflect.Struct:
				selems, err := e.encodeStruct(field)
				if err != nil {
					return nil, err
				}
				elems = append(elems, selems...)
				continue
			default:
				return nil, errors.New("inline is only supported for map and struct types")
			}
		}

		elem, err := e.elemFromValue(key, field, minsize)
		if err != nil {
			return nil, err
		}
		elems = append(elems, elem)
	}
	return elems, nil
}

func (e *encoder) isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Struct:
		if z, ok := v.Interface().(Zeroer); ok {
			return z.IsZero()
		}
		return false
	}

	return false
}

func (e *encoder) elemFromValue(key string, val reflect.Value, minsize bool) (*Element, error) {
	var elem *Element
	switch val.Kind() {
	case reflect.Interface, reflect.Ptr:
		if !val.IsNil() {
			return nil, errors.New("Values must be unwrapped when calling elemFromValue, try calling underlyingVal first")
		}
		elem = EC.Null(key)
	case reflect.Bool:
		elem = EC.Boolean(key, val.Bool())
	case reflect.Int8, reflect.Int16, reflect.Int32:
		elem = EC.Int32(key, int32(val.Int()))
	case reflect.Int, reflect.Int64:
		i := val.Int()
		if minsize && i < math.MaxInt32 {
			elem = EC.Int32(key, int32(val.Int()))
			break
		}
		elem = EC.Int64(key, val.Int())
	case reflect.Uint8, reflect.Uint16:
		i := val.Uint()
		elem = EC.Int32(key, int32(i))
	case reflect.Uint, reflect.Uint32, reflect.Uint64:
		i := val.Uint()
		switch {
		case i < math.MaxInt32 && minsize:
			elem = EC.Int32(key, int32(i))
		case i < math.MaxInt64:
			elem = EC.Int64(key, int64(i))
		default:
			return nil, fmt.Errorf("BSON only has signed integer types and %d overflows an int64", i)
		}
	case reflect.Float32, reflect.Float64:
		elem = EC.Double(key, val.Float())
	case reflect.String:
		elem = EC.String(key, val.String())
	case reflect.Map:
		// We specifically check if the value is nil so we can properly round trip.
		// If we didn't do this, we couldn't differentiate between an empty map, which should
		// be a document, and a nil map, which should be null. In Go, there is a difference
		// between the empty value and nil, we should preserve that.
		if val.IsNil() {
			elem = EC.Null(key)
			break
		}
		mapElems, err := e.encodeMap(val)
		if err != nil {
			return nil, err
		}
		elem = EC.SubDocumentFromElements(key, mapElems...)
	case reflect.Slice:
		// We specifically check if the value is nil so we can properly round trip.
		// If we didn't do this, we couldn't differentiate between an empty slice, which should
		// be an array, and a nil slice, which should be null. In Go, there is a difference
		// between the empty value and nil, we should preserve that.
		if val.IsNil() {
			elem = EC.Null(key)
			break
		}
		if val.Type() == tByteSlice {
			elem = EC.Binary(key, val.Slice(0, val.Len()).Interface().([]byte))
			break
		}
		sliceElems, err := e.encodeSliceAsArray(val, minsize)
		if err != nil {
			return nil, err
		}
		elem = EC.ArrayFromElements(key, sliceElems...)
	case reflect.Array:
		switch {
		case val.Type() == tOID:
			elem = EC.ObjectID(key, val.Interface().(objectid.ObjectID))
		case val.Type().Elem() == tByte:
			b := make([]byte, val.Len())
			for i := 0; i < val.Len(); i++ {
				b[i] = byte(val.Index(i).Uint())
			}
			elem = EC.Binary(key, b)
		default:
			arrayElems, err := e.encodeSliceAsArray(val, minsize)
			if err != nil {
				return nil, err
			}
			elem = EC.ArrayFromElements(key, arrayElems...)
		}
	case reflect.Struct:
		structElems, err := e.encodeStruct(val)
		if err != nil {
			return nil, err
		}
		elem = EC.SubDocumentFromElements(key, structElems...)
	default:
		return nil, fmt.Errorf("Unsupported value type %s", val.Kind())
	}
	return elem, nil
}

func (e *encoder) valueFromValue(val reflect.Value, minsize bool) (*Value, error) {
	var elem *Value
	switch val.Kind() {
	case reflect.Interface, reflect.Ptr:
		if !val.IsNil() {
			return nil, errors.New("Values must be unwrapped when calling elemFromValue, try calling underlyingVal first")
		}
		elem = VC.Null()
	case reflect.Bool:
		elem = VC.Boolean(val.Bool())
	case reflect.Int8, reflect.Int16, reflect.Int32:
		elem = VC.Int32(int32(val.Int()))
	case reflect.Int, reflect.Int64:
		i := val.Int()
		if minsize && i < math.MaxInt32 {
			elem = VC.Int32(int32(val.Int()))
			break
		}
		elem = VC.Int64(val.Int())
	case reflect.Uint8, reflect.Uint16:
		i := val.Uint()
		elem = VC.Int32(int32(i))
	case reflect.Uint, reflect.Uint32, reflect.Uint64:
		i := val.Uint()
		switch {
		case i < math.MaxInt32 && minsize:
			elem = VC.Int32(int32(i))
		case i < math.MaxInt64:
			elem = VC.Int64(int64(i))
		default:
			return nil, fmt.Errorf("BSON only has signed integer types and %d overflows an int64", i)
		}
	case reflect.Float32, reflect.Float64:
		elem = VC.Double(val.Float())
	case reflect.String:
		elem = VC.String(val.String())
	case reflect.Map:
		// We specifically check if the value is nil so we can properly round trip.
		// If we didn't do this, we couldn't differentiate between an empty map, which should
		// be a document, and a nil map, which should be null. In Go, there is a difference
		// between the empty value and nil, we should preserve that.
		if val.IsNil() {
			elem = VC.Null()
			break
		}
		mapElems, err := e.encodeMap(val)
		if err != nil {
			return nil, err
		}
		elem = VC.DocumentFromElements(mapElems...)
	case reflect.Slice:
		// We specifically check if the value is nil so we can properly round trip.
		// If we didn't do this, we couldn't differentiate between an empty slice, which should
		// be an array, and a nil slice, which should be null. In Go, there is a difference
		// between the empty value and nil, we should preserve that.
		if val.IsNil() {
			elem = VC.Null()
			break
		}
		if val.Type() == tByteSlice {
			elem = VC.Binary(val.Slice(0, val.Len()).Interface().([]byte))
			break
		}
		sliceElems, err := e.encodeSliceAsArray(val, minsize)
		if err != nil {
			return nil, err
		}
		elem = VC.ArrayFromValues(sliceElems...)
	case reflect.Array:
		switch {
		case val.Type() == tOID:
			elem = VC.ObjectID(val.Interface().(objectid.ObjectID))
		case val.Type().Elem() == tByte:
			b := make([]byte, val.Len())
			for i := 0; i < val.Len(); i++ {
				b[i] = byte(val.Index(i).Uint())
			}
			elem = VC.Binary(b)
		default:
			arrayElems, err := e.encodeSliceAsArray(val, minsize)
			if err != nil {
				return nil, err
			}
			elem = VC.ArrayFromValues(arrayElems...)
		}
	case reflect.Struct:
		structElems, err := e.encodeStruct(val)
		if err != nil {
			return nil, err
		}
		elem = VC.DocumentFromElements(structElems...)
	default:
		return nil, fmt.Errorf("Unsupported value type %s", val.Kind())
	}
	return elem, nil
}
