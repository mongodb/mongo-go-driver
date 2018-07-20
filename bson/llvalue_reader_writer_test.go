package bson

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

type llvrwInvoked byte

const (
	_ llvrwInvoked = iota
	llvrwReadDouble
	llvrwReadInt32
	llvrwReadInt64
	llvrwReadBoolean
	llvrwReadString
	llvrwWriteDouble
	llvrwWriteInt32
	llvrwWriteInt64
	llvrwWriteBoolean
	llvrwWriteString
)

type llValueReaderWriter struct {
	t        *testing.T
	invoked  interface{}
	readval  interface{}
	bsontype Type
	err      error
}

func (llvrw *llValueReaderWriter) Type() Type {
	return llvrw.bsontype
}

func (llvrw *llValueReaderWriter) Skip() error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) ReadArray() (ArrayReader, error) {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) ReadBinary() (b []byte, btype byte, err error) {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) ReadBoolean() (bool, error) {
	llvrw.invoked = llvrwReadBoolean
	b, ok := llvrw.readval.(bool)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadBoolean: %T", llvrw.readval)
		return false, nil
	}

	return b, llvrw.err
}

func (llvrw *llValueReaderWriter) ReadDocument() (DocumentReader, error) {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) ReadCodeWithScope() (code string, dr DocumentReader, err error) {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) ReadDBPointer() (ns string, oid objectid.ObjectID, err error) {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) ReadDateTime() (int64, error) {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) ReadDecimal128() (decimal.Decimal128, error) {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) ReadDouble() (float64, error) {
	llvrw.invoked = llvrwReadDouble
	f64, ok := llvrw.readval.(float64)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadDouble: %T", llvrw.readval)
		return 0, nil
	}

	return f64, llvrw.err
}

func (llvrw *llValueReaderWriter) ReadInt32() (int32, error) {
	llvrw.invoked = llvrwReadInt32
	i32, ok := llvrw.readval.(int32)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadInt32: %T", llvrw.readval)
		return 0, nil
	}

	return i32, llvrw.err
}

func (llvrw *llValueReaderWriter) ReadInt64() (int64, error) {
	llvrw.invoked = llvrwReadInt64
	i64, ok := llvrw.readval.(int64)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadInt64: %T", llvrw.readval)
		return 0, nil
	}

	return i64, llvrw.err
}

func (llvrw *llValueReaderWriter) ReadJavascript() (code string, err error) {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) ReadMaxKey() error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) ReadMinKey() error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) ReadNull() error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) ReadObjectID() (objectid.ObjectID, error) {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) ReadRegex() (pattern string, options string, err error) {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) ReadString() (string, error) {
	llvrw.invoked = llvrwReadString
	str, ok := llvrw.readval.(string)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadString: %T", llvrw.readval)
		return "", nil
	}

	return str, llvrw.err
}

func (llvrw *llValueReaderWriter) ReadSymbol() (symbol string, err error) {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) ReadTimestamp() (t uint32, i uint32, err error) {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) ReadUndefined() error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) WriteArray() (ArrayWriter, error) {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) WriteBinary(b []byte) error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) WriteBinaryWithSubtype(b []byte, btype byte) error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) WriteBoolean(bool) error {
	llvrw.invoked = llvrwWriteBoolean
	return nil
}

func (llvrw *llValueReaderWriter) WriteCodeWithScope(code string) (DocumentWriter, error) {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) WriteDBPointer(ns string, oid objectid.ObjectID) error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) WriteDateTime(dt int64) error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) WriteDecimal128(decimal.Decimal128) error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) WriteDouble(float64) error {
	llvrw.invoked = llvrwWriteDouble
	return nil
}

func (llvrw *llValueReaderWriter) WriteInt32(int32) error {
	llvrw.invoked = llvrwWriteInt32
	return nil
}

func (llvrw *llValueReaderWriter) WriteInt64(int64) error {
	llvrw.invoked = llvrwWriteInt64
	return nil
}

func (llvrw *llValueReaderWriter) WriteJavascript(code string) error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) WriteMaxKey() error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) WriteMinKey() error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) WriteNull() error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) WriteObjectID(objectid.ObjectID) error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) WriteRegex(pattern string, options string) error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) WriteString(string) error {
	llvrw.invoked = llvrwWriteString
	return nil
}

func (llvrw *llValueReaderWriter) WriteDocument() (DocumentWriter, error) {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) WriteSymbol(symbol string) error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) WriteTimestamp(t uint32, i uint32) error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) WriteUndefined() error {
	panic("not implemented")
}
