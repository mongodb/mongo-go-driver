// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"fmt"
	"io"
	"math"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"bytes"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

var tBinary = reflect.TypeOf(Binary{})
var tBool = reflect.TypeOf(false)
var tCodeWithScope = reflect.TypeOf(CodeWithScope{})
var tDBPointer = reflect.TypeOf(DBPointer{})
var tDecimal = reflect.TypeOf(decimal.Decimal128{})
var tDocument = reflect.TypeOf((*Document)(nil))
var tFloat32 = reflect.TypeOf(float32(0))
var tFloat64 = reflect.TypeOf(float64(0))
var tInt = reflect.TypeOf(int(0))
var tInt8 = reflect.TypeOf(int8(0))
var tInt16 = reflect.TypeOf(int16(0))
var tInt32 = reflect.TypeOf(int32(0))
var tInt64 = reflect.TypeOf(int64(0))
var tJavaScriptCode = reflect.TypeOf(JavaScriptCode(""))
var tOID = reflect.TypeOf(objectid.ObjectID{})
var tReader = reflect.TypeOf(Reader(nil))
var tRegex = reflect.TypeOf(Regex{})
var tString = reflect.TypeOf("")
var tSymbol = reflect.TypeOf(Symbol(""))
var tTime = reflect.TypeOf(time.Time{})
var tTimestamp = reflect.TypeOf(Timestamp{})
var tUint = reflect.TypeOf(uint(0))
var tUint8 = reflect.TypeOf(uint8(0))
var tUint16 = reflect.TypeOf(uint16(0))
var tUint32 = reflect.TypeOf(uint32(0))
var tUint64 = reflect.TypeOf(uint64(0))

var tEmpty = reflect.TypeOf((*interface{})(nil)).Elem()
var tEmptySlice = reflect.TypeOf([]interface{}(nil))

var zeroVal reflect.Value

// this references the quantity of milliseconds between zero time and
// the unix epoch. useful for making sure that we convert time.Time
// objects correctly to match the legacy bson library's handling of
// time.Time values.
const zeroEpochMs = int64(62135596800000)

// Unmarshaler describes a type that can unmarshal itself from BSON bytes.
type Unmarshaler interface {
	UnmarshalBSON([]byte) error
}

// DocumentUnmarshaler describes a type that can unmarshal itself from a bson.Document.
type DocumentUnmarshaler interface {
	UnmarshalBSONDocument(*Document) error
}

// Decoder describes a BSON representation that can decodes itself into a value.
type Decoder interface {
	Decode(interface{}) error
}

// decoder facilitates decoding a value from an io.Reader yielding a BSON document as bytes.
type decoder struct {
	pReader    *peekLengthReader
	bsonReader Reader
}

type peekLengthReader struct {
	io.Reader
	length [4]byte
	pos    int32
}

func newPeekLengthReader(r io.Reader) *peekLengthReader {
	return &peekLengthReader{Reader: r, pos: -1}
}

func (r *peekLengthReader) peekLength() (int32, error) {
	_, err := io.ReadFull(r, r.length[:])
	if err != nil {
		return 0, err
	}

	// Mark that the length has been read.
	r.pos = 0

	return readi32(r.length[:]), nil
}

func (r *peekLengthReader) Read(b []byte) (int, error) {
	// If either peekLength hasn't been called or the length has been read past, read from the
	// io.Reader.
	if r.pos < 0 || r.pos > 3 {
		return r.Reader.Read(b)
	}

	// Read as much of the length as possible into the buffer
	bytesToRead := 4 - r.pos
	if len(b) < int(bytesToRead) {
		bytesToRead = int32(len(b))
	}

	r.pos += int32(copy(b, r.length[r.pos:r.pos+bytesToRead]))

	// Because we use io.ReadFull everywhere, we don't need to read any further since it will be
	// read in a subsequent call to Read.
	return int(bytesToRead), nil
}

func convertToPtr(val reflect.Value) reflect.Value {
	valPtr := reflect.New(val.Type())
	valPtr.Elem().Set(val)
	return valPtr
}

