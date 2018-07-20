package bson

import (
	"testing"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

type llvrwInvoked byte

const (
	llvrwNothing llvrwInvoked = iota
	llvrwReadArray
	llvrwReadBinary
	llvrwReadBoolean
	llvrwReadDocument
	llvrwReadCodeWithScope
	llvrwReadDBPointer
	llvrwReadDateTime
	llvrwReadDecimal128
	llvrwReadDouble
	llvrwReadInt32
	llvrwReadInt64
	llvrwReadJavascript
	llvrwReadMaxKey
	llvrwReadMinKey
	llvrwReadNull
	llvrwReadObjectID
	llvrwReadRegex
	llvrwReadString
	llvrwReadSymbol
	llvrwReadTimestamp
	llvrwReadUndefined
	llvrwReadElement
	llvrwReadValue
	llvrwWriteArray
	llvrwWriteBinary
	llvrwWriteBinaryWithSubtype
	llvrwWriteBoolean
	llvrwWriteCodeWithScope
	llvrwWriteDBPointer
	llvrwWriteDateTime
	llvrwWriteDecimal128
	llvrwWriteDouble
	llvrwWriteInt32
	llvrwWriteInt64
	llvrwWriteJavascript
	llvrwWriteMaxKey
	llvrwWriteMinKey
	llvrwWriteNull
	llvrwWriteObjectID
	llvrwWriteRegex
	llvrwWriteString
	llvrwWriteDocument
	llvrwWriteSymbol
	llvrwWriteTimestamp
	llvrwWriteUndefined
	llvrwWriteDocumentElement
	llvrwWriteDocumentEnd
	llvrwWriteArrayElement
	llvrwWriteArrayEnd
)

type llValueReaderWriter struct {
	t        *testing.T
	invoked  llvrwInvoked
	readval  interface{}
	bsontype Type
	err      error
	errAfter llvrwInvoked // error after this method is called
}

func (llvrw *llValueReaderWriter) Type() Type {
	return llvrw.bsontype
}

func (llvrw *llValueReaderWriter) Skip() error {
	panic("not implemented")
}

func (llvrw *llValueReaderWriter) ReadArray() (ArrayReader, error) {
	llvrw.invoked = llvrwReadArray
	if llvrw.errAfter == llvrw.invoked {
		return nil, llvrw.err
	}

	return llvrw, nil
}

func (llvrw *llValueReaderWriter) ReadBinary() (b []byte, btype byte, err error) {
	llvrw.invoked = llvrwReadBinary
	if llvrw.errAfter == llvrw.invoked {
		return nil, 0x00, llvrw.err
	}

	bin, ok := llvrw.readval.(Binary)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadBinary: %T", llvrw.readval)
		return nil, 0x00, nil
	}
	return bin.Data, bin.Subtype, nil
}

func (llvrw *llValueReaderWriter) ReadBoolean() (bool, error) {
	llvrw.invoked = llvrwReadBoolean
	if llvrw.errAfter == llvrw.invoked {
		return false, llvrw.err
	}

	b, ok := llvrw.readval.(bool)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadBoolean: %T", llvrw.readval)
		return false, nil
	}

	return b, llvrw.err
}

func (llvrw *llValueReaderWriter) ReadDocument() (DocumentReader, error) {
	llvrw.invoked = llvrwReadDocument
	if llvrw.errAfter == llvrw.invoked {
		return nil, llvrw.err
	}

	return llvrw, nil
}

func (llvrw *llValueReaderWriter) ReadCodeWithScope() (code string, dr DocumentReader, err error) {
	llvrw.invoked = llvrwReadCodeWithScope
	if llvrw.errAfter == llvrw.invoked {
		return "", nil, llvrw.err
	}

	return "", llvrw, nil
}

func (llvrw *llValueReaderWriter) ReadDBPointer() (ns string, oid objectid.ObjectID, err error) {
	llvrw.invoked = llvrwReadDBPointer
	if llvrw.errAfter == llvrw.invoked {
		return "", objectid.ObjectID{}, llvrw.err
	}

	db, ok := llvrw.readval.(DBPointer)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadDBPointer: %T", llvrw.readval)
		return "", objectid.ObjectID{}, nil
	}

	return db.DB, db.Pointer, nil
}

func (llvrw *llValueReaderWriter) ReadDateTime() (int64, error) {
	llvrw.invoked = llvrwReadDateTime
	if llvrw.errAfter == llvrw.invoked {
		return 0, llvrw.err
	}

	dt, ok := llvrw.readval.(int64)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadDateTime: %T", llvrw.readval)
		return 0, nil
	}

	return dt, nil
}

