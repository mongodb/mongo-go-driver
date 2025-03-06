// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
)

var (
	_ ValueReader = &valueReaderWriter{}
	_ ValueWriter = &valueReaderWriter{}
)

// invoked is a type used to indicate what method was called last.
type invoked byte

// These are the different methods that can be invoked.
const (
	nothing invoked = iota
	readArray
	readBinary
	readBoolean
	readDocument
	readCodeWithScope
	readDBPointer
	readDateTime
	readDecimal128
	readDouble
	readInt32
	readInt64
	readJavascript
	readMaxKey
	readMinKey
	readNull
	readObjectID
	readRegex
	readString
	readSymbol
	readTimestamp
	readUndefined
	readElement
	readValue
	writeArray
	writeBinary
	writeBinaryWithSubtype
	writeBoolean
	writeCodeWithScope
	writeDBPointer
	writeDateTime
	writeDecimal128
	writeDouble
	writeInt32
	writeInt64
	writeJavascript
	writeMaxKey
	writeMinKey
	writeNull
	writeObjectID
	writeRegex
	writeString
	writeDocument
	writeSymbol
	writeTimestamp
	writeUndefined
	writeDocumentElement
	writeDocumentEnd
	writeArrayElement
	writeArrayEnd
	skip
)

func (i invoked) String() string {
	switch i {
	case nothing:
		return "Nothing"
	case readArray:
		return "ReadArray"
	case readBinary:
		return "ReadBinary"
	case readBoolean:
		return "ReadBoolean"
	case readDocument:
		return "ReadDocument"
	case readCodeWithScope:
		return "ReadCodeWithScope"
	case readDBPointer:
		return "ReadDBPointer"
	case readDateTime:
		return "ReadDateTime"
	case readDecimal128:
		return "ReadDecimal128"
	case readDouble:
		return "ReadDouble"
	case readInt32:
		return "ReadInt32"
	case readInt64:
		return "ReadInt64"
	case readJavascript:
		return "ReadJavascript"
	case readMaxKey:
		return "ReadMaxKey"
	case readMinKey:
		return "ReadMinKey"
	case readNull:
		return "ReadNull"
	case readObjectID:
		return "ReadObjectID"
	case readRegex:
		return "ReadRegex"
	case readString:
		return "ReadString"
	case readSymbol:
		return "ReadSymbol"
	case readTimestamp:
		return "ReadTimestamp"
	case readUndefined:
		return "ReadUndefined"
	case readElement:
		return "ReadElement"
	case readValue:
		return "ReadValue"
	case writeArray:
		return "WriteArray"
	case writeBinary:
		return "WriteBinary"
	case writeBinaryWithSubtype:
		return "WriteBinaryWithSubtype"
	case writeBoolean:
		return "WriteBoolean"
	case writeCodeWithScope:
		return "WriteCodeWithScope"
	case writeDBPointer:
		return "WriteDBPointer"
	case writeDateTime:
		return "WriteDateTime"
	case writeDecimal128:
		return "WriteDecimal128"
	case writeDouble:
		return "WriteDouble"
	case writeInt32:
		return "WriteInt32"
	case writeInt64:
		return "WriteInt64"
	case writeJavascript:
		return "WriteJavascript"
	case writeMaxKey:
		return "WriteMaxKey"
	case writeMinKey:
		return "WriteMinKey"
	case writeNull:
		return "WriteNull"
	case writeObjectID:
		return "WriteObjectID"
	case writeRegex:
		return "WriteRegex"
	case writeString:
		return "WriteString"
	case writeDocument:
		return "WriteDocument"
	case writeSymbol:
		return "WriteSymbol"
	case writeTimestamp:
		return "WriteTimestamp"
	case writeUndefined:
		return "WriteUndefined"
	case writeDocumentElement:
		return "WriteDocumentElement"
	case writeDocumentEnd:
		return "WriteDocumentEnd"
	case writeArrayElement:
		return "WriteArrayElement"
	case writeArrayEnd:
		return "WriteArrayEnd"
	default:
		return "<unknown>"
	}
}