// NewDecoder constructs a new default Decoder implementation from the given io.Reader.
//
// In this implementation, the value can be any one of the following types:
//
//   - bson.Unmarshaler
//   - io.Writer
//   - []byte
//   - bson.Reader
//   - any map with string keys
//   - a struct (possibly with tags)
//
// In the case of struct values, only exported fields will be deserialized. The lowercased field
// name is used as the key for each exported field, but this behavior may be changed using a struct
// tag. The tag may also contain flags to adjust the unmarshaling behavior for the field. The tag
// formats accepted are:
//
//     "[<key>][,<flag1>[,<flag2>]]"
//
//     `(...) bson:"[<key>][,<flag1>[,<flag2>]]" (...)`
//
// The target field or element types of out may not necessarily match the BSON values of the
// provided data. The following conversions are made automatically:
//
//   - Numeric types are converted if at least the integer part of the value would be preserved
//     correctly
//
// If the value would not fit the type and cannot be converted, it is silently skipped.
//
// Pointer values are initialized when necessary.
func NewDecoder(r io.Reader) Decoder {
	return newDecoder(r)
}

func newDecoder(r io.Reader) *decoder {
	return &decoder{pReader: newPeekLengthReader(r)}
}

// Decode decodes the BSON document from the underlying io.Reader into the given value.
func (d *decoder) Decode(v interface{}) error {
	switch t := v.(type) {
	case Unmarshaler:
		err := d.decodeToReader()
		if err != nil {
			return err
		}

		return t.UnmarshalBSON(d.bsonReader)
	case io.Writer:
		err := d.decodeToReader()
		if err != nil {
			return err
		}

		_, err = t.Write(d.bsonReader)
		return err
	case []byte:
		length, err := d.pReader.peekLength()
		if err != nil {
			return err
		}

		if len(t) < int(length) {
			return NewErrTooSmall()
		}

		_, err = io.ReadFull(d.pReader, t)
		if err != nil {
			return err
		}

		_, err = Reader(t).Validate()
		return err
	case Reader:
		length, err := d.pReader.peekLength()
		if err != nil {
			return err
		}

		if len(t) < int(length) {
			return NewErrTooSmall()
		}

		_, err = io.ReadAtLeast(d.pReader, t, int(length))
		if err != nil {
			return err
		}

		_, err = t.Validate()
		return err

	default:
		rval := reflect.ValueOf(v)
		return d.reflectDecode(rval)
	}
}

func (d *decoder) decodeToReader() error {
	var err error
	d.bsonReader, err = NewFromIOReader(d.pReader)
	if err != nil {
		return err
	}

	_, err = d.bsonReader.Validate()
	return err

}

func (d *decoder) reflectDecode(val reflect.Value) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%s", e)
		}
	}()

	switch val.Kind() {
	case reflect.Map:
		return d.decodeIntoMap(val)
	case reflect.Slice, reflect.Array:
		return d.decodeIntoElementSlice(val)
	case reflect.Struct:
		return d.decodeIntoStruct(val)
	case reflect.Ptr:
		v := val.Elem()

		if v.Kind() == reflect.Struct {
			return d.decodeIntoStruct(v)
		}

		fallthrough
	default:
		return fmt.Errorf("cannot decode BSON document to type %s", val.Type())
	}
}

func (d *decoder) createEmptyValue(r Reader, t reflect.Type) (reflect.Value, error) {
	var val reflect.Value

	if t == tReader {
		return reflect.ValueOf(r), nil
	}

	switch t.Kind() {
	case reflect.Map:
		val = reflect.MakeMap(t)
	case reflect.Ptr:
		if t == tDocument {
			val = reflect.ValueOf(NewDocument())
			break
		}

		empty, err := d.createEmptyValue(r, t.Elem())
		if err != nil {
			return val, err
		}

		val = reflect.New(empty.Type())
		val.Elem().Set(empty)
	case reflect.Slice:
		length := 0
		_, err := r.readElements(func(_ *Element) error {
			length++
			return nil
		})

		if err != nil {
			return val, err
		}

		val = reflect.MakeSlice(t.Elem(), length, length)
	case reflect.Struct:
		val = reflect.New(t)
	default:
		val = reflect.Zero(t)
	}

	return val, nil
}

