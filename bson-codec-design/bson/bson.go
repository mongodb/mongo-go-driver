package bson

import (
	"io"
	"reflect"
	"time"

	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson-codec-design/bson/objectid"
)

type Codec interface {
	EncodeValue(Registry, ValueWriter, interface{}) error
	DecodeValue(Registry, ValueReader, interface{}) error
}

type ArrayReader ValueReader

type DocumentReader interface {
	ReadElement() (string, ValueReader, error)
}

type ValueReader interface {
	ReadBytes([]byte) error
	ReadSlice() ([]byte, error)
	Size() int
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
	ReadTime() (time.Time, error)
	ReadTimeStamp(t, i uint32, err error)
	ReadUndefined() error
}

type ArrayWriter interface {
	WriteElement() (ValueWriter, error)
	WriteEndArray() error
}

type DocumentWriter interface {
	WriteElement(string) (ValueWriter, error)
	WriteEndDocument() error
}

type ValueWriter interface {
	WriteTo(io.Writer) (int64, error)

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
	WriteTime(time.Time) error
	WriteTimestamp(t, i uint32) error
	WriteUndefined() error
}

type writer struct{}

type Registry struct {
	tr *TypeRegistry
	ir *InterfaceRegistry

	dm Codec // default map codec
	ds Codec // default struct codec
}

func (r *Registry) Lookup(reflect.Type) (Codec, error) { return nil, nil }

type TypeRegistry struct{}

func (tr *TypeRegistry) Lookup(reflect.Type) (Codec, error) { return nil, nil }

type InterfaceRegistry struct{}

func (ir *InterfaceRegistry) Lookup(reflect.Type) (Codec, error) { return nil, nil }

func Marshal(interface{}) ([]byte, error)                       { return nil, nil }
func MarshalWithRegistry(Registry, interface{}) ([]byte, error) { return nil, nil }
func Unmarshal([]byte, interface{}) error                       { return nil }
func UnmarshalWithRegistry(Registry, []byte, interface{}) error { return nil }