// valueReaderWriter is a test implementation of a bsonrw.ValueReader and bsonrw.ValueWriter
type valueReaderWriter struct {
	T        *testing.T
	invoked  invoked
	Return   interface{} // Can be a primitive or a bsoncore.Value
	BSONType Type
	Err      error
	ErrAfter invoked // error after this method is called
	depth    uint64
}

// prevent infinite recursion.
func (llvrw *valueReaderWriter) checkdepth() {
	llvrw.depth++
	if llvrw.depth > 1000 {
		panic("max depth exceeded")
	}
}

// Type implements the ValueReader interface.
func (llvrw *valueReaderWriter) Type() Type {
	llvrw.checkdepth()
	return llvrw.BSONType
}

// Skip implements the ValueReader interface.
func (llvrw *valueReaderWriter) Skip() error {
	llvrw.checkdepth()
	llvrw.invoked = skip
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// ReadArray implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadArray() (ArrayReader, error) {
	llvrw.checkdepth()
	llvrw.invoked = readArray
	if llvrw.ErrAfter == llvrw.invoked {
		return nil, llvrw.Err
	}

	return llvrw, nil
}

// ReadBinary implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadBinary() (b []byte, btype byte, err error) {
	llvrw.checkdepth()
	llvrw.invoked = readBinary
	if llvrw.ErrAfter == llvrw.invoked {
		return nil, 0x00, llvrw.Err
	}

	switch tt := llvrw.Return.(type) {
	case bsoncore.Value:
		subtype, data, _, ok := bsoncore.ReadBinary(tt.Data)
		if !ok {
			llvrw.T.Error("Invalid Value provided for return value of ReadBinary.")
			return nil, 0x00, nil
		}
		return data, subtype, nil
	default:
		llvrw.T.Errorf("Incorrect type provided for return value of ReadBinary: %T", llvrw.Return)
		return nil, 0x00, nil
	}
}

// ReadBoolean implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadBoolean() (bool, error) {
	llvrw.checkdepth()
	llvrw.invoked = readBoolean
	if llvrw.ErrAfter == llvrw.invoked {
		return false, llvrw.Err
	}

	switch tt := llvrw.Return.(type) {
	case bool:
		return tt, nil
	case bsoncore.Value:
		b, _, ok := bsoncore.ReadBoolean(tt.Data)
		if !ok {
			llvrw.T.Error("Invalid Value provided for return value of ReadBoolean.")
			return false, nil
		}
		return b, nil
	default:
		llvrw.T.Errorf("Incorrect type provided for return value of ReadBoolean: %T", llvrw.Return)
		return false, nil
	}
}

// ReadDocument implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadDocument() (DocumentReader, error) {
	llvrw.checkdepth()
	llvrw.invoked = readDocument
	if llvrw.ErrAfter == llvrw.invoked {
		return nil, llvrw.Err
	}

	return llvrw, nil
}

// ReadCodeWithScope implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadCodeWithScope() (code string, dr DocumentReader, err error) {
	llvrw.checkdepth()
	llvrw.invoked = readCodeWithScope
	if llvrw.ErrAfter == llvrw.invoked {
		return "", nil, llvrw.Err
	}

	return "", llvrw, nil
}

// ReadDBPointer implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadDBPointer() (ns string, oid ObjectID, err error) {
	llvrw.checkdepth()
	llvrw.invoked = readDBPointer
	if llvrw.ErrAfter == llvrw.invoked {
		return "", ObjectID{}, llvrw.Err
	}

	switch tt := llvrw.Return.(type) {
	case bsoncore.Value:
		ns, oid, _, ok := bsoncore.ReadDBPointer(tt.Data)
		if !ok {
			llvrw.T.Error("Invalid Value instance provided for return value of ReadDBPointer")
			return "", ObjectID{}, nil
		}
		return ns, oid, nil
	default:
		llvrw.T.Errorf("Incorrect type provided for return value of ReadDBPointer: %T", llvrw.Return)
		return "", ObjectID{}, nil
	}
}