func (d *decoder) getReflectValue(v *Value, containerType reflect.Type, outer reflect.Type) (reflect.Value, error) {
	var val reflect.Value

	isPtr := (containerType.Kind() == reflect.Ptr)

	for containerType.Kind() == reflect.Ptr {
		containerType = containerType.Elem()
	}

	switch v.Type() {
	case 0x1:
		f := v.Double()

		switch containerType {
		case tUint8:
			if f > 0 && math.Floor(f) == f && f <= float64(math.MaxUint8) {
				val = reflect.ValueOf(uint8(f))
			}
		case tUint16:
			if f > 0 && math.Floor(f) == f && f <= float64(math.MaxUint16) {
				val = reflect.ValueOf(uint16(f))
			}
		case tUint32:
			if f > 0 && math.Floor(f) == f && f <= float64(math.MaxUint32) {
				val = reflect.ValueOf(uint32(f))
			}
		case tUint64:
			if f > 0 && math.Floor(f) == f && f <= float64(math.MaxUint64) {
				val = reflect.ValueOf(uint64(f))
			}
		case tUint:
			if f < 0 || math.Floor(f) != f || f > float64(math.MaxUint64) {
				break
			}

			u := uint64(f)
			if uint64(uint(u)) == u {
				val = reflect.ValueOf(uint(f))
			}
		case tInt8:
			if math.Floor(f) == f && f >= float64(math.MinInt8) && f <= float64(math.MaxInt8) {
				val = reflect.ValueOf(int8(f))
			}
		case tInt16:
			if math.Floor(f) == f && f >= float64(math.MinInt16) && f <= float64(math.MaxInt16) {
				val = reflect.ValueOf(int16(f))
			}
		case tInt32:
			if math.Floor(f) == f && f >= float64(math.MinInt32) && f <= float64(math.MaxInt32) {
				val = reflect.ValueOf(int32(f))
			}
		case tInt64:
			if math.Floor(f) == f && f >= float64(math.MinInt64) && f <= float64(math.MaxInt64) {
				val = reflect.ValueOf(int64(f))
			}
		case tInt:
			if math.Floor(f) != f || f < float64(math.MinInt64) || f > float64(math.MaxInt64) {
				break
			}

			i := int64(f)
			if int64(int(i)) == i {
				val = reflect.ValueOf(int(i))
			}

		case tFloat32:
			val = reflect.ValueOf(float32(f))

		case tFloat64, tEmpty:
			val = reflect.ValueOf(f)
		case tJSONNumber:
			val = reflect.ValueOf(strconv.FormatFloat(f, 'f', -1, 64)).Convert(tJSONNumber)
		default:
			return val, nil
		}

	case 0x2:
		str := v.StringValue()
		switch containerType {
		case tString, tEmpty:
			val = reflect.ValueOf(str)
		case tJSONNumber:
			_, err := strconv.ParseFloat(str, 64)
			if err != nil {
				return val, err
			}
			val = reflect.ValueOf(str).Convert(tJSONNumber)

		case tURL:
			u, err := url.Parse(str)
			if err != nil {
				return val, err
			}
			val = reflect.ValueOf(u).Elem()
		default:
			return val, nil
		}
	case 0x4:
		if containerType == tEmpty {
			d := newDecoder(bytes.NewBuffer(v.ReaderArray()))
			newVal, err := d.decodeBSONArrayToSlice(tEmptySlice)
			if err != nil {
				return val, err
			}

			if isPtr {
				val = convertToPtr(newVal)
				isPtr = false
			} else {
				val = newVal
			}

			break
		}

		if containerType.Kind() == reflect.Slice {
			d := newDecoder(bytes.NewBuffer(v.ReaderArray()))
			newVal, err := d.decodeBSONArrayToSlice(containerType)
			if err != nil {
				return val, err
			}

			if isPtr {
				val = convertToPtr(newVal)
				isPtr = false
			} else {
				val = newVal
			}

			break
		}

		if containerType.Kind() == reflect.Array {
			d := newDecoder(bytes.NewBuffer(v.ReaderArray()))
			newVal, err := d.decodeBSONArrayIntoArray(containerType)
			if err != nil {
				return val, err
			}

			if isPtr {
				val = convertToPtr(newVal)
				isPtr = false
			} else {
				val = newVal
			}

			break
		}

		fallthrough

	case 0x3:
		r := v.ReaderDocument()

		typeToCreate := containerType
		if typeToCreate == tEmpty {
			typeToCreate = outer
		}

		empty, err := d.createEmptyValue(r, typeToCreate)
		if err != nil {
			return val, err
		}

		d := NewDecoder(bytes.NewBuffer(r))
		err = d.Decode(empty.Interface())
		if err != nil {
			return val, err
		}

		if reflect.PtrTo(typeToCreate) == empty.Type() {
			empty = empty.Elem()
			if isPtr {
				empty = convertToPtr(empty)
				isPtr = false
			}
		}

		val = empty

	case 0x5:
		switch containerType {
		case tByteSlice:
			_, data := v.Binary()
			val = reflect.ValueOf(data)
		case tEmpty, tBinary:
			st, data := v.Binary()
			val = reflect.ValueOf(Binary{Subtype: st, Data: data})
		}

	case 0x6:
		if containerType != tEmpty {
			return val, nil
		}

		val = reflect.ValueOf(Undefined)
	case 0x7:
		if containerType != tOID && containerType != tEmpty {
			return val, nil
		}

		val = reflect.ValueOf(v.ObjectID())
	case 0x8:
		if containerType != tBool && containerType != tEmpty {
			return val, nil
		}

		val = reflect.ValueOf(v.Boolean())
	case 0x9:
		if containerType != tTime && containerType != tEmpty {
			return val, nil
		}

		if int64(v.getUint64()) == -zeroEpochMs {
			val = reflect.ValueOf(time.Time{})
		} else {
			val = reflect.ValueOf(v.DateTime())
		}

	case 0xA:
		if containerType != tEmpty {
			return val, nil
		}

		val = reflect.ValueOf(Null)
	case 0xB:
		if containerType != tRegex && containerType != tEmpty {
			return val, nil
		}

		p, o := v.Regex()
		val = reflect.ValueOf(Regex{Pattern: p, Options: o})
	case 0xC:
		if containerType != tDBPointer && containerType != tEmpty {
			return val, nil
		}

		db, p := v.DBPointer()
		val = reflect.ValueOf(DBPointer{DB: db, Pointer: p})
	case 0xD:
		if containerType != tJavaScriptCode && containerType != tString && containerType != tEmpty {
			return val, nil
		}

		val = reflect.ValueOf(v.JavaScript())
	case 0xE:
		if containerType != tSymbol && containerType != tString && containerType != tEmpty {
			return val, nil
		}

		val = reflect.ValueOf(v.Symbol())
	case 0xF:
		if containerType != tCodeWithScope && containerType != tEmpty {
			return val, nil
		}

		code, scope := v.MutableJavaScriptWithScope()
		val = reflect.ValueOf(CodeWithScope{Code: code, Scope: scope})
	case 0x10:
		i := v.Int32()

		switch containerType {
		case tInt8:
			if i >= int32(math.MinInt8) && i <= int32(math.MaxInt8) {
				val = reflect.ValueOf(int8(i))
			}

		case tInt16:
			if i >= int32(math.MinInt16) && i <= int32(math.MaxInt16) {
				val = reflect.ValueOf(int16(i))
			}

		case tUint8:
			if i >= 0 && i <= int32(math.MaxUint8) {
				val = reflect.ValueOf(uint8(i))
			}

		case tUint16:
			if i >= 0 && i <= int32(math.MaxUint16) {
				val = reflect.ValueOf(uint16(i))
			}

		case tUint32:
			if i < 0 {
				return val, nil
			}

			val = reflect.ValueOf(uint32(i))
		case tUint64:
			if i < 0 {
				return val, nil
			}

			val = reflect.ValueOf(uint64(i))
		case tUint:
			if i < 0 {
				return val, nil
			}

			val = reflect.ValueOf(uint(i))
		case tEmpty, tInt32, tInt64, tInt, tFloat32, tFloat64:
			val = reflect.ValueOf(i).Convert(containerType)
		case tJSONNumber:
			val = reflect.ValueOf(strconv.FormatInt(int64(i), 10)).Convert(tJSONNumber)
		default:
			return val, nil
		}

	case 0x11:
		if containerType != tTimestamp && containerType != tEmpty {
			return val, nil
		}

		t, i := v.Timestamp()
		val = reflect.ValueOf(Timestamp{T: t, I: i})
	case 0x12:
		i := v.Int64()

		switch containerType {
		case tInt8:
			if i >= int64(math.MinInt8) && i <= int64(math.MaxInt8) {
				val = reflect.ValueOf(int8(i))
			}

		case tInt16:
			if i >= int64(math.MinInt16) && i <= int64(math.MaxInt16) {
				val = reflect.ValueOf(int16(i))
			}

		case tUint8:
			if i >= 0 && i <= int64(math.MaxUint8) {
				val = reflect.ValueOf(uint8(i))
			}

		case tUint16:
			if i >= 0 && i <= int64(math.MaxUint16) {
				val = reflect.ValueOf(uint16(i))
			}

		case tUint32:
			if i >= 0 && i <= math.MaxUint32 {
				val = reflect.ValueOf(uint32(i))
			}
		case tUint64:
			if i >= 0 {
				val = reflect.ValueOf(uint64(i))
			}
		case tUint:
			if i >= 0 && int64(uint(i)) == i {
				val = reflect.ValueOf(uint(i))
			}
		case tInt32:
			if i >= int64(math.MinInt32) && i <= int64(math.MaxInt32) {
				val = reflect.ValueOf(int32(i))
			}
		case tInt:
			// Check the value can fit in an int
			if int64(int(i)) == i {
				val = reflect.ValueOf(int(i))
			}
		case tInt64, tEmpty:
			val = reflect.ValueOf(i)
		case tFloat32:
			val = reflect.ValueOf(float32(i))
		case tFloat64:
			val = reflect.ValueOf(float64(i))
		case tJSONNumber:
			val = reflect.ValueOf(strconv.FormatInt(i, 10)).Convert(tJSONNumber)
		}

	case 0x13:
		if containerType != tDecimal && containerType != tEmpty {
			return val, nil
		}

		val = reflect.ValueOf(v.Decimal128())
	case 0xFF:
		if containerType != tEmpty {
			return val, nil
		}

		val = reflect.ValueOf(MinKey)
	case 0x7f:
		if containerType != tEmpty {
			return val, nil
		}

		val = reflect.ValueOf(MaxKey)
	default:
		return val, fmt.Errorf("invalid BSON type: %s", v.Type())
	}

	if isPtr && val.IsValid() && !val.CanAddr() {
		val = convertToPtr(val)
	}
	return val, nil
}

