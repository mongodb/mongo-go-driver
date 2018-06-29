package bson

import (
	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/objectid"
)

type documentValueWriter struct{}

func newDocumentValueWriter(*Document) *documentValueWriter { return nil }

func (dvw *documentValueWriter) WriteArray() (ArrayWriter, error) {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteBinary(b []byte) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteBinaryWithSubtype(b []byte, btype byte) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteBoolean(bool) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteCodeWithScope(code string) (DocumentWriter, error) {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteDBPointer(ns string, oid objectid.ObjectID) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteDateTime(dt int64) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteDecimal128(decimal.Decimal128) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteDouble(float64) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteInt32(int32) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteInt64(int64) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteJavascript(code string) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteMaxKey() error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteMinKey() error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteNull() error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteObjectID(objectid.ObjectID) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteRegex(pattern string, options string) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteString(string) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteDocument() (DocumentWriter, error) {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteSymbol(symbol string) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteTimestamp(t uint32, i uint32) error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteUndefined() error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteDocumentElement(string) (ValueWriter, error) {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteDocumentEnd() error {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteArrayElement() (ValueWriter, error) {
	panic("not implemented")
}

func (dvw *documentValueWriter) WriteArrayEnd() error {
	panic("not implemented")
}
