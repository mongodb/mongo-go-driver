// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsonx

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/mongodb/mongo-go-driver/bson/bsoncodec"
	"github.com/mongodb/mongo-go-driver/bson/bsonrw"
	"github.com/mongodb/mongo-go-driver/bson/bsontype"
)

var tDocument = reflect.TypeOf((Doc)(nil))
var tMDoc = reflect.TypeOf((MDoc)(nil))
var tArray = reflect.TypeOf((Arr)(nil))

// PrimitiveCodecs is a namespace for all of the default bsoncodec.Codecs for the primitive types
// defined in this package.
type PrimitiveCodecs struct{}

// DocumentEncodeValue is the ValueEncoderFunc for *Document.
func (pc PrimitiveCodecs) DocumentEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	doc, ok := i.(Doc)
	if !ok {
		return bsoncodec.ValueEncoderError{Name: "DocumentEncodeValue", Types: []interface{}{(Doc)(nil), (*Doc)(nil)}, Received: i}
	}

	if doc == nil {
		return vw.WriteNull()
	}

	dw, err := vw.WriteDocument()
	if err != nil {
		return err
	}

	return pc.encodeDocument(ec, dw, doc)
}

// DocumentDecodeValue is the ValueDecoderFunc for *Document.
func (pc PrimitiveCodecs) DocumentDecodeValue(dctx bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	doc, ok := i.(*Doc)
	if !ok {
		return bsoncodec.ValueDecoderError{Name: "DocumentDecodeValue", Types: []interface{}{(*Doc)(nil)}, Received: i}
	}

	if doc == nil {
		return errors.New("DocumentDecodeValue can only be used to decode non-nil *Doc")
	}

	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}

	return pc.decodeDocument(dctx, dr, doc)
}

// ArrayEncodeValue is the ValueEncoderFunc for *Array.
func (pc PrimitiveCodecs) ArrayEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var arr Arr
	switch tt := i.(type) {
	case Arr:
		if tt == nil {
			return vw.WriteNull()
		}
		arr = tt
	case *Arr:
		if tt == nil {
			return vw.WriteNull()
		}
		arr = *tt
	default:
		return bsoncodec.ValueEncoderError{Name: "ArrayEncodeValue", Types: []interface{}{(Arr)(nil), (*Arr)(nil)}, Received: i}
	}

	aw, err := vw.WriteArray()
	if err != nil {
		return err
	}

	for _, val := range arr {
		dvw, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		err = pc.encodeValue(ec, dvw, val)

		if err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

// ArrayDecodeValue is the ValueDecoderFunc for *Array.
func (pc PrimitiveCodecs) ArrayDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	parr, ok := i.(*Arr)
	if !ok {
		return bsoncodec.ValueDecoderError{Name: "ArrayDecodeValue", Types: []interface{}{(*Arr)(nil)}, Received: i}
	}

	if parr == nil {
		return errors.New("ArrayDecodeValue can only be used to decode non-nil *Arr")
	}

	ar, err := vr.ReadArray()
	if err != nil {
		return err
	}

	if *parr == nil {
		*parr = make(Arr, 0)
	}
	*parr = (*parr)[:0]
	for {
		vr, err := ar.ReadValue()
		if err == bsonrw.ErrEOA {
			break
		}
		if err != nil {
			return err
		}

		var val Val
		err = pc.valueDecodeValue(dc, vr, &val)
		if err != nil {
			return err
		}

		*parr = append(*parr, val)
	}

	return nil
}

// ElementSliceEncodeValue is the ValueEncoderFunc for []*Element.
func (pc PrimitiveCodecs) ElementSliceEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var slce []Elem
	switch t := i.(type) {
	case []Elem:
		slce = t
	case *[]Elem:
		if t == nil {
			return vw.WriteNull()
		}
		slce = *t
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "ElementSliceEncodeValue",
			Types:    []interface{}{[]Elem{}, (*[]Elem)(nil)},
			Received: i,
		}
	}

	return pc.DocumentEncodeValue(ec, vw, Doc(slce))
}