func (d *decoder) decodeIntoMap(mapVal reflect.Value) error {
	err := d.decodeToReader()
	if err != nil {
		return err
	}

	itr, err := d.bsonReader.Iterator()
	if err != nil {
		return err
	}

	valType := mapVal.Type().Elem()

	for itr.Next() {
		elem := itr.Element()

		v, err := d.getReflectValue(elem.value, valType, mapVal.Type())
		if err != nil {
			return err
		}

		k := reflect.ValueOf(elem.Key())
		mapVal.SetMapIndex(k, v)
	}

	return itr.Err()
}

func (d *decoder) decodeBSONArrayToSlice(sliceType reflect.Type) (reflect.Value, error) {
	var out reflect.Value

	elems := make([]reflect.Value, 0)

	err := d.decodeToReader()
	if err != nil {
		return out, err
	}

	itr, err := d.bsonReader.Iterator()
	if err != nil {
		return out, err
	}

	for itr.Next() {
		v, err := d.getReflectValue(
			itr.Element().Clone().Value(),
			sliceType.Elem(),
			sliceType,
		)
		if err != nil {
			return out, err
		}
		if !v.IsValid() {
			continue
		}

		elems = append(elems, v)
	}

	out = reflect.MakeSlice(sliceType, len(elems), len(elems))

	for i, elem := range elems {
		if i >= out.Len() {
			break
		}

		if sliceType.Elem().Kind() == reflect.Ptr {
			if elem.CanAddr() {
				elem = elem.Addr()
			} else {
				elem = elem.Elem().Addr()
			}
		}
		out.Index(i).Set(elem)
	}

	return out, nil
}

