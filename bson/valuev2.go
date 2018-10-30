package bson

import (
	"bytes"
	"encoding/binary"
	"math"
	"time"

	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// Valuev2 represents a BSON value.
type Valuev2 struct {
	// NOTE: The bootstrap is a small amount of space that'll be on the stack. At 15 bytes this
	// doesn't make this type any larger, since there are 7 bytes of padding and we want an int64 to
	// store small values (e.g. boolean, double, int64, etc...). The primitive property is where all
	// of the larger values go. They will use either Go primitives or the *Primitive types.
	t         bsontype.Type
	bootstrap [15]byte
	primitive interface{}
}

func (v Valuev2) string() string {
	if v.primitive != nil {
		return v.primitive.(string)
	}
	// The string will either end with a null byte or it fills the entire bootstrap space.
	idx := bytes.IndexByte(v.bootstrap[:], 0x00)
	if idx == -1 {
		idx = 15
	}
	return string(v.bootstrap[:idx])
}

func (v Valuev2) writestring(str string) Valuev2 {
	switch {
	case len(str) < 16:
		copy(v.bootstrap[:], str)
	default:
		v.primitive = str
	}
	return v
}

func (v Valuev2) i64() int64 {
	return int64(v.bootstrap[0]) | int64(v.bootstrap[1])<<8 | int64(v.bootstrap[2])<<16 |
		int64(v.bootstrap[3])<<24 | int64(v.bootstrap[4])<<32 | int64(v.bootstrap[5])<<40 |
		int64(v.bootstrap[6])<<48 | int64(v.bootstrap[7])<<56
}

func (v Valuev2) writei64(i64 int64) Valuev2 {
	v.bootstrap[0] = byte(i64)
	v.bootstrap[1] = byte(i64 >> 8)
	v.bootstrap[2] = byte(i64 >> 16)
	v.bootstrap[3] = byte(i64 >> 24)
	v.bootstrap[4] = byte(i64 >> 32)
	v.bootstrap[5] = byte(i64 >> 40)
	v.bootstrap[6] = byte(i64 >> 48)
	v.bootstrap[7] = byte(i64 >> 56)
	return v
}

// IsZero returns true if this value is zero.
func (v Valuev2) IsZero() bool { return v.t == bsontype.Type(0) }

// Interface returns the Go value of this Value as an empty interface.
//
// This method will return nil if it is empty, otherwise it will return a Go primitive or a
// *Primitive instance.
func (v Valuev2) Interface() interface{} {
	switch v.Type() {
	case TypeDouble:
		return v.Double()
	case TypeString:
		return v.StringValue()
	case TypeEmbeddedDocument:
		return v.Document()
	case TypeArray:
		return v.Array()
	case TypeBinary:
		return v.Binary()
	case TypeUndefined:
		return UndefinedPrimitive{}
	case TypeObjectID:
		return v.ObjectID()
	case TypeBoolean:
		return v.Boolean()
	case TypeDateTime:
		return v.DateTime()
	case TypeNull:
		return NullPrimitive{}
	case TypeRegex:
		return v.Regex()
	case TypeDBPointer:
		return v.DBPointer()
	case TypeJavaScript:
		return v.JavaScript()
	case TypeSymbol:
		return v.Symbol()
	case TypeCodeWithScope:
		return v.CodeWithScope()
	case TypeInt32:
		return v.Int32()
	case TypeTimestamp:
		return v.Timestamp()
	case TypeInt64:
		return v.Int64()
	case TypeDecimal128:
		return v.Decimal128()
	case TypeMinKey:
		return MinKeyPrimitive{}
	case TypeMaxKey:
		return MaxKeyPrimitive{}
	default:
		return nil
	}
}

// Type returns the BSON type of this value.
func (v Valuev2) Type() bsontype.Type { return v.t }

// IsNumber returns true if the type of v is a numberic BSON type.
func (v Valuev2) IsNumber() bool {
	switch v.Type() {
	case TypeDouble, TypeInt32, TypeInt64, TypeDecimal128:
		return true
	default:
		return false
	}
}

// Double returns the BSON double value the Value represents. It panics if the value is a BSON type
// other than double.
func (v Valuev2) Double() float64 {
	if v.t != bsontype.Double {
		panic(ElementTypeError{"bson.Value.Double", v.t})
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(v.bootstrap[0:8]))
}

// DoubleOK is the same as Double, but returns a boolean instead of panicking.
func (v Valuev2) DoubleOK() (float64, bool) {
	if v.t != TypeDouble {
		return 0, false
	}
	return v.Double(), true
}

// StringValue returns the BSON string the Value represents. It panics if the value is a BSON type
// other than string.
//
// NOTE: This method is called StringValue to avoid it implementing the
// fmt.Stringer interface.
func (v Valuev2) StringValue() string {
	if v.t != bsontype.String {
		panic(ElementTypeError{"bson.Value.StringValue", v.t})
	}
	return v.string()
}

// StringValueOK is the same as StringValue, but returns a boolean instead of
// panicking.
func (v Valuev2) StringValueOK() (string, bool) {
	if v.t != bsontype.String {
		return "", false
	}
	return v.StringValue(), true
}

// Document returns the BSON embedded document value the Value represents. It panics if the value
// is a BSON type other than embedded document.
func (v Valuev2) Document() *Document {
	if v.t != bsontype.EmbeddedDocument {
		panic(ElementTypeError{"bson.Value.Document", v.t})
	}
	return v.primitive.(*Document)
}

// DocumentOK is the same as Document, except it returns a boolean
// instead of panicking.
func (v Valuev2) DocumentOK() (*Document, bool) {
	if v.t != bsontype.EmbeddedDocument {
		return nil, false
	}
	return v.Document(), true
}

// Array returns the BSON array value the Value represents. It panics if the value is a BSON type
// other than array.
func (v Valuev2) Array() *Array {
	if v.t != bsontype.Array {
		panic(ElementTypeError{"bson.Value.Array", v.t})
	}
	return v.primitive.(*Array)
}

// ArrayOK is the same as Array, except it returns a boolean
// instead of panicking.
func (v Valuev2) ArrayOK() (*Array, bool) {
	if v.t != bsontype.Array {
		return nil, false
	}
	return v.Array(), true
}

// Binary returns the BSON binary value the Value represents. It panics if the value is a BSON type
// other than binary.
func (v Valuev2) Binary() BinaryPrimitive {
	if v.t != bsontype.Binary {
		panic(ElementTypeError{"bson.Value.Binary", v.t})
	}
	return v.primitive.(BinaryPrimitive)
}

// BinaryOK is the same as Binary, except it returns a boolean instead of
// panicking.
func (v Valuev2) BinaryOK() (BinaryPrimitive, bool) {
	if v.t != bsontype.Binary {
		return BinaryPrimitive{}, false
	}
	return v.Binary(), true
}

// Undefined returns the BSON undefined the Value represents. It panics if the value is a BSON type
// other than binary.
func (v Valuev2) Undefined() UndefinedPrimitive {
	if v.t != bsontype.Undefined {
		panic(ElementTypeError{"bson.Value.Undefined", v.t})
	}
	return UndefinedPrimitive{}
}

// UndefinedOK is the same as Undefined, except it returns a boolean instead of
// panicking.
func (v Valuev2) UndefinedOK() (UndefinedPrimitive, bool) {
	if v.t != bsontype.Undefined {
		return UndefinedPrimitive{}, false
	}
	return v.Undefined(), true
}

// ObjectID returns the BSON ObjectID the Value represents. It panics if the value is a BSON type
// other than ObjectID.
func (v Valuev2) ObjectID() objectid.ObjectID {
	if v.t != bsontype.ObjectID {
		panic(ElementTypeError{"bson.Value.ObjectID", v.t})
	}
	var oid objectid.ObjectID
	copy(oid[:], v.bootstrap[:12])
	return oid
}

// ObjectIDOK is the same as ObjectID, except it returns a boolean instead of
// panicking.
func (v Valuev2) ObjectIDOK() (objectid.ObjectID, bool) {
	if v.t != bsontype.ObjectID {
		return objectid.ObjectID{}, false
	}
	return v.ObjectID(), true
}

// Boolean returns the BSON boolean the Value represents. It panics if the value is a BSON type
// other than boolean.
func (v Valuev2) Boolean() bool {
	if v.t != bsontype.Boolean {
		panic(ElementTypeError{"bson.Value.Boolean", v.t})
	}
	return v.bootstrap[0] == 0x01
}

// BooleanOK is the same as Boolean, except it returns a boolean instead of
// panicking.
func (v Valuev2) BooleanOK() (bool, bool) {
	if v.t != bsontype.Boolean {
		return false, false
	}
	return v.Boolean(), true
}

// DateTime returns the BSON datetime the Value represents. It panics if the value is a BSON type
// other than datetime.
func (v Valuev2) DateTime() DateTimePrimitive {
	if v.t != bsontype.DateTime {
		panic(ElementTypeError{"bson.Value.DateTime", v.t})
	}
	return DateTimePrimitive(v.i64())
}

// DateTimeOK is the same as DateTime, except it returns a boolean instead of
// panicking.
func (v Valuev2) DateTimeOK() (DateTimePrimitive, bool) {
	if v.t != bsontype.DateTime {
		return 0, false
	}
	return v.DateTime(), true
}

// Time returns the BSON datetime the Value represents as time.Time. It panics if the value is a BSON
// type other than datetime.
func (v Valuev2) Time() time.Time {
	if v.t != bsontype.DateTime {
		panic(ElementTypeError{"bson.Value.Time", v.t})
	}
	i := v.i64()
	return time.Unix(int64(i)/1000, int64(i)%1000*1000000)
}

// TimeOK is the same as Time, except it returns a boolean instead of
// panicking.
func (v Valuev2) TimeOK() (time.Time, bool) {
	if v.t != bsontype.DateTime {
		return time.Time{}, false
	}
	return v.Time(), true
}

// Null returns the BSON undefined the Value represents. It panics if the value is a BSON type
// other than binary.
func (v Valuev2) Null() NullPrimitive {
	if v.t != bsontype.Null {
		panic(ElementTypeError{"bson.Value.Null", v.t})
	}
	return NullPrimitive{}
}

// NullOK is the same as Null, except it returns a boolean instead of
// panicking.
func (v Valuev2) NullOK() (NullPrimitive, bool) {
	if v.t != bsontype.Null {
		return NullPrimitive{}, false
	}
	return v.Null(), true
}

// Regex returns the BSON regex the Value represents. It panics if the value is a BSON type
// other than regex.
func (v Valuev2) Regex() RegexPrimitive {
	if v.t != bsontype.Regex {
		panic(ElementTypeError{"bson.Value.Regex", v.t})
	}
	return v.primitive.(RegexPrimitive)
}

// RegexOK is the same as Regex, except that it returns a boolean
// instead of panicking.
func (v Valuev2) RegexOK() (RegexPrimitive, bool) {
	if v.t != bsontype.Regex {
		return RegexPrimitive{}, false
	}
	return v.Regex(), true
}

// DBPointer returns the BSON dbpointer the Value represents. It panics if the value is a BSON type
// other than dbpointer.
func (v Valuev2) DBPointer() DBPointerPrimitive {
	if v.t != bsontype.DBPointer {
		panic(ElementTypeError{"bson.Value.DBPointer", v.t})
	}
	return v.primitive.(DBPointerPrimitive)
}

// DBPointerOK is the same as DBPoitner, except that it returns a boolean
// instead of panicking.
func (v Valuev2) DBPointerOK() (DBPointerPrimitive, bool) {
	if v.t != bsontype.DBPointer {
		return DBPointerPrimitive{}, false
	}
	return v.DBPointer(), true
}

// JavaScript returns the BSON JavaScript the Value represents. It panics if the value is a BSON type
// other than JavaScript.
func (v Valuev2) JavaScript() JavaScriptCodePrimitive {
	if v.t != bsontype.JavaScript {
		panic(ElementTypeError{"bson.Value.JavaScript", v.t})
	}
	return JavaScriptCodePrimitive(v.string())
}

// JavaScriptOK is the same as Javascript, except that it returns a boolean
// instead of panicking.
func (v Valuev2) JavaScriptOK() (JavaScriptCodePrimitive, bool) {
	if v.t != bsontype.JavaScript {
		return "", false
	}
	return v.JavaScript(), true
}

// Symbol returns the BSON symbol the Value represents. It panics if the value is a BSON type
// other than symbol.
func (v Valuev2) Symbol() SymbolPrimitive {
	if v.t != bsontype.Symbol {
		panic(ElementTypeError{"bson.Value.Symbol", v.t})
	}
	return SymbolPrimitive(v.string())
}

// SymbolOK is the same as Javascript, except that it returns a boolean
// instead of panicking.
func (v Valuev2) SymbolOK() (SymbolPrimitive, bool) {
	if v.t != bsontype.Symbol {
		return "", false
	}
	return v.Symbol(), true
}

// CodeWithScope returns the BSON code with scope value the Value represents. It panics if the
// value is a BSON type other than code with scope.
func (v Valuev2) CodeWithScope() CodeWithScopePrimitive {
	if v.t != bsontype.CodeWithScope {
		panic(ElementTypeError{"bson.Value.CodeWithScope", v.t})
	}
	return v.primitive.(CodeWithScopePrimitive)
}

// CodeWithScopeOK is the same as JavascriptWithScope,
// except that it returns a boolean instead of panicking.
func (v Valuev2) CodeWithScopeOK() (CodeWithScopePrimitive, bool) {
	if v.t != bsontype.CodeWithScope {
		return CodeWithScopePrimitive{}, false
	}
	return v.CodeWithScope(), true
}

// Int32 returns the BSON int32 the Value represents. It panics if the value is a BSON type
// other than int32.
func (v Valuev2) Int32() int32 {
	if v.t != bsontype.Int32 {
		panic(ElementTypeError{"bson.Value.Int32", v.t})
	}
	return int32(v.bootstrap[0]) | int32(v.bootstrap[1])<<8 |
		int32(v.bootstrap[2])<<16 | int32(v.bootstrap[3])<<24
}

// Int32OK is the same as Int32, except that it returns a boolean instead of
// panicking.
func (v Valuev2) Int32OK() (int32, bool) {
	if v.t != bsontype.Int32 {
		return 0, false
	}
	return v.Int32(), true
}

// Timestamp returns the BSON timestamp the Value represents. It panics if the value is a
// BSON type other than timestamp.
func (v Valuev2) Timestamp() TimestampPrimitive {
	if v.t != bsontype.Timestamp {
		panic(ElementTypeError{"bson.Value.Timestamp", v.t})
	}
	return TimestampPrimitive{
		I: uint32(v.bootstrap[0]) | uint32(v.bootstrap[1])<<8 |
			uint32(v.bootstrap[2])<<16 | uint32(v.bootstrap[3])<<24,
		T: uint32(v.bootstrap[4]) | uint32(v.bootstrap[5])<<8 |
			uint32(v.bootstrap[6])<<16 | uint32(v.bootstrap[7])<<24,
	}
}

// TimestampOK is the same as Timestamp, except that it returns a boolean
// instead of panicking.
func (v Valuev2) TimestampOK() (TimestampPrimitive, bool) {
	if v.t != bsontype.Timestamp {
		return TimestampPrimitive{}, false
	}
	return v.Timestamp(), true
}

// Int64 returns the BSON int64 the Value represents. It panics if the value is a BSON type
// other than int64.
func (v Valuev2) Int64() int64 {
	if v.t != bsontype.Int64 {
		panic(ElementTypeError{"bson.Value.Int64", v.t})
	}
	return v.i64()
}

// Int64OK is the same as Int64, except that it returns a boolean instead of
// panicking.
func (v Valuev2) Int64OK() (int64, bool) {
	if v.t != bsontype.Int64 {
		return 0, false
	}
	return v.Int64(), true
}

// Decimal128 returns the BSON decimal128 value the Value represents. It panics if the value is a
// BSON type other than decimal128.
func (v Valuev2) Decimal128() decimal.Decimal128 {
	if v.t != bsontype.Decimal128 {
		panic(ElementTypeError{"bson.Value.Decimal128", v.t})
	}
	return v.primitive.(decimal.Decimal128)
}

// Decimal128OK is the same as Decimal128, except that it returns a boolean
// instead of panicking.
func (v Valuev2) Decimal128OK() (decimal.Decimal128, bool) {
	if v.t != bsontype.Decimal128 {
		return decimal.Decimal128{}, false
	}
	return v.Decimal128(), true
}

// MinKey returns the BSON minkey the Value represents. It panics if the value is a BSON type
// other than binary.
func (v Valuev2) MinKey() MinKeyPrimitive {
	if v.t != bsontype.MinKey {
		panic(ElementTypeError{"bson.Value.MinKey", v.t})
	}
	return MinKeyPrimitive{}
}

// MinKeyOK is the same as MinKey, except it returns a boolean instead of
// panicking.
func (v Valuev2) MinKeyOK() (MinKeyPrimitive, bool) {
	if v.t != bsontype.MinKey {
		return MinKeyPrimitive{}, false
	}
	return v.MinKey(), true
}

// MaxKey returns the BSON maxkey the Value represents. It panics if the value is a BSON type
// other than binary.
func (v Valuev2) MaxKey() MaxKeyPrimitive {
	if v.t != bsontype.MaxKey {
		panic(ElementTypeError{"bson.Value.MaxKey", v.t})
	}
	return MaxKeyPrimitive{}
}

// MaxKeyOK is the same as MaxKey, except it returns a boolean instead of
// panicking.
func (v Valuev2) MaxKeyOK() (MaxKeyPrimitive, bool) {
	if v.t != bsontype.MaxKey {
		return MaxKeyPrimitive{}, false
	}
	return v.MaxKey(), true
}

// Equal compares v to v2 and returns true if they are equal. Unknown BSON types are
// never equal. Two empty values are equal.
func (v Valuev2) Equal(v2 Valuev2) bool {
	if v.t != v2.t {
		return false
	}
	if v.IsZero() && v2.IsZero() {
		return true
	}

	switch v.Type() {
	case TypeDouble, TypeDateTime, TypeTimestamp, TypeInt64:
		return bytes.Equal(v.bootstrap[0:8], v2.bootstrap[0:8])
	case TypeString:
		return v.string() == v2.string()
	case TypeEmbeddedDocument:
		return v.Document().Equal(v2.Document())
	case TypeArray:
		return v.Array().Equal(v2.Array())
	case TypeBinary:
		return v.Binary().Equal(v2.Binary())
	case TypeUndefined:
		return true
	case TypeObjectID:
		return bytes.Equal(v.bootstrap[0:12], v2.bootstrap[0:12])
	case TypeBoolean:
		return v.bootstrap[0] == v2.bootstrap[0]
	case TypeNull:
		return true
	case TypeRegex:
		return v.Regex().Equal(v2.Regex())
	case TypeDBPointer:
		return v.DBPointer().Equal(v2.DBPointer())
	case TypeJavaScript:
		return v.JavaScript() == v2.JavaScript()
	case TypeSymbol:
		return v.Symbol() == v2.Symbol()
	case TypeCodeWithScope:
		return v.CodeWithScope().Equal(v2.CodeWithScope())
	case TypeInt32:
		return v.Int32() == v2.Int32()
	case TypeDecimal128:
		h, l := v.Decimal128().GetBytes()
		h2, l2 := v2.Decimal128().GetBytes()
		return h == h2 && l == l2
	case TypeMinKey:
		return true
	case TypeMaxKey:
		return true
	default:
		return false
	}
}