// ElementSliceDecodeValue is the ValueDecoderFunc for []*Element.
func (pc PrimitiveCodecs) ElementSliceDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	dr, err := vr.ReadDocument()
	if err != nil {
		return err
	}
	elems := make([]Elem, 0)
	for {
		key, vr, err := dr.ReadElement()
		if err == bsonrw.ErrEOD {
			break
		}
		if err != nil {
			return err
		}

		var elem Elem
		err = pc.elementDecodeValue(dc, vr, key, &elem)
		if err != nil {
			return err
		}

		elems = append(elems, elem)
	}

	target, ok := i.(*[]Elem)
	if !ok || target == nil {
		return bsoncodec.ValueDecoderError{Name: "ElementSliceDecodeValue", Types: []interface{}{(*[]Elem)(nil)}, Received: i}
	}

	*target = elems
	return nil
}

// ValueEncodeValue is the ValueEncoderFunc for *Value.
func (pc PrimitiveCodecs) ValueEncodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, i interface{}) error {
	var val Val
	switch tt := i.(type) {
	case Val:
		val = tt
	case *Val:
		if tt == nil {
			return vw.WriteNull()
		}
		val = *tt
	default:
		return bsoncodec.ValueEncoderError{
			Name:     "ValueEncodeValue",
			Types:    []interface{}{Val{}, (*Val)(nil)},
			Received: i,
		}
	}

	return pc.encodeValue(ec, vw, val)
}

// ValueDecodeValue is the ValueDecoderFunc for *Value.
func (pc PrimitiveCodecs) ValueDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, i interface{}) error {
	pval, ok := i.(*Val)
	if !ok {
		return bsoncodec.ValueDecoderError{Name: "ValueDecodeValue", Types: []interface{}{(*Val)(nil)}, Received: i}
	}

	if pval == nil {
		return errors.New("ValueDecodeValue can only be used to decode non-nil *Value")
	}

	return pc.valueDecodeValue(dc, vr, pval)
}

// encodeDocument is a separate function that we use because CodeWithScope
// returns us a DocumentWriter and we need to do the same logic that we would do
// for a document but cannot use a Codec.
func (pc PrimitiveCodecs) encodeDocument(ec bsoncodec.EncodeContext, dw bsonrw.DocumentWriter, doc Doc) error {
	for _, elem := range doc {
		dvw, err := dw.WriteDocumentElement(elem.Key)
		if err != nil {
			return err
		}

		err = pc.encodeValue(ec, dvw, elem.Value)

		if err != nil {
			return err
		}
	}

	return dw.WriteDocumentEnd()
}

// DecodeDocument haves decoding into a Doc from a bsonrw.DocumentReader.
func (pc PrimitiveCodecs) DecodeDocument(dctx bsoncodec.DecodeContext, dr bsonrw.DocumentReader, pdoc *Doc) error {
	return pc.decodeDocument(dctx, dr, pdoc)
}

func (pc PrimitiveCodecs) decodeDocument(dctx bsoncodec.DecodeContext, dr bsonrw.DocumentReader, pdoc *Doc) error {
	if *pdoc == nil {
		*pdoc = make(Doc, 0)
	}
	*pdoc = (*pdoc)[:0]
	for {
		key, vr, err := dr.ReadElement()
		if err == bsonrw.ErrEOD {
			break
		}
		if err != nil {
			return err
		}

		var elem Elem
		err = pc.elementDecodeValue(dctx, vr, key, &elem)
		if err != nil {
			return err
		}

		*pdoc = append(*pdoc, elem)
	}
	return nil
}