func (llvrw *llValueReaderWriter) ReadDecimal128() (decimal.Decimal128, error) {
	llvrw.invoked = llvrwReadDecimal128
	if llvrw.errAfter == llvrw.invoked {
		return decimal.Decimal128{}, llvrw.err
	}

	d128, ok := llvrw.readval.(decimal.Decimal128)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadDecimal128: %T", llvrw.readval)
		return decimal.Decimal128{}, nil
	}

	return d128, nil
}

func (llvrw *llValueReaderWriter) ReadDouble() (float64, error) {
	llvrw.invoked = llvrwReadDouble
	if llvrw.errAfter == llvrw.invoked {
		return 0, llvrw.err
	}

	f64, ok := llvrw.readval.(float64)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadDouble: %T", llvrw.readval)
		return 0, nil
	}

	return f64, nil
}

func (llvrw *llValueReaderWriter) ReadInt32() (int32, error) {
	llvrw.invoked = llvrwReadInt32
	if llvrw.errAfter == llvrw.invoked {
		return 0, llvrw.err
	}

	i32, ok := llvrw.readval.(int32)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadInt32: %T", llvrw.readval)
		return 0, nil
	}

	return i32, nil
}

func (llvrw *llValueReaderWriter) ReadInt64() (int64, error) {
	llvrw.invoked = llvrwReadInt64
	if llvrw.errAfter == llvrw.invoked {
		return 0, llvrw.err
	}
	i64, ok := llvrw.readval.(int64)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadInt64: %T", llvrw.readval)
		return 0, nil
	}

	return i64, nil
}

func (llvrw *llValueReaderWriter) ReadJavascript() (code string, err error) {
	llvrw.invoked = llvrwReadJavascript
	if llvrw.errAfter == llvrw.invoked {
		return "", llvrw.err
	}
	js, ok := llvrw.readval.(string)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadJavascript: %T", llvrw.readval)
		return "", nil
	}

	return js, nil
}

func (llvrw *llValueReaderWriter) ReadMaxKey() error {
	llvrw.invoked = llvrwReadMaxKey
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}

	return nil
}

func (llvrw *llValueReaderWriter) ReadMinKey() error {
	llvrw.invoked = llvrwReadMinKey
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}

	return nil
}

func (llvrw *llValueReaderWriter) ReadNull() error {
	llvrw.invoked = llvrwReadNull
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}

	return nil
}

func (llvrw *llValueReaderWriter) ReadObjectID() (objectid.ObjectID, error) {
	llvrw.invoked = llvrwReadObjectID
	if llvrw.errAfter == llvrw.invoked {
		return objectid.ObjectID{}, llvrw.err
	}
	oid, ok := llvrw.readval.(objectid.ObjectID)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadObjectID: %T", llvrw.readval)
		return objectid.ObjectID{}, nil
	}

	return oid, nil
}

func (llvrw *llValueReaderWriter) ReadRegex() (pattern string, options string, err error) {
	llvrw.invoked = llvrwReadRegex
	if llvrw.errAfter == llvrw.invoked {
		return "", "", llvrw.err
	}
	rgx, ok := llvrw.readval.(Regex)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadRegex: %T", llvrw.readval)
		return "", "", nil
	}

	return rgx.Pattern, rgx.Options, nil
}

func (llvrw *llValueReaderWriter) ReadString() (string, error) {
	llvrw.invoked = llvrwReadString
	if llvrw.errAfter == llvrw.invoked {
		return "", llvrw.err
	}
	str, ok := llvrw.readval.(string)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadString: %T", llvrw.readval)
		return "", nil
	}

	return str, nil
}

func (llvrw *llValueReaderWriter) ReadSymbol() (symbol string, err error) {
	llvrw.invoked = llvrwReadSymbol
	if llvrw.errAfter == llvrw.invoked {
		return "", llvrw.err
	}
	symb, ok := llvrw.readval.(Symbol)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadSymbol: %T", llvrw.readval)
		return "", nil
	}

	return string(symb), nil
}

func (llvrw *llValueReaderWriter) ReadTimestamp() (t uint32, i uint32, err error) {
	llvrw.invoked = llvrwReadTimestamp
	if llvrw.errAfter == llvrw.invoked {
		return 0, 0, llvrw.err
	}
	ts, ok := llvrw.readval.(Timestamp)
	if !ok {
		llvrw.t.Errorf("Incorrect type provided for return value of ReadTimestamp: %T", llvrw.readval)
		return 0, 0, nil
	}

	return ts.T, ts.I, nil
}

