package bsoncodec

import (
	"io"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// extJSONValueReader is for reading extended JSON.
type extJSONValueReader struct{}

func newExtJSONValueReader(io.Reader) *extJSONValueReader { return nil }

func (ejvr *extJSONValueReader) Type() bson.Type {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) Skip() error {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadArray() (ArrayReader, error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadBinary() (b []byte, btype byte, err error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadBoolean() (bool, error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadDocument() (DocumentReader, error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadCodeWithScope() (code string, dr DocumentReader, err error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadDBPointer() (ns string, oid objectid.ObjectID, err error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadDateTime() (int64, error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadDecimal128() (decimal.Decimal128, error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadDouble() (float64, error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadInt32() (int32, error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadInt64() (int64, error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadJavascript() (code string, err error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadMaxKey() error {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadMinKey() error {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadNull() error {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadObjectID() (objectid.ObjectID, error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadRegex() (pattern string, options string, err error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadString() (string, error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadSymbol() (symbol string, err error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadTimeStamp(t uint32, i uint32, err error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadUndefined() error {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadElement() (string, ValueReader, error) {
	panic("not implemented")
}

func (ejvr *extJSONValueReader) ReadValue() (ValueReader, error) {
	panic("not implemented")
}
