package bsoncodec

import (
	"errors"
	"io"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

type extJSONValueWriter struct{}

// NewExtJSONValueWriter creates a ValueWriter that writes Extended JSON to w.
func NewExtJSONValueWriter(w io.Writer) (ValueWriter, error) {
	if w == nil {
		return nil, errors.New("cannot create a ValueWriter from a nil io.Writer")
	}

	return newExtJSONWriter(w), nil
}

func newExtJSONWriter(io.Writer) *extJSONValueWriter { return nil }

func (ejvw *extJSONValueWriter) WriteArray() (ArrayWriter, error) {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteBinary(b []byte) error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteBinaryWithSubtype(b []byte, btype byte) error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteBoolean(bool) error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteCodeWithScope(code string) (DocumentWriter, error) {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteDBPointer(ns string, oid objectid.ObjectID) error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteDateTime(dt int64) error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteDecimal128(decimal.Decimal128) error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteDouble(float64) error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteInt32(int32) error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteInt64(int64) error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteJavascript(code string) error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteMaxKey() error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteMinKey() error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteNull() error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteObjectID(objectid.ObjectID) error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteRegex(pattern string, options string) error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteString(string) error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteDocument() (DocumentWriter, error) {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteSymbol(symbol string) error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteTimestamp(t uint32, i uint32) error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteUndefined() error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteDocumentElement(string) (ValueWriter, error) {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteDocumentEnd() error {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteArrayElement() (ValueWriter, error) {
	panic("not implemented")
}

func (ejvw *extJSONValueWriter) WriteArrayEnd() error {
	panic("not implemented")
}