func (llvrw *llValueReaderWriter) ReadUndefined() error {
	llvrw.invoked = llvrwReadUndefined
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}

	return nil
}

func (llvrw *llValueReaderWriter) WriteArray() (ArrayWriter, error) {
	llvrw.invoked = llvrwWriteArray
	if llvrw.errAfter == llvrw.invoked {
		return nil, llvrw.err
	}
	return llvrw, nil
}

func (llvrw *llValueReaderWriter) WriteBinary(b []byte) error {
	llvrw.invoked = llvrwWriteBinary
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteBinaryWithSubtype(b []byte, btype byte) error {
	llvrw.invoked = llvrwWriteBinaryWithSubtype
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteBoolean(bool) error {
	llvrw.invoked = llvrwWriteBoolean
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteCodeWithScope(code string) (DocumentWriter, error) {
	llvrw.invoked = llvrwWriteCodeWithScope
	if llvrw.errAfter == llvrw.invoked {
		return nil, llvrw.err
	}
	return llvrw, nil
}

func (llvrw *llValueReaderWriter) WriteDBPointer(ns string, oid objectid.ObjectID) error {
	llvrw.invoked = llvrwWriteDBPointer
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteDateTime(dt int64) error {
	llvrw.invoked = llvrwWriteDateTime
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteDecimal128(decimal.Decimal128) error {
	llvrw.invoked = llvrwWriteDecimal128
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteDouble(float64) error {
	llvrw.invoked = llvrwWriteDouble
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteInt32(int32) error {
	llvrw.invoked = llvrwWriteInt32
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteInt64(int64) error {
	llvrw.invoked = llvrwWriteInt64
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteJavascript(code string) error {
	llvrw.invoked = llvrwWriteJavascript
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteMaxKey() error {
	llvrw.invoked = llvrwWriteMaxKey
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteMinKey() error {
	llvrw.invoked = llvrwWriteMinKey
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteNull() error {
	llvrw.invoked = llvrwWriteNull
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteObjectID(objectid.ObjectID) error {
	llvrw.invoked = llvrwWriteObjectID
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteRegex(pattern string, options string) error {
	llvrw.invoked = llvrwWriteRegex
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteString(string) error {
	llvrw.invoked = llvrwWriteString
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteDocument() (DocumentWriter, error) {
	llvrw.invoked = llvrwWriteDocument
	if llvrw.errAfter == llvrw.invoked {
		return nil, llvrw.err
	}
	return llvrw, nil
}

func (llvrw *llValueReaderWriter) WriteSymbol(symbol string) error {
	llvrw.invoked = llvrwWriteSymbol
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteTimestamp(t uint32, i uint32) error {
	llvrw.invoked = llvrwWriteTimestamp
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) WriteUndefined() error {
	llvrw.invoked = llvrwWriteUndefined
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}
	return nil
}

func (llvrw *llValueReaderWriter) ReadElement() (string, ValueReader, error) {
	llvrw.invoked = llvrwReadElement
	if llvrw.errAfter == llvrw.invoked {
		return "", nil, llvrw.err
	}

	return "", llvrw, nil
}

func (llvrw *llValueReaderWriter) WriteDocumentElement(string) (ValueWriter, error) {
	llvrw.invoked = llvrwWriteDocumentElement
	if llvrw.errAfter == llvrw.invoked {
		return nil, llvrw.err
	}

	return llvrw, nil
}

func (llvrw *llValueReaderWriter) WriteDocumentEnd() error {
	llvrw.invoked = llvrwWriteDocumentEnd
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}

	return nil
}

func (llvrw *llValueReaderWriter) ReadValue() (ValueReader, error) {
	llvrw.invoked = llvrwReadValue
	if llvrw.errAfter == llvrw.invoked {
		return nil, llvrw.err
	}

	return llvrw, nil
}

func (llvrw *llValueReaderWriter) WriteArrayElement() (ValueWriter, error) {
	llvrw.invoked = llvrwWriteArrayElement
	if llvrw.errAfter == llvrw.invoked {
		return nil, llvrw.err
	}

	return llvrw, nil
}

func (llvrw *llValueReaderWriter) WriteArrayEnd() error {
	llvrw.invoked = llvrwWriteArrayEnd
	if llvrw.errAfter == llvrw.invoked {
		return llvrw.err
	}

	return nil
}
