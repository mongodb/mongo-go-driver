// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

// copyDocument handles copying one document from the src to the dst.
func copyDocument(dst ValueWriter, src ValueReader) error {
	dr, err := src.ReadDocument()
	if err != nil {
		return err
	}

	dw, err := dst.WriteDocument()
	if err != nil {
		return err
	}

	return copyDocumentCore(dw, dr)
}

// copyArrayFromBytes copies the values from a BSON array represented as a
// []byte to a ValueWriter.
func copyArrayFromBytes(dst ValueWriter, src []byte) error {
	aw, err := dst.WriteArray()
	if err != nil {
		return err
	}

	err = copyBytesToArrayWriter(aw, src)
	if err != nil {
		return err
	}

	return aw.WriteArrayEnd()
}

// copyDocumentFromBytes copies the values from a BSON document represented as a
// []byte to a ValueWriter.
func copyDocumentFromBytes(dst ValueWriter, src []byte) error {
	dw, err := dst.WriteDocument()
	if err != nil {
		return err
	}

	err = copyBytesToDocumentWriter(dw, src)
	if err != nil {
		return err
	}

	return dw.WriteDocumentEnd()
}

type writeElementFn func(key string) (ValueWriter, error)

// copyBytesToArrayWriter copies the values from a BSON Array represented as a []byte to an
// ArrayWriter.
func copyBytesToArrayWriter(dst ArrayWriter, src []byte) error {
	wef := func(_ string) (ValueWriter, error) {
		return dst.WriteArrayElement()
	}

	return copyBytesToValueWriter(src, wef)
}

// copyBytesToDocumentWriter copies the values from a BSON document represented as a []byte to a
// DocumentWriter.
func copyBytesToDocumentWriter(dst DocumentWriter, src []byte) error {
	wef := func(key string) (ValueWriter, error) {
		return dst.WriteDocumentElement(key)
	}

	return copyBytesToValueWriter(src, wef)
}

func copyBytesToValueWriter(src []byte, wef writeElementFn) error {
	// TODO(skriptble): Create errors types here. Anything that is a tag should be a property.
	length, rem, ok := bsoncore.ReadLength(src)
	if !ok {
		return fmt.Errorf("couldn't read length from src, not enough bytes. length=%d", len(src))
	}
	if len(src) < int(length) {
		return fmt.Errorf("length read exceeds number of bytes available. length=%d bytes=%d", len(src), length)
	}
	rem = rem[:length-4]

	var t bsoncore.Type
	var key string
	var val bsoncore.Value
	for {
		t, rem, ok = bsoncore.ReadType(rem)
		if !ok {
			return io.EOF
		}
		if t == bsoncore.Type(0) {
			if len(rem) != 0 {
				return fmt.Errorf("document end byte found before end of document. remaining bytes=%v", rem)
			}
			break
		}

		key, rem, ok = bsoncore.ReadKey(rem)
		if !ok {
			return fmt.Errorf("invalid key found. remaining bytes=%v", rem)
		}

		// write as either array element or document element using writeElementFn
		vw, err := wef(key)
		if err != nil {
			return err
		}

		val, rem, ok = bsoncore.ReadValue(rem, t)
		if !ok {
			return fmt.Errorf("not enough bytes available to read type. bytes=%d type=%s", len(rem), t)
		}
		err = copyValueFromBytes(vw, Type(t), val.Data)
		if err != nil {
			return err
		}
	}
	return nil
}

// copyDocumentToBytes copies an entire document from the ValueReader and
// returns it as bytes.
func copyDocumentToBytes(src ValueReader) ([]byte, error) {
	return appendDocumentBytes(nil, src)
}

// appendDocumentBytes functions the same as CopyDocumentToBytes, but will
// append the result to dst.
func appendDocumentBytes(dst []byte, src ValueReader) ([]byte, error) {
	if br, ok := src.(bytesReader); ok {
		_, dst, err := br.readValueBytes(dst)
		return dst, err
	}

	vw := vwPool.Get().(*valueWriter)
	defer putValueWriter(vw)

	vw.reset(dst)

	err := copyDocument(vw, src)
	dst = vw.buf
	return dst, err
}

// appendArrayBytes copies an array from the ValueReader to dst.
func appendArrayBytes(dst []byte, src ValueReader) ([]byte, error) {
	if br, ok := src.(bytesReader); ok {
		_, dst, err := br.readValueBytes(dst)
		return dst, err
	}

	vw := vwPool.Get().(*valueWriter)
	defer putValueWriter(vw)

	vw.reset(dst)

	err := copyArray(vw, src)
	dst = vw.buf
	return dst, err
}

// copyValueFromBytes will write the value represtend by t and src to dst.
func copyValueFromBytes(dst ValueWriter, t Type, src []byte) error {
	if wvb, ok := dst.(bytesWriter); ok {
		return wvb.writeValueBytes(t, src)
	}

	vr := newDocumentReader(bytes.NewReader(src))
	vr.pushElement(t)

	return copyValue(dst, vr)
}

// copyValueToBytes copies a value from src and returns it as a Type and a
// []byte.
func copyValueToBytes(src ValueReader) (Type, []byte, error) {
	if br, ok := src.(bytesReader); ok {
		return br.readValueBytes(nil)
	}

	vw := vwPool.Get().(*valueWriter)
	defer putValueWriter(vw)

	vw.reset(nil)
	vw.push(mElement)

	err := copyValue(vw, src)
	if err != nil {
		return 0, nil, err
	}

	return Type(vw.buf[0]), vw.buf[2:], nil
}

