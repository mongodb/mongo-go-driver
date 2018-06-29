package bson

import (
	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/objectid"
)

// documentValueReader is for reading *Document.
type documentValueReader struct{}

func newDocumentValueReader(*Document) *documentValueReader { return nil }

func (dvr *documentValueReader) Type() Type {
	panic("not implemented")
}

func (dvr *documentValueReader) Skip() error {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadArray() (ArrayReader, error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadBinary() (b []byte, btype byte, err error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadBoolean() (bool, error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadDocument() (DocumentReader, error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadCodeWithScope() (code string, dr DocumentReader, err error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadDBPointer() (ns string, oid objectid.ObjectID, err error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadDateTime() (int64, error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadDecimal128() (decimal.Decimal128, error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadDouble() (float64, error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadInt32() (int32, error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadInt64() (int64, error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadJavascript() (code string, err error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadMaxKey() error {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadMinKey() error {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadNull() error {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadObjectID() (objectid.ObjectID, error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadRegex() (pattern string, options string, err error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadString() (string, error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadSymbol() (symbol string, err error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadTimeStamp(t uint32, i uint32, err error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadUndefined() error {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadElement() (string, ValueReader, error) {
	panic("not implemented")
}

func (dvr *documentValueReader) ReadValue() (ValueReader, error) {
	panic("not implemented")
}