// ReadDateTime implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadDateTime() (int64, error) {
	llvrw.checkdepth()
	llvrw.invoked = readDateTime
	if llvrw.ErrAfter == llvrw.invoked {
		return 0, llvrw.Err
	}

	dt, ok := llvrw.Return.(int64)
	if !ok {
		llvrw.T.Errorf("Incorrect type provided for return value of ReadDateTime: %T", llvrw.Return)
		return 0, nil
	}

	return dt, nil
}

// ReadDecimal128 implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadDecimal128() (Decimal128, error) {
	llvrw.checkdepth()
	llvrw.invoked = readDecimal128
	if llvrw.ErrAfter == llvrw.invoked {
		return Decimal128{}, llvrw.Err
	}

	d128, ok := llvrw.Return.(Decimal128)
	if !ok {
		llvrw.T.Errorf("Incorrect type provided for return value of ReadDecimal128: %T", llvrw.Return)
		return Decimal128{}, nil
	}

	return d128, nil
}

// ReadDouble implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadDouble() (float64, error) {
	llvrw.checkdepth()
	llvrw.invoked = readDouble
	if llvrw.ErrAfter == llvrw.invoked {
		return 0, llvrw.Err
	}

	f64, ok := llvrw.Return.(float64)
	if !ok {
		llvrw.T.Errorf("Incorrect type provided for return value of ReadDouble: %T", llvrw.Return)
		return 0, nil
	}

	return f64, nil
}

// ReadInt32 implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadInt32() (int32, error) {
	llvrw.checkdepth()
	llvrw.invoked = readInt32
	if llvrw.ErrAfter == llvrw.invoked {
		return 0, llvrw.Err
	}

	i32, ok := llvrw.Return.(int32)
	if !ok {
		llvrw.T.Errorf("Incorrect type provided for return value of ReadInt32: %T", llvrw.Return)
		return 0, nil
	}

	return i32, nil
}

// ReadInt64 implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadInt64() (int64, error) {
	llvrw.checkdepth()
	llvrw.invoked = readInt64
	if llvrw.ErrAfter == llvrw.invoked {
		return 0, llvrw.Err
	}
	i64, ok := llvrw.Return.(int64)
	if !ok {
		llvrw.T.Errorf("Incorrect type provided for return value of ReadInt64: %T", llvrw.Return)
		return 0, nil
	}

	return i64, nil
}

// ReadJavascript implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadJavascript() (code string, err error) {
	llvrw.checkdepth()
	llvrw.invoked = readJavascript
	if llvrw.ErrAfter == llvrw.invoked {
		return "", llvrw.Err
	}
	js, ok := llvrw.Return.(string)
	if !ok {
		llvrw.T.Errorf("Incorrect type provided for return value of ReadJavascript: %T", llvrw.Return)
		return "", nil
	}

	return js, nil
}

// ReadMaxKey implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadMaxKey() error {
	llvrw.checkdepth()
	llvrw.invoked = readMaxKey
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}

	return nil
}

// ReadMinKey implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadMinKey() error {
	llvrw.checkdepth()
	llvrw.invoked = readMinKey
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}

	return nil
}

// ReadNull implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadNull() error {
	llvrw.checkdepth()
	llvrw.invoked = readNull
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}

	return nil
}

// ReadObjectID implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadObjectID() (ObjectID, error) {
	llvrw.checkdepth()
	llvrw.invoked = readObjectID
	if llvrw.ErrAfter == llvrw.invoked {
		return ObjectID{}, llvrw.Err
	}
	oid, ok := llvrw.Return.(ObjectID)
	if !ok {
		llvrw.T.Errorf("Incorrect type provided for return value of ReadObjectID: %T", llvrw.Return)
		return ObjectID{}, nil
	}

	return oid, nil
}