func (d *decoder) decodeBSONArrayIntoArray(arrayType reflect.Type) (reflect.Value, error) {
	length := arrayType.Len()
	arrayVal := reflect.New(arrayType)

	err := d.decodeToReader()
	if err != nil {
		return arrayVal, err
	}

	itr, err := d.bsonReader.Iterator()
	if err != nil {
		return arrayVal, err
	}

	i := 0
	for itr.Next() {
		if i >= length {
			break
		}

		v, err := d.getReflectValue(
			itr.Element().Clone().Value(),
			arrayType.Elem(),
			arrayType,
		)
		if err != nil {
			return arrayVal, err
		}

		arrayVal.Elem().Index(i).Set(v)
		i++
	}

	if err = itr.Err(); err != nil {
		return arrayVal, err
	}

	return arrayVal.Elem(), nil
}

func (d *decoder) decodeIntoElementSlice(sliceVal reflect.Value) error {
	if sliceVal.Type().Elem() != tElement {
		return nil
	}

	sliceLength := sliceVal.Len()

	err := d.decodeToReader()
	if err != nil {
		return err
	}

	itr, err := d.bsonReader.Iterator()
	if err != nil {
		return err
	}

	i := 0
	for itr.Next() {
		if i >= sliceLength {
			return NewErrTooSmall()
		}

		elem := reflect.ValueOf(itr.Element().Clone())
		sliceVal.Index(i).Set(elem)
		i++
	}

	return itr.Err()
}

