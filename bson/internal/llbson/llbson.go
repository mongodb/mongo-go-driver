// Package llbson contains functions that can be used to encode and decode BSON
// elements and values to or from a slice of bytes. These functions are aimed at
// allowing low level manipulation of BSON and can be used to build a higher
// level BSON library.
//
// The Read* functions within this package return the values of the element and
// a boolean indicating if the values are valid. A boolean was used instead of
// an error because any error that would be returned would be the same: not
// enough bytes. This library attempts to do no validation, it will only return
// false if there are not enough bytes for an item to be read. For example, the
// ReadDocument function checks the length, if that length is larger than the
// number of bytes availble, it will return false, if there are enough bytes, it
// will return those bytes and true. It is the consumers responsibility to
// validate those bytes.
//
// The Append* functions within this package will append the type value to the
// given dst slice. If the slice has enough capacity, it will not grow the
// slice. The Append*Element functions within this package operate in the same
// way, but additionally append the BSON type and the key before the value.
package llbson

import (
	"bytes"
	"math"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// AppendType will append t to dst and return the extended buffer.
func AppendType(dst []byte, t Type) []byte { return append(dst, byte(t)) }

// AppendKey will append key to dst and return the extended buffer.
func AppendKey(dst []byte, key string) []byte { return append(dst, key+string(0x00)...) }

// AppendHeader will append Type t and key to dst and return the extended
// buffer.
func AppendHeader(dst []byte, t Type, key string) []byte {
	dst = AppendType(dst, t)
	dst = append(dst, key...)
	return append(dst, 0x00)
	// return append(AppendType(dst, t), key+string(0x00)...)
}

// ReadType will return the first byte of the provided []byte as a type. If
// there is no availble byte, false is returned.
func ReadType(src []byte) (Type, bool) {
	if len(src) < 1 {
		return 0, false
	}
	return Type(src[0]), true
}

// ReadKey will return the first key in src. The 0x00 byte will not be present
// in the returned string. If there are not enough bytes available, false is
// returned.
func ReadKey(src []byte) (string, bool) { return readcstring(src) }

// ReadHeader will return the type byte and the key in src. If both of these
// values cannot be read, false is returned.
func ReadHeader(src []byte) (t Type, key string, ok bool) {
	t, ok = ReadType(src)
	if !ok {
		return 0, "", false
	}
	key, ok = ReadKey(src[1:])
	if !ok {
		return 0, "", false
	}

	return t, key, true
}

// AppendDouble will append f to dst and return the extended buffer.
func AppendDouble(dst []byte, f float64) []byte {
	return appendu64(dst, math.Float64bits(f))
}

// AppendDoubleElement will append a BSON double element using key and f to dst
// and return the extended buffer.
func AppendDoubleElement(dst []byte, key string, f float64) []byte {
	return AppendDouble(AppendHeader(dst, TypeDouble, key), f)
}

// ReadDouble will read a float64 from src. If there are not enough bytes it
// will return false.
func ReadDouble(src []byte) (float64, bool) {
	bits, ok := readu64(src)
	if !ok {
		return 0, false
	}
	return math.Float64frombits(bits), true
}

// AppendString will append s to dst and return the extended buffer.
func AppendString(dst []byte, s string) []byte {
	return appendstring(dst, s)
}

// AppendStringElement will append a BSON string element using key and val to dst
// and return the extended buffer.
func AppendStringElement(dst []byte, key, val string) []byte {
	return AppendString(AppendHeader(dst, TypeString, key), val)
}

// ReadString will read a string from src. If there are not enough bytes it
// will return false.
func ReadString(src []byte) (string, bool) {
	return readstring(src)
}

// AppendDocument will append doc to dst and return the extended buffer.
func AppendDocument(dst []byte, doc []byte) []byte { return append(dst, doc...) }

// AppendDocumentElement will append a BSON embeded document element using key
// and doc to dst and return the extended buffer.
func AppendDocumentElement(dst []byte, key string, doc []byte) []byte {
	return AppendDocument(AppendHeader(dst, TypeEmbeddedDocument, key), doc)
}

// ReadDocument will read a document from src. If there are not enough bytes it
// will return false.
func ReadDocument(src []byte) ([]byte, bool) { return readLengthBytes(src) }

// AppendArray will append arr to dst and return the extended buffer.
func AppendArray(dst []byte, arr []byte) []byte { return append(dst, arr...) }

// AppendArrayElement will append a BSON array element using key and arr to dst
// and return the extended buffer.
func AppendArrayElement(dst []byte, key string, arr []byte) []byte {
	return AppendArray(AppendHeader(dst, TypeArray, key), arr)
}

// ReadArray will read an array from src. If there are not enough bytes it
// will return false.
func ReadArray(src []byte) ([]byte, bool) { return readLengthBytes(src) }

// AppendBinary will append subtype and b to dst and return the extended buffer.
func AppendBinary(dst []byte, subtype byte, b []byte) []byte {
	if subtype == 0x02 {
		return appendBinarySubtype2(dst, subtype, b)
	}
	dst = append(appendLength(dst, int32(len(b))), subtype)
	return append(dst, b...)
}

// AppendBinaryElement will append a BSON binary element using key, subtype, and
// b to dst and return the extended buffer.
func AppendBinaryElement(dst []byte, key string, subtype byte, b []byte) []byte {
	return AppendBinary(AppendHeader(dst, TypeBinary, key), subtype, b)
}

// ReadBinary will read a subtype and bin from src. If there are not enough bytes it
// will return false.
func ReadBinary(src []byte) (subtype byte, bin []byte, ok bool) {
	length, ok := readLength(src)
	if !ok {
		return 0x00, nil, false
	}
	if len(src[4:]) < 1 { // subtype
		return 0x00, nil, false
	}
	subtype = src[4]

	if subtype == 0x02 {
		length, ok = readLength(src[5:])
		if !ok || len(src[9:]) < int(length) {
			return 0x00, nil, false
		}
		return subtype, src[9 : length+9], true
	}

	if len(src[5:]) < int(length) {
		return 0x00, nil, false
	}

	return subtype, src[5 : length+5], true
}

// AppendUndefinedElement will append a BSON undefined element using key to dst
// and return the extended buffer.
func AppendUndefinedElement(dst []byte, key string) []byte {
	return AppendHeader(dst, TypeUndefined, key)
}

// AppendObjectID will append oid to dst and return the extended buffer.
func AppendObjectID(dst []byte, oid objectid.ObjectID) []byte { return append(dst, oid[:]...) }

// AppendObjectIDElement will append a BSON ObjectID element using key and oid to dst
// and return the extended buffer.
func AppendObjectIDElement(dst []byte, key string, oid objectid.ObjectID) []byte {
	return AppendObjectID(AppendHeader(dst, TypeObjectID, key), oid)
}

// ReadObjectID will read an ObjectID from src. If there are not enough bytes it
// will return false.
func ReadObjectID(src []byte) (objectid.ObjectID, bool) {
	if len(src) < 12 {
		return objectid.ObjectID{}, false
	}
	var oid objectid.ObjectID
	copy(oid[:], src[0:12])
	return oid, true
}

// AppendBoolean will append b to dst and return the extended buffer.
func AppendBoolean(dst []byte, b bool) []byte {
	if b {
		return append(dst, 0x01)
	}
	return append(dst, 0x00)
}

// AppendBooleanElement will append a BSON boolean element using key and b to dst
// and return the extended buffer.
func AppendBooleanElement(dst []byte, key string, b bool) []byte {
	return AppendBoolean(AppendHeader(dst, TypeBoolean, key), b)
}

// ReadBoolean will read a bool from src. If there are not enough bytes it
// will return false.
func ReadBoolean(src []byte) (bool, bool) {
	if len(src) < 1 {
		return false, false
	}

	return src[0] == 0x01, true
}

// AppendDateTime will append dt to dst and return the extended buffer.
func AppendDateTime(dst []byte, dt int64) []byte { return appendi64(dst, dt) }

// AppendDateTimeElement will append a BSON datetime element using key and dt to dst
// and return the extended buffer.
func AppendDateTimeElement(dst []byte, key string, dt int64) []byte {
	return AppendDateTime(AppendHeader(dst, TypeDateTime, key), dt)
}

// ReadDateTime will read an int64 datetime from src. If there are not enough bytes it
// will return false.
func ReadDateTime(src []byte) (int64, bool) { return readi64(src) }

// AppendNullElement will append a BSON null element using key to dst
// and return the extended buffer.
func AppendNullElement(dst []byte, key string) []byte { return AppendHeader(dst, TypeNull, key) }

// AppendRegex will append pattern and options to dst and return the extended buffer.
func AppendRegex(dst []byte, pattern, options string) []byte {
	return append(dst, pattern+string(0x00)+options+string(0x00)...)
}

// AppendRegexElement will append a BSON regex element using key, pattern, and
// options to dst and return the extended buffer.
func AppendRegexElement(dst []byte, key, pattern, options string) []byte {
	return AppendRegex(AppendHeader(dst, TypeRegex, key), pattern, options)
}

// ReadRegex will read a pattern and options from src. If there are not enough bytes it
// will return false.
func ReadRegex(src []byte) (pattern, options string, ok bool) {
	pattern, ok = readcstring(src)
	if !ok {
		return "", "", false
	}
	options, ok = readcstring(src[len(pattern)+1:])
	if !ok {
		return "", "", false
	}
	return pattern, options, true
}

// AppendDBPointer will append ns and oid to dst and return the extended buffer.
func AppendDBPointer(dst []byte, ns string, oid objectid.ObjectID) []byte {
	return append(appendstring(dst, ns), oid[:]...)
}

// AppendDBPointerElement will append a BSON DBPointer element using key, ns,
// and oid to dst and return the extended buffer.
func AppendDBPointerElement(dst []byte, key, ns string, oid objectid.ObjectID) []byte {
	return AppendDBPointer(AppendHeader(dst, TypeDBPointer, key), ns, oid)
}

// ReadDBPointer will read a ns and oid from src. If there are not enough bytes it
// will return false.
func ReadDBPointer(src []byte) (ns string, oid objectid.ObjectID, ok bool) {
	ns, ok = readstring(src)
	if !ok {
		return "", objectid.ObjectID{}, false
	}
	oid, ok = ReadObjectID(src[4+len(ns)+1:])
	if !ok {
		return "", objectid.ObjectID{}, false
	}
	return ns, oid, true
}

// AppendJavaScript will append js to dst and return the extended buffer.
func AppendJavaScript(dst []byte, js string) []byte { return appendstring(dst, js) }

// AppendJavaScriptElement will append a BSON JavaScript element using key and
// js to dst and return the extended buffer.
func AppendJavaScriptElement(dst []byte, key, js string) []byte {
	return AppendJavaScript(AppendHeader(dst, TypeJavaScript, key), js)
}

// ReadJavaScript will read a js string from src. If there are not enough bytes it
// will return false.
func ReadJavaScript(src []byte) (js string, ok bool) { return readstring(src) }

// AppendSymbol will append symbol to dst and return the extended buffer.
func AppendSymbol(dst []byte, symbol string) []byte { return appendstring(dst, symbol) }

// AppendSymbolElement will append a BSON symbol element using key and symbol to dst
// and return the extended buffer.
func AppendSymbolElement(dst []byte, key, symbol string) []byte {
	return AppendSymbol(AppendHeader(dst, TypeSymbol, key), symbol)
}

// ReadSymbol will read a symbol string from src. If there are not enough bytes it
// will return false.
func ReadSymbol(src []byte) (symbol string, ok bool) { return readstring(src) }

// AppendCodeWithScope will append code and scope to dst and return the extended buffer.
func AppendCodeWithScope(dst []byte, code string, scope []byte) []byte {
	length := int32(4 + 4 + len(code) + 1 + len(scope)) // length of cws, length of code, code, 0x00, scope
	dst = appendLength(dst, length)

	return append(appendstring(dst, code), scope...)
}

// AppendCodeWithScopeElement will append a BSON code with scope element using
// key, code, and scope to dst
// and return the extended buffer.
func AppendCodeWithScopeElement(dst []byte, key, code string, scope []byte) []byte {
	return AppendCodeWithScope(AppendHeader(dst, TypeCodeWithScope, key), code, scope)
}

// ReadCodeWithScope will read code and scope from src. If there are not enough bytes it
// will return false.
func ReadCodeWithScope(src []byte) (code string, scope []byte, ok bool) {
	length, ok := readLength(src)
	if !ok || len(src) < int(length) {
		return "", nil, false
	}

	code, ok = readstring(src[4:length])
	if !ok {
		return "", nil, false
	}

	scope = src[9+len(code):] // 9 = 4 (total len) + 4 (code length) + 1 (0x00)
	return code, scope, true
}

// AppendInt32 will append i32 to dst and return the extended buffer.
func AppendInt32(dst []byte, i32 int32) []byte { return appendi32(dst, i32) }

// AppendInt32Element will append a BSON int32 element using key and i32 to dst
// and return the extended buffer.
func AppendInt32Element(dst []byte, key string, i32 int32) []byte {
	return AppendInt32(AppendHeader(dst, TypeInt32, key), i32)
}

// ReadInt32 will read an int32 from src. If there are not enough bytes it
// will return false.
func ReadInt32(src []byte) (int32, bool) { return readi32(src) }

// AppendTimestamp will append t and i to dst and return the extended buffer.
func AppendTimestamp(dst []byte, t, i uint32) []byte {
	return appendu32(appendu32(dst, i), t) // i is the lower 4 bytes, t is the higher 4 bytes
}

// AppendTimestampElement will append a BSON timestamp element using key, t, and
// i to dst and return the extended buffer.
func AppendTimestampElement(dst []byte, key string, t, i uint32) []byte {
	return AppendTimestamp(AppendHeader(dst, TypeTimestamp, key), t, i)
}

// ReadTimestamp will read t and i from src. If there are not enough bytes it
// will return false.
func ReadTimestamp(src []byte) (t, i uint32, ok bool) {
	i, ok = readu32(src)
	if !ok {
		return 0, 0, false
	}
	t, ok = readu32(src[4:])
	if !ok {
		return 0, 0, false
	}
	return t, i, true
}

// AppendInt64 will append i64 to dst and return the extended buffer.
func AppendInt64(dst []byte, i64 int64) []byte { return appendi64(dst, i64) }

// AppendInt64Element will append a BSON int64 element using key and i64 to dst
// and return the extended buffer.
func AppendInt64Element(dst []byte, key string, i64 int64) []byte {
	return AppendInt64(AppendHeader(dst, TypeInt64, key), i64)
}

// ReadInt64 will read an int64 from src. If there are not enough bytes it
// will return false.
func ReadInt64(src []byte) (int64, bool) { return readi64(src) }

// AppendDecimal128 will append d128 to dst and return the extended buffer.
func AppendDecimal128(dst []byte, d128 decimal.Decimal128) []byte {
	high, low := d128.GetBytes()
	return appendu64(appendu64(dst, low), high)
}

// AppendDecimal128Element will append a BSON decimal128 element using key and
// d128 to dst and return the extended buffer.
func AppendDecimal128Element(dst []byte, key string, d128 decimal.Decimal128) []byte {
	return AppendDecimal128(AppendHeader(dst, TypeDecimal128, key), d128)
}

// ReadDecimal128 will read a decimal.Decimal128 from src. If there are not enough bytes it
// will return false.
func ReadDecimal128(src []byte) (decimal.Decimal128, bool) {
	l, ok := readu64(src)
	if !ok {
		return decimal.Decimal128{}, false
	}

	h, ok := readu64(src[8:])
	if !ok {
		return decimal.Decimal128{}, false
	}

	return decimal.NewDecimal128(h, l), true
}

// AppendMaxKeyElement will append a BSON max key element using key to dst
// and return the extended buffer.
func AppendMaxKeyElement(dst []byte, key string) []byte { return AppendHeader(dst, TypeMaxKey, key) }

// AppendMinKeyElement will append a BSON min key element using key to dst
// and return the extended buffer.
func AppendMinKeyElement(dst []byte, key string) []byte { return AppendHeader(dst, TypeMinKey, key) }

// EqualValue will return true if the two values are equal.
func EqualValue(t1, t2 Type, v1, v2 []byte) bool {
	if t1 != t2 {
		return false
	}
	length1, ok := valueLength(t1, v1)
	if !ok {
		return false
	}
	length2, ok := valueLength(t2, v2)
	if !ok {
		return false
	}
	return bytes.Equal(v1[:length1], v2[:length2])
}

func valueLength(t Type, val []byte) (int32, bool) {
	var length int32
	ok := true
	switch t {
	case TypeArray, TypeEmbeddedDocument, TypeCodeWithScope:
		length, ok = readLength(val)
	case TypeBinary:
		length, ok = readLength(val)
		length += 4 + 1 // binary length + subtype byte
	case TypeBoolean:
		length = 1
	case TypeDBPointer:
		length, ok = readLength(val)
		length += 4 + 12 // string length + ObjectID length
	case TypeDateTime, TypeDouble, TypeInt64, TypeTimestamp:
		length = 8
	case TypeDecimal128:
		length = 16
	case TypeInt32:
		length = 4
	case TypeJavaScript, TypeString, TypeSymbol:
		length, ok = readLength(val)
		length += 4
	case TypeMaxKey, TypeMinKey, TypeNull, TypeUndefined:
		length = 0
	case TypeObjectID:
		length = 12
	case TypeRegex:
		regex := bytes.IndexByte(val, 0x00)
		if regex < 0 {
			ok = false
			break
		}
		pattern := bytes.IndexByte(val, 0x00)
		if pattern < 0 {
			ok = false
			break
		}
		length = int32(int64(regex) + 1 + int64(pattern) + 1)
	default:
		ok = false
	}

	return length, ok
}

func appendLength(dst []byte, l int32) []byte { return appendi32(dst, l) }

func appendi32(dst []byte, i32 int32) []byte {
	return append(dst, byte(i32), byte(i32>>8), byte(i32>>16), byte(i32>>24))
}

func readLength(src []byte) (int32, bool) { return readi32(src) }

func readi32(src []byte) (int32, bool) {
	if len(src) < 4 {
		return 0, false
	}

	return (int32(src[0]) | int32(src[1])<<8 | int32(src[2])<<16 | int32(src[3])<<24), true
}

func appendi64(dst []byte, i64 int64) []byte {
	return append(dst,
		byte(i64), byte(i64>>8), byte(i64>>16), byte(i64>>24),
		byte(i64>>32), byte(i64>>40), byte(i64>>48), byte(i64>>56),
	)
}

func readi64(src []byte) (int64, bool) {
	if len(src) < 8 {
		return 0, false
	}
	i64 := (int64(src[0]) | int64(src[1])<<8 | int64(src[2])<<16 | int64(src[3])<<24 |
		int64(src[4])<<32 | int64(src[5])<<40 | int64(src[6])<<48 | int64(src[7])<<56)
	return i64, true
}

func appendu32(dst []byte, u32 uint32) []byte {
	return append(dst, byte(u32), byte(u32>>8), byte(u32>>16), byte(u32>>24))
}

func readu32(src []byte) (uint32, bool) {
	if len(src) < 4 {
		return 0, false
	}

	return (uint32(src[0]) | uint32(src[1])<<8 | uint32(src[2])<<16 | uint32(src[3])<<24), true
}

func appendu64(dst []byte, u64 uint64) []byte {
	return append(dst,
		byte(u64), byte(u64>>8), byte(u64>>16), byte(u64>>24),
		byte(u64>>32), byte(u64>>40), byte(u64>>48), byte(u64>>56),
	)
}

func readu64(src []byte) (uint64, bool) {
	if len(src) < 8 {
		return 0, false
	}
	u64 := (uint64(src[0]) | uint64(src[1])<<8 | uint64(src[2])<<16 | uint64(src[3])<<24 |
		uint64(src[4])<<32 | uint64(src[5])<<40 | uint64(src[6])<<48 | uint64(src[7])<<56)
	return u64, true
}

func readcstring(src []byte) (string, bool) {
	idx := bytes.IndexByte(src, 0x00)
	if idx < 0 {
		return "", false
	}
	return string(src[:idx]), true
}

func appendstring(dst []byte, s string) []byte {
	l := int32(len(s) + 1)
	dst = appendLength(dst, l)
	dst = append(dst, s...)
	return append(dst, 0x00)
}

func readstring(src []byte) (string, bool) {
	l, ok := readLength(src)
	if !ok {
		return "", false
	}
	if len(src[4:]) < int(l) {
		return "", false
	}

	return string(src[4 : l+4-1]), true
}

// readLengthBytes attempts to read a length and that number of bytes. This
// function requires that the length include the four bytes for itself.
func readLengthBytes(src []byte) ([]byte, bool) {
	l, ok := readLength(src)
	if !ok {
		return nil, false
	}
	if len(src) < int(l) {
		return nil, false
	}
	return src[:l], true
}

func appendBinarySubtype2(dst []byte, subtype byte, b []byte) []byte {
	dst = appendLength(dst, int32(len(b)+4)) // The bytes we'll encode need to be 4 larger for the length bytes
	dst = append(dst, subtype)
	dst = appendLength(dst, int32(len(b)))
	return append(dst, b...)
}