// ReadRegex implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadRegex() (pattern string, options string, err error) {
	llvrw.checkdepth()
	llvrw.invoked = readRegex
	if llvrw.ErrAfter == llvrw.invoked {
		return "", "", llvrw.Err
	}
	switch tt := llvrw.Return.(type) {
	case bsoncore.Value:
		pattern, options, _, ok := bsoncore.ReadRegex(tt.Data)
		if !ok {
			llvrw.T.Error("Invalid Value instance provided for ReadRegex")
			return "", "", nil
		}
		return pattern, options, nil
	default:
		llvrw.T.Errorf("Incorrect type provided for return value of ReadRegex: %T", llvrw.Return)
		return "", "", nil
	}
}

// ReadString implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadString() (string, error) {
	llvrw.checkdepth()
	llvrw.invoked = readString
	if llvrw.ErrAfter == llvrw.invoked {
		return "", llvrw.Err
	}
	str, ok := llvrw.Return.(string)
	if !ok {
		llvrw.T.Errorf("Incorrect type provided for return value of ReadString: %T", llvrw.Return)
		return "", nil
	}

	return str, nil
}

// ReadSymbol implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadSymbol() (symbol string, err error) {
	llvrw.checkdepth()
	llvrw.invoked = readSymbol
	if llvrw.ErrAfter == llvrw.invoked {
		return "", llvrw.Err
	}
	switch tt := llvrw.Return.(type) {
	case string:
		return tt, nil
	case bsoncore.Value:
		symbol, _, ok := bsoncore.ReadSymbol(tt.Data)
		if !ok {
			llvrw.T.Error("Invalid Value instance provided for ReadSymbol")
			return "", nil
		}
		return symbol, nil
	default:
		llvrw.T.Errorf("Incorrect type provided for return value of ReadSymbol: %T", llvrw.Return)
		return "", nil
	}
}

// ReadTimestamp implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadTimestamp() (t uint32, i uint32, err error) {
	llvrw.checkdepth()
	llvrw.invoked = readTimestamp
	if llvrw.ErrAfter == llvrw.invoked {
		return 0, 0, llvrw.Err
	}
	switch tt := llvrw.Return.(type) {
	case bsoncore.Value:
		t, i, _, ok := bsoncore.ReadTimestamp(tt.Data)
		if !ok {
			llvrw.T.Errorf("Invalid Value instance provided for return value of ReadTimestamp")
			return 0, 0, nil
		}
		return t, i, nil
	default:
		llvrw.T.Errorf("Incorrect type provided for return value of ReadTimestamp: %T", llvrw.Return)
		return 0, 0, nil
	}
}

// ReadUndefined implements the ValueReader interface.
func (llvrw *valueReaderWriter) ReadUndefined() error {
	llvrw.checkdepth()
	llvrw.invoked = readUndefined
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}

	return nil
}

// WriteArray implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteArray() (ArrayWriter, error) {
	llvrw.checkdepth()
	llvrw.invoked = writeArray
	if llvrw.ErrAfter == llvrw.invoked {
		return nil, llvrw.Err
	}
	return llvrw, nil
}

