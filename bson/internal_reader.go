package bson

import (
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// ArrayReader is implemented by types that allow reading values from a BSON
// array.
type ArrayReader interface {
	ReadValue() (ValueReader, error)
}

// DocumentReader is implemented by types that allow reading elements from a
// BSON document.
type DocumentReader interface {
	ReadElement() (string, ValueReader, error)
}

// ValueReader is a generic interface used to read values from BSON. This type
// is implemented by several types with different underlying representations of
// BSON, such as a bson.Document, raw BSON bytes, or extended JSON.
type ValueReader interface {
	Type() Type
	Skip() error

	ReadArray() (ArrayReader, error)
	ReadBinary() (b []byte, btype byte, err error)
	ReadBoolean() (bool, error)
	ReadDocument() (DocumentReader, error)
	ReadCodeWithScope() (code string, dr DocumentReader, err error)
	ReadDBPointer() (ns string, oid objectid.ObjectID, err error)
	ReadDateTime() (int64, error)
	ReadDecimal128() (decimal.Decimal128, error)
	ReadDouble() (float64, error)
	ReadInt32() (int32, error)
	ReadInt64() (int64, error)
	ReadJavascript() (code string, err error)
	ReadMaxKey() error
	ReadMinKey() error
	ReadNull() error
	ReadObjectID() (objectid.ObjectID, error)
	ReadRegex() (pattern, options string, err error)
	ReadString() (string, error)
	ReadSymbol() (symbol string, err error)
	ReadTimestamp() (t, i uint32, err error)
	ReadUndefined() error
}

// ElementAsBSON will retrieve the next value from vr and return it as a kind
// byte and the value as a slice of bytes.
func ElementAsBSON(vr ValueReader) (kind byte, data []byte, err error) {
	return
}

type reader struct {
	b   []byte
	idx int64
}

func (r *reader) Read(p []byte) (int, error) {
	return 0, nil
}

func (r *reader) ReadAt(p []byte, off int64) (int, error) {
	return 0, nil
}