func (pc PrimitiveCodecs) elementDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, key string, elem *Elem) error {
	var val Val
	switch vr.Type() {
	case bsontype.Double:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		val = Double(f64)
	case bsontype.String:
		str, err := vr.ReadString()
		if err != nil {
			return err
		}
		val = String(str)
	case bsontype.EmbeddedDocument:
		var embeddedDoc Doc
		err := pc.DocumentDecodeValue(dc, vr, &embeddedDoc)
		if err != nil {
			return err
		}
		val = Document(embeddedDoc)
	case bsontype.Array:
		var arr Arr
		err := pc.ArrayDecodeValue(dc, vr, &arr)
		if err != nil {
			return err
		}
		val = Array(arr)
	case bsontype.Binary:
		data, subtype, err := vr.ReadBinary()
		if err != nil {
			return err
		}
		val = Binary(subtype, data)
	case bsontype.Undefined:
		err := vr.ReadUndefined()
		if err != nil {
			return err
		}
		val = Undefined()
	case bsontype.ObjectID:
		oid, err := vr.ReadObjectID()
		if err != nil {
			return err
		}
		val = ObjectID(oid)
	case bsontype.Boolean:
		b, err := vr.ReadBoolean()
		if err != nil {
			return err
		}
		val = Boolean(b)
	case bsontype.DateTime:
		dt, err := vr.ReadDateTime()
		if err != nil {
			return err
		}
		val = DateTime(dt)
	case bsontype.Null:
		err := vr.ReadNull()
		if err != nil {
			return err
		}
		val = Null()
	case bsontype.Regex:
		pattern, options, err := vr.ReadRegex()
		if err != nil {
			return err
		}
		val = Regex(pattern, options)
	case bsontype.DBPointer:
		ns, pointer, err := vr.ReadDBPointer()
		if err != nil {
			return err
		}
		val = DBPointer(ns, pointer)
	case bsontype.JavaScript:
		js, err := vr.ReadJavascript()
		if err != nil {
			return err
		}
		val = JavaScript(js)
	case bsontype.Symbol:
		symbol, err := vr.ReadSymbol()
		if err != nil {
			return err
		}
		val = Symbol(symbol)
	case bsontype.CodeWithScope:
		code, scope, err := vr.ReadCodeWithScope()
		if err != nil {
			return err
		}
		var doc Doc
		err = pc.decodeDocument(dc, scope, &doc)
		if err != nil {
			return err
		}
		val = CodeWithScope(code, doc)
	case bsontype.Int32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		val = Int32(i32)
	case bsontype.Timestamp:
		t, i, err := vr.ReadTimestamp()
		if err != nil {
			return err
		}
		val = Timestamp(t, i)
	case bsontype.Int64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		val = Int64(i64)
	case bsontype.Decimal128:
		d128, err := vr.ReadDecimal128()
		if err != nil {
			return err
		}
		val = Decimal128(d128)
	case bsontype.MinKey:
		err := vr.ReadMinKey()
		if err != nil {
			return err
		}
		val = MinKey()
	case bsontype.MaxKey:
		err := vr.ReadMaxKey()
		if err != nil {
			return err
		}
		val = MaxKey()
	default:
		return fmt.Errorf("Cannot read unknown BSON type %s", vr.Type())
	}

	*elem = Elem{Key: key, Value: val}
	return nil
}

// encodeValue does not validation, and the callers must perform validation on val before calling
// this method.
func (pc PrimitiveCodecs) encodeValue(ec bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val Val) error {
	var err error
	switch val.Type() {
	case bsontype.Double:
		err = vw.WriteDouble(val.Double())
	case bsontype.String:
		err = vw.WriteString(val.StringValue())
	case bsontype.EmbeddedDocument:
		var encoder bsoncodec.ValueEncoder
		encoder, err = ec.LookupEncoder(tDocument)
		if err != nil {
			break
		}
		err = encoder.EncodeValue(ec, vw, val.Document())
	case bsontype.Array:
		var encoder bsoncodec.ValueEncoder
		encoder, err = ec.LookupEncoder(tArray)
		if err != nil {
			break
		}
		err = encoder.EncodeValue(ec, vw, val.Array())
	case bsontype.Binary:
		// TODO: FIX THIS (╯°□°）╯︵ ┻━┻
		subtype, data := val.Binary()
		err = vw.WriteBinaryWithSubtype(data, subtype)
	case bsontype.Undefined:
		err = vw.WriteUndefined()
	case bsontype.ObjectID:
		err = vw.WriteObjectID(val.ObjectID())
	case bsontype.Boolean:
		err = vw.WriteBoolean(val.Boolean())
	case bsontype.DateTime:
		err = vw.WriteDateTime(val.DateTime())
	case bsontype.Null:
		err = vw.WriteNull()
	case bsontype.Regex:
		err = vw.WriteRegex(val.Regex())
	case bsontype.DBPointer:
		err = vw.WriteDBPointer(val.DBPointer())
	case bsontype.JavaScript:
		err = vw.WriteJavascript(val.JavaScript())
	case bsontype.Symbol:
		err = vw.WriteSymbol(val.Symbol())
	case bsontype.CodeWithScope:
		code, scope := val.CodeWithScope()

		var cwsw bsonrw.DocumentWriter
		cwsw, err = vw.WriteCodeWithScope(code)
		if err != nil {
			break
		}

		err = pc.encodeDocument(ec, cwsw, scope)
	case bsontype.Int32:
		err = vw.WriteInt32(val.Int32())
	case bsontype.Timestamp:
		err = vw.WriteTimestamp(val.Timestamp())
	case bsontype.Int64:
		err = vw.WriteInt64(val.Int64())
	case bsontype.Decimal128:
		err = vw.WriteDecimal128(val.Decimal128())
	case bsontype.MinKey:
		err = vw.WriteMinKey()
	case bsontype.MaxKey:
		err = vw.WriteMaxKey()
	default:
		err = fmt.Errorf("%T is not a valid BSON type to encode", val.Type())
	}

	return err
}