// WriteBinary implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteBinary([]byte) error {
	llvrw.checkdepth()
	llvrw.invoked = writeBinary
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteBinaryWithSubtype implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteBinaryWithSubtype([]byte, byte) error {
	llvrw.checkdepth()
	llvrw.invoked = writeBinaryWithSubtype
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteBoolean implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteBoolean(bool) error {
	llvrw.checkdepth()
	llvrw.invoked = writeBoolean
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteCodeWithScope implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteCodeWithScope(string) (DocumentWriter, error) {
	llvrw.checkdepth()
	llvrw.invoked = writeCodeWithScope
	if llvrw.ErrAfter == llvrw.invoked {
		return nil, llvrw.Err
	}
	return llvrw, nil
}

// WriteDBPointer implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteDBPointer(string, ObjectID) error {
	llvrw.checkdepth()
	llvrw.invoked = writeDBPointer
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteDateTime implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteDateTime(int64) error {
	llvrw.checkdepth()
	llvrw.invoked = writeDateTime
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteDecimal128 implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteDecimal128(Decimal128) error {
	llvrw.checkdepth()
	llvrw.invoked = writeDecimal128
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteDouble implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteDouble(float64) error {
	llvrw.checkdepth()
	llvrw.invoked = writeDouble
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteInt32 implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteInt32(int32) error {
	llvrw.checkdepth()
	llvrw.invoked = writeInt32
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteInt64 implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteInt64(int64) error {
	llvrw.checkdepth()
	llvrw.invoked = writeInt64
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteJavascript implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteJavascript(string) error {
	llvrw.checkdepth()
	llvrw.invoked = writeJavascript
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteMaxKey implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteMaxKey() error {
	llvrw.checkdepth()
	llvrw.invoked = writeMaxKey
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteMinKey implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteMinKey() error {
	llvrw.checkdepth()
	llvrw.invoked = writeMinKey
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteNull implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteNull() error {
	llvrw.checkdepth()
	llvrw.invoked = writeNull
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteObjectID implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteObjectID(ObjectID) error {
	llvrw.checkdepth()
	llvrw.invoked = writeObjectID
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteRegex implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteRegex(string, string) error {
	llvrw.checkdepth()
	llvrw.invoked = writeRegex
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteString implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteString(string) error {
	llvrw.checkdepth()
	llvrw.invoked = writeString
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteDocument implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteDocument() (DocumentWriter, error) {
	llvrw.checkdepth()
	llvrw.invoked = writeDocument
	if llvrw.ErrAfter == llvrw.invoked {
		return nil, llvrw.Err
	}
	return llvrw, nil
}

// WriteSymbol implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteSymbol(string) error {
	llvrw.checkdepth()
	llvrw.invoked = writeSymbol
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteTimestamp implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteTimestamp(uint32, uint32) error {
	llvrw.checkdepth()
	llvrw.invoked = writeTimestamp
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// WriteUndefined implements the ValueWriter interface.
func (llvrw *valueReaderWriter) WriteUndefined() error {
	llvrw.checkdepth()
	llvrw.invoked = writeUndefined
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}
	return nil
}

// ReadElement implements the DocumentReader interface.
func (llvrw *valueReaderWriter) ReadElement() (string, ValueReader, error) {
	llvrw.checkdepth()
	llvrw.invoked = readElement
	if llvrw.ErrAfter == llvrw.invoked {
		return "", nil, llvrw.Err
	}

	return "", llvrw, nil
}

// WriteDocumentElement implements the DocumentWriter interface.
func (llvrw *valueReaderWriter) WriteDocumentElement(string) (ValueWriter, error) {
	llvrw.checkdepth()
	llvrw.invoked = writeDocumentElement
	if llvrw.ErrAfter == llvrw.invoked {
		return nil, llvrw.Err
	}

	return llvrw, nil
}

// WriteDocumentEnd implements the DocumentWriter interface.
func (llvrw *valueReaderWriter) WriteDocumentEnd() error {
	llvrw.checkdepth()
	llvrw.invoked = writeDocumentEnd
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}

	return nil
}

// ReadValue implements the ArrayReader interface.
func (llvrw *valueReaderWriter) ReadValue() (ValueReader, error) {
	llvrw.checkdepth()
	llvrw.invoked = readValue
	if llvrw.ErrAfter == llvrw.invoked {
		return nil, llvrw.Err
	}

	return llvrw, nil
}

// WriteArrayElement implements the ArrayWriter interface.
func (llvrw *valueReaderWriter) WriteArrayElement() (ValueWriter, error) {
	llvrw.checkdepth()
	llvrw.invoked = writeArrayElement
	if llvrw.ErrAfter == llvrw.invoked {
		return nil, llvrw.Err
	}

	return llvrw, nil
}

// WriteArrayEnd implements the ArrayWriter interface.
func (llvrw *valueReaderWriter) WriteArrayEnd() error {
	llvrw.checkdepth()
	llvrw.invoked = writeArrayEnd
	if llvrw.ErrAfter == llvrw.invoked {
		return llvrw.Err
	}

	return nil
}
