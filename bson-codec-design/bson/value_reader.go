package bson

import (
	"io"

	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/objectid"
)

// valueReader is for reading BSON values.
type valueReader struct{}

func newValueReader(io.Reader) *valueReader { return nil }

func (vr *valueReader) Type() Type {
	panic("not implemented")
}

func (vr *valueReader) Skip() error {
	panic("not implemented")
}

func (vr *valueReader) ReadArray() (ArrayReader, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadBinary() (b []byte, btype byte, err error) {
	panic("not implemented")
}

func (vr *valueReader) ReadBoolean() (bool, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadDocument() (DocumentReader, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadCodeWithScope() (code string, dr DocumentReader, err error) {
	panic("not implemented")
}

func (vr *valueReader) ReadDBPointer() (ns string, oid objectid.ObjectID, err error) {
	panic("not implemented")
}

func (vr *valueReader) ReadDateTime() (int64, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadDecimal128() (decimal.Decimal128, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadDouble() (float64, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadInt32() (int32, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadInt64() (int64, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadJavascript() (code string, err error) {
	panic("not implemented")
}

func (vr *valueReader) ReadMaxKey() error {
	panic("not implemented")
}

func (vr *valueReader) ReadMinKey() error {
	panic("not implemented")
}

func (vr *valueReader) ReadNull() error {
	panic("not implemented")
}

func (vr *valueReader) ReadObjectID() (objectid.ObjectID, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadRegex() (pattern string, options string, err error) {
	panic("not implemented")
}

func (vr *valueReader) ReadString() (string, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadSymbol() (symbol string, err error) {
	panic("not implemented")
}

func (vr *valueReader) ReadTimeStamp(t uint32, i uint32, err error) {
	panic("not implemented")
}

func (vr *valueReader) ReadUndefined() error {
	panic("not implemented")
}

func (vr *valueReader) ReadElement() (string, ValueReader, error) {
	panic("not implemented")
}

func (vr *valueReader) ReadValue() (ValueReader, error) {
	panic("not implemented")
}
