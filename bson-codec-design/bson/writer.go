package bson

import (
	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/objectid"
)

// ArrayWriter is the interface used to create a BSON or BSON adjacent array.
// Callers must ensure they call WriteArrayEnd when they have finished creating
// the array.
type ArrayWriter interface {
	WriteArrayElement() (ValueWriter, error)
	WriteArrayEnd() error
}

// DocumentWriter is the interface used to create a BSON or BSON adjacent
// document. Callers must ensure they call WriteDocumentEnd when they have
// finished creating the document.
type DocumentWriter interface {
	WriteDocumentElement(string) (ValueWriter, error)
	WriteDocumentEnd() error
}

// ValueWriter is the interface used to write BSON values. Implementations of
// this interface handle creating BSON or BSON adjacent representations of the
// values.
type ValueWriter interface {
	WriteArray() (ArrayWriter, error)
	WriteBinary(b []byte) error
	WriteBinaryWithSubtype(b []byte, btype byte) error
	WriteBoolean(bool) error
	WriteCodeWithScope(code string) (DocumentWriter, error)
	WriteDBPointer(ns string, oid objectid.ObjectID) error
	WriteDateTime(dt int64) error
	WriteDecimal128(decimal.Decimal128) error
	WriteDouble(float64) error
	WriteInt32(int32) error
	WriteInt64(int64) error
	WriteJavascript(code string) error
	WriteMaxKey() error
	WriteMinKey() error
	WriteNull() error
	WriteObjectID(objectid.ObjectID) error
	WriteRegex(pattern, options string) error
	WriteString(string) error
	WriteDocument() (DocumentWriter, error)
	WriteSymbol(symbol string) error
	WriteTimestamp(t, i uint32) error
	WriteUndefined() error
}

type writer []byte

func (w *writer) Write(p []byte) (int, error) {
	index := len(*w)
	return w.WriteAt(p, int64(index))
}

func (w *writer) WriteAt(p []byte, off int64) (int, error) {
	if int64(len(p)) > int64(cap(*w))-off {
		buf := make([]byte, 2*cap(*w)+len(p))
		copy(buf, *w)
		*w = buf
	}

	*w = []byte(*w)[:off+int64(len(p))]
	copy([]byte(*w)[off:], p)
	return len(p), nil
	return 0, nil
}