func matchesField(key string, field string, sType reflect.Type) bool {
	sField, found := sType.FieldByName(field)
	if !found {
		return false
	}

	tag, ok := sField.Tag.Lookup("bson")
	if !ok {
		// Get the full tag string
		tag = string(sField.Tag)

		if len(sField.Tag) == 0 || strings.ContainsRune(tag, ':') {
			return strings.ToLower(key) == strings.ToLower(field)
		}
	}

	var fieldKey string
	i := strings.IndexRune(tag, ',')
	if i == -1 {
		fieldKey = tag
	} else {
		fieldKey = tag[:i]
	}

	return fieldKey == key
}

func (d *decoder) decodeIntoStruct(structVal reflect.Value) error {
	err := d.decodeToReader()
	if err != nil {
		return err
	}

	itr, err := d.bsonReader.Iterator()
	if err != nil {
		return err
	}

	sType := structVal.Type()

	for itr.Next() {
		elem := itr.Element()

		field := structVal.FieldByNameFunc(func(field string) bool {
			return matchesField(elem.Key(), field, sType)
		})
		if field == zeroVal {
			continue
		}

		v, err := d.getReflectValue(elem.value, field.Type(), structVal.Type())
		if err != nil {
			return err
		}

		if v != zeroVal {
			if field.Type().Kind() == reflect.Ptr {
				if v.CanAddr() {
					v = v.Addr()
				} else {
					v = v.Elem().Addr()
				}
			}

			field.Set(v)
		}
	}

	return itr.Err()
}