// copyValue will copy a single value from src to dst.
func copyValue(dst ValueWriter, src ValueReader) error {
	var err error
	switch src.Type() {
	case TypeDouble:
		var f64 float64
		f64, err = src.ReadDouble()
		if err != nil {
			break
		}
		err = dst.WriteDouble(f64)
	case TypeString:
		var str string
		str, err = src.ReadString()
		if err != nil {
			return err
		}
		err = dst.WriteString(str)
	case TypeEmbeddedDocument:
		err = copyDocument(dst, src)
	case TypeArray:
		err = copyArray(dst, src)
	case TypeBinary:
		var data []byte
		var subtype byte
		data, subtype, err = src.ReadBinary()
		if err != nil {
			break
		}
		err = dst.WriteBinaryWithSubtype(data, subtype)
	case TypeUndefined:
		err = src.ReadUndefined()
		if err != nil {
			break
		}
		err = dst.WriteUndefined()
	case TypeObjectID:
		var oid ObjectID
		oid, err = src.ReadObjectID()
		if err != nil {
			break
		}
		err = dst.WriteObjectID(oid)
	case TypeBoolean:
		var b bool
		b, err = src.ReadBoolean()
		if err != nil {
			break
		}
		err = dst.WriteBoolean(b)
	case TypeDateTime:
		var dt int64
		dt, err = src.ReadDateTime()
		if err != nil {
			break
		}
		err = dst.WriteDateTime(dt)
	case TypeNull:
		err = src.ReadNull()
		if err != nil {
			break
		}
		err = dst.WriteNull()
	case TypeRegex:
		var pattern, options string
		pattern, options, err = src.ReadRegex()
		if err != nil {
			break
		}
		err = dst.WriteRegex(pattern, options)
	case TypeDBPointer:
		var ns string
		var pointer ObjectID
		ns, pointer, err = src.ReadDBPointer()
		if err != nil {
			break
		}
		err = dst.WriteDBPointer(ns, pointer)
	case TypeJavaScript:
		var js string
		js, err = src.ReadJavascript()
		if err != nil {
			break
		}
		err = dst.WriteJavascript(js)
	case TypeSymbol:
		var symbol string
		symbol, err = src.ReadSymbol()
		if err != nil {
			break
		}
		err = dst.WriteSymbol(symbol)
	case TypeCodeWithScope:
		var code string
		var srcScope DocumentReader
		code, srcScope, err = src.ReadCodeWithScope()
		if err != nil {
			break
		}

		var dstScope DocumentWriter
		dstScope, err = dst.WriteCodeWithScope(code)
		if err != nil {
			break
		}
		err = copyDocumentCore(dstScope, srcScope)
	case TypeInt32:
		var i32 int32
		i32, err = src.ReadInt32()
		if err != nil {
			break
		}
		err = dst.WriteInt32(i32)
	case TypeTimestamp:
		var t, i uint32
		t, i, err = src.ReadTimestamp()
		if err != nil {
			break
		}
		err = dst.WriteTimestamp(t, i)
	case TypeInt64:
		var i64 int64
		i64, err = src.ReadInt64()
		if err != nil {
			break
		}
		err = dst.WriteInt64(i64)
	case TypeDecimal128:
		var d128 Decimal128
		d128, err = src.ReadDecimal128()
		if err != nil {
			break
		}
		err = dst.WriteDecimal128(d128)
	case TypeMinKey:
		err = src.ReadMinKey()
		if err != nil {
			break
		}
		err = dst.WriteMinKey()
	case TypeMaxKey:
		err = src.ReadMaxKey()
		if err != nil {
			break
		}
		err = dst.WriteMaxKey()
	default:
		err = fmt.Errorf("cannot copy unknown BSON type %s", src.Type())
	}

	return err
}

func copyArray(dst ValueWriter, src ValueReader) error {
	ar, err := src.ReadArray()
	if err != nil {
		return err
	}

	aw, err := dst.WriteArray()
	if err != nil {
		return err
	}

	for {
		vr, err := ar.ReadValue()
		if errors.Is(err, ErrEOA) {
			break
		}
		if err != nil {
			return err
		}

		vw, err := aw.WriteArrayElement()
		if err != nil {
			return err
		}

		err = copyValue(vw, vr)
		if err != nil {
			return err
		}
	}

	return aw.WriteArrayEnd()
}

func copyDocumentCore(dw DocumentWriter, dr DocumentReader) error {
	for {
		key, vr, err := dr.ReadElement()
		if errors.Is(err, ErrEOD) {
			break
		}
		if err != nil {
			return err
		}

		vw, err := dw.WriteDocumentElement(key)
		if err != nil {
			return err
		}

		err = copyValue(vw, vr)
		if err != nil {
			return err
		}
	}

	return dw.WriteDocumentEnd()
}

// bytesReader is the interface used to read BSON bytes from a valueReader.
//
// The bytes of the value will be appended to dst.
type bytesReader interface {
	readValueBytes(dst []byte) (Type, []byte, error)
}

// bytesWriter is the interface used to write BSON bytes to a valueWriter.
type bytesWriter interface {
	writeValueBytes(t Type, b []byte) error
}