func (pc PrimitiveCodecs) valueDecodeValue(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val *Val) error {
	switch vr.Type() {
	case bsontype.Double:
		f64, err := vr.ReadDouble()
		if err != nil {
			return err
		}
		*val = Double(f64)
	case bsontype.String:
		str, err := vr.ReadString()
		if err != nil {
			return err
		}
		*val = String(str)
	case bsontype.EmbeddedDocument:
		var embeddedDoc Doc
		err := pc.DocumentDecodeValue(dc, vr, &embeddedDoc)
		if err != nil {
			return err
		}
		*val = Document(embeddedDoc)
	case bsontype.Array:
		var arr Arr
		err := pc.ArrayDecodeValue(dc, vr, &arr)
		if err != nil {
			return err
		}
		*val = Array(arr)
	case bsontype.Binary:
		data, subtype, err := vr.ReadBinary()
		if err != nil {
			return err
		}
		*val = Binary(subtype, data)
	case bsontype.Undefined:
		err := vr.ReadUndefined()
		if err != nil {
			return err
		}
		*val = Undefined()
	case bsontype.ObjectID:
		oid, err := vr.ReadObjectID()
		if err != nil {
			return err
		}
		*val = ObjectID(oid)
	case bsontype.Boolean:
		b, err := vr.ReadBoolean()
		if err != nil {
			return err
		}
		*val = Boolean(b)
	case bsontype.DateTime:
		dt, err := vr.ReadDateTime()
		if err != nil {
			return err
		}
		*val = DateTime(dt)
	case bsontype.Null:
		err := vr.ReadNull()
		if err != nil {
			return err
		}
		*val = Null()
	case bsontype.Regex:
		pattern, options, err := vr.ReadRegex()
		if err != nil {
			return err
		}
		*val = Regex(pattern, options)
	case bsontype.DBPointer:
		ns, pointer, err := vr.ReadDBPointer()
		if err != nil {
			return err
		}
		*val = DBPointer(ns, pointer)
	case bsontype.JavaScript:
		js, err := vr.ReadJavascript()
		if err != nil {
			return err
		}
		*val = JavaScript(js)
	case bsontype.Symbol:
		symbol, err := vr.ReadSymbol()
		if err != nil {
			return err
		}
		*val = Symbol(symbol)
	case bsontype.CodeWithScope:
		code, scope, err := vr.ReadCodeWithScope()
		if err != nil {
			return err
		}
		var scopeDoc Doc
		err = pc.decodeDocument(dc, scope, &scopeDoc)
		if err != nil {
			return err
		}
		*val = CodeWithScope(code, scopeDoc)
	case bsontype.Int32:
		i32, err := vr.ReadInt32()
		if err != nil {
			return err
		}
		*val = Int32(i32)
	case bsontype.Timestamp:
		t, i, err := vr.ReadTimestamp()
		if err != nil {
			return err
		}
		*val = Timestamp(t, i)
	case bsontype.Int64:
		i64, err := vr.ReadInt64()
		if err != nil {
			return err
		}
		*val = Int64(i64)
	case bsontype.Decimal128:
		d128, err := vr.ReadDecimal128()
		if err != nil {
			return err
		}
		*val = Decimal128(d128)
	case bsontype.MinKey:
		err := vr.ReadMinKey()
		if err != nil {
			return err
		}
		*val = MinKey()
	case bsontype.MaxKey:
		err := vr.ReadMaxKey()
		if err != nil {
			return err
		}
		*val = MaxKey()
	default:
		return fmt.Errorf("Cannot read unknown BSON type %s", vr.Type())
	}

	return nil
}
