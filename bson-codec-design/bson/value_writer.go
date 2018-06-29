package bson

import (
	"io"

	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/objectid"
)

type valueWriter struct{}

func newValueWriter(io.Writer) *valueWriter { return nil }

func (vw *valueWriter) WriteArray() (ArrayWriter, error) {
	panic("not implemented")
}

func (vw *valueWriter) WriteBinary(b []byte) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteBinaryWithSubtype(b []byte, btype byte) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteBoolean(bool) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteCodeWithScope(code string) (DocumentWriter, error) {
	panic("not implemented")
}

func (vw *valueWriter) WriteDBPointer(ns string, oid objectid.ObjectID) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteDateTime(dt int64) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteDecimal128(decimal.Decimal128) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteDouble(float64) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteInt32(int32) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteInt64(int64) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteJavascript(code string) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteMaxKey() error {
	panic("not implemented")
}

func (vw *valueWriter) WriteMinKey() error {
	panic("not implemented")
}

func (vw *valueWriter) WriteNull() error {
	panic("not implemented")
}

func (vw *valueWriter) WriteObjectID(objectid.ObjectID) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteRegex(pattern string, options string) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteString(string) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteDocument() (DocumentWriter, error) {
	panic("not implemented")
}

func (vw *valueWriter) WriteSymbol(symbol string) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteTimestamp(t uint32, i uint32) error {
	panic("not implemented")
}

func (vw *valueWriter) WriteUndefined() error {
	panic("not implemented")
}

func (vw *valueWriter) WriteDocumentElement(string) (ValueWriter, error) {
	panic("not implemented")
}

func (vw *valueWriter) WriteDocumentEnd() error {
	panic("not implemented")
}

func (vw *valueWriter) WriteArrayElement() (ValueWriter, error) {
	panic("not implemented")
}

func (vw *valueWriter) WriteArrayEnd() error {
	panic("not implemented")
}
