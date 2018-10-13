package bsoncore

import (
	"time"

	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// Value represents a BSON value with a type and raw bytes.
type Value struct {
	Type bsontype.Type
	Data []byte
}

// Validate attempts to validate the value.
func (Value) Validate() error { return nil }

// IsNumber returns true if the type of v is a numeric BSON type.
func (Value) IsNumber() bool { return false }

func (Value) Equal(v2 Value) bool { return false }

func (Value) Double() float64                                { return 0 }
func (Value) DoubleOK() (float64, bool)                      { return 0, false }
func (Value) StringValue() string                            { return "" }
func (Value) StringValueOK() (string, bool)                  { return "", false }
func (Value) Document() Document                             { return nil }
func (Value) DocumentOK() (Document, bool)                   { return nil, false }
func (Value) Array() Array                                   { return nil }
func (Value) ArrayOK() (Array, bool)                         { return nil, false }
func (Value) Binary() (subtype byte, data []byte)            { return 0x00, nil }
func (Value) BinaryOK() (subtype byte, data []byte, ok bool) { return 0x00, nil, false }
func (Value) ObjectID() objectid.ObjectID                    { return objectid.ObjectID{} }
func (Value) ObjectIDOK() (objectid.ObjectID, bool)          { return objectid.ObjectID{}, false }
func (Value) Boolean() bool                                  { return false }
func (Value) BooleanOK() (bool, bool)                        { return false, false }
func (Value) DateTime() int64                                { return 0 }
func (Value) DateTimeOK() (int64, bool)                      { return 0, false }
func (Value) Time() time.Time                                { return time.Time{} }
func (Value) TimeOK() (time.Time, bool)                      { return time.Time{}, false }
func (Value) Regex() (pattern, options string)               { return "", "" }
func (Value) RegexOK() (pattern, options string, ok bool)    { return "", "", false }
func (Value) DBPointer() (string, objectid.ObjectID)         { return "", objectid.ObjectID{} }
func (Value) DBPointerOK() (string, objectid.ObjectID, bool) { return "", objectid.ObjectID{}, false }
func (Value) JavaScript() string                             { return "" }
func (Value) JavaScriptOK() (string, bool)                   { return "", false }
func (Value) Symbol() string                                 { return "" }
func (Value) SymbolOK() (string, bool)                       { return "", false }
func (Value) CodeWithScope() (string, Document)              { return "", nil }
func (Value) CodeWithScopeOK() (string, Document, bool)      { return "", nil, false }
func (Value) Int32() int32                                   { return 0 }
func (Value) Int32OK() (int32, bool)                         { return 0, false }
func (Value) Timestamp() (t, i uint32)                       { return 0, 0 }
func (Value) TimestampOK() (t, i uint32, ok bool)            { return 0, 0, false }
func (Value) Int64() int64                                   { return 0 }
func (Value) Int64OK() (int64, bool)                         { return 0, false }
func (Value) Decimal128() decimal.Decimal128                 { return decimal.Decimal128{} }
func (Value) Decimal128OK() (decimal.Decimal128, bool)       { return decimal.Decimal128{}, false }
