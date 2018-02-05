package bson

import (
	"github.com/10gen/mongo-go-driver/bson/decimal"
	"github.com/10gen/mongo-go-driver/bson/elements"
	"github.com/10gen/mongo-go-driver/bson/objectid"
)

// C is a convenience variable provided for access to the Constructor methods.
var C Constructor

// AC is a convenience variable provided for access to the ArrayConstructor methods.
var AC ArrayConstructor

// Constructor is used as a namespace for document element constructor functions.
type Constructor struct{}

// ArrayConstructor is used as a namespace for document element constructor functions.
type ArrayConstructor struct{}

// Double creates a double element with the given key and value.
func (Constructor) Double(key string, f float64) *Element {
	b := make([]byte, 1+len(key)+1+8)
	elem := newElement(0, 1+uint32(len(key))+1)
	_, err := elements.Double.Element(0, b, key, f)
	if err != nil {
		panic(err)
	}
	elem.value.data = b
	return elem
}

// String creates a string element with the given key and value.
func (Constructor) String(key string, val string) *Element {
	size := uint32(1 + len(key) + 1 + 4 + len(val) + 1)
	b := make([]byte, size)
	elem := newElement(0, 1+uint32(len(key))+1)
	_, err := elements.String.Element(0, b, key, val)
	if err != nil {
		panic(err)
	}
	elem.value.data = b
	return elem
}

// SubDocument creates a subdocument element with the given key and value.
func (Constructor) SubDocument(key string, d *Document) *Element {
	size := uint32(1 + len(key) + 1)
	b := make([]byte, size)
	elem := newElement(0, size)
	_, err := elements.Byte.Encode(0, b, '\x03')
	if err != nil {
		panic(err)
	}
	_, err = elements.CString.Encode(1, b, key)
	if err != nil {
		panic(err)
	}
	elem.value.data = b
	elem.value.d = d
	return elem
}

// SubDocumentFromReader creates a subdocument element with the given key and value.
func (Constructor) SubDocumentFromReader(key string, r Reader) *Element {
	size := uint32(1 + len(key) + 1 + len(r))
	b := make([]byte, size)
	elem := newElement(0, uint32(1+len(key)+1))
	_, err := elements.Byte.Encode(0, b, '\x03')
	if err != nil {
		panic(err)
	}
	_, err = elements.CString.Encode(1, b, key)
	if err != nil {
		panic(err)
	}
	// NOTE: We don't validate the Reader here since we don't validate the
	// Document when provided to SubDocument.
	copy(b[1+len(key)+1:], r)
	elem.value.data = b
	return elem
}

// SubDocumentFromElements creates a subdocument element with the given key. The elements passed as
// arguments will be used to create a new document as the value.
func (c Constructor) SubDocumentFromElements(key string, elems ...*Element) *Element {
	return c.SubDocument(key, NewDocument(elems...))
}

// Array creates an array element with the given key and value.
func (Constructor) Array(key string, a *Array) *Element {
	size := uint32(1 + len(key) + 1)
	b := make([]byte, size)
	elem := newElement(0, size)
	_, err := elements.Byte.Encode(0, b, '\x04')
	if err != nil {
		panic(err)
	}
	_, err = elements.CString.Encode(1, b, key)
	if err != nil {
		panic(err)
	}
	elem.value.data = b
	elem.value.d = a.doc
	return elem
}

// ArrayFromElements creates an element with the given key. The elements passed as
// arguments will be used to create a new array as the value.
func (c Constructor) ArrayFromElements(key string, values ...*Value) *Element {
	return c.Array(key, NewArray(values...))
}

// Binary creates a binary element with the given key and value.
func (c Constructor) Binary(key string, b []byte) *Element {
	return c.BinaryWithSubtype(key, b, 0)
}

// BinaryWithSubtype creates a binary element with the given key. It will create a new BSON binary value
// with the given data and subtype.
func (Constructor) BinaryWithSubtype(key string, b []byte, btype byte) *Element {
	size := uint32(1 + len(key) + 1 + 4 + 1 + len(b))
	if btype == 2 {
		size += 4
	}

	buf := make([]byte, size)
	elem := newElement(0, 1+uint32(len(key))+1)
	_, err := elements.Binary.Element(0, buf, key, b, btype)
	if err != nil {
		panic(err)
	}

	elem.value.data = buf
	return elem
}

// Undefined creates a undefined element with the given key.
func (Constructor) Undefined(key string) *Element {
	size := 1 + uint32(len(key)) + 1
	b := make([]byte, size)
	elem := newElement(0, size)
	_, err := elements.Byte.Encode(0, b, '\x06')
	if err != nil {
		panic(err)
	}
	_, err = elements.CString.Encode(1, b, key)
	if err != nil {
		panic(err)
	}
	elem.value.data = b
	return elem
}

// ObjectID creates a objectid element with the given key and value.
func (Constructor) ObjectID(key string, oid objectid.ObjectID) *Element {
	size := uint32(1 + len(key) + 1 + 12)
	elem := newElement(0, 1+uint32(len(key))+1)
	elem.value.data = make([]byte, size)

	_, err := elements.ObjectID.Element(0, elem.value.data, key, oid)
	if err != nil {
		panic(err)
	}

	return elem
}

// Boolean creates a boolean element with the given key and value.
func (Constructor) Boolean(key string, b bool) *Element {
	size := uint32(1 + len(key) + 1 + 1)
	elem := newElement(0, 1+uint32(len(key))+1)
	elem.value.data = make([]byte, size)

	_, err := elements.Boolean.Element(0, elem.value.data, key, b)
	if err != nil {
		panic(err)
	}

	return elem
}

// DateTime creates a datetime element with the given key and value.
func (Constructor) DateTime(key string, dt int64) *Element {
	size := uint32(1 + len(key) + 1 + 8)
	elem := newElement(0, 1+uint32(len(key))+1)
	elem.value.data = make([]byte, size)

	_, err := elements.DateTime.Element(0, elem.value.data, key, dt)
	if err != nil {
		panic(err)
	}

	return elem
}

// Null creates a null element with the given key.
func (Constructor) Null(key string) *Element {
	size := uint32(1 + len(key) + 1)
	b := make([]byte, size)
	elem := newElement(0, uint32(1+len(key)+1))
	_, err := elements.Byte.Encode(0, b, '\x0A')
	if err != nil {
		panic(err)
	}
	_, err = elements.CString.Encode(1, b, key)
	if err != nil {
		panic(err)
	}
	elem.value.data = b
	return elem
}

// Regex creates a regex element with the given key and value.
func (Constructor) Regex(key string, pattern, options string) *Element {
	size := uint32(1 + len(key) + 1 + len(pattern) + 1 + len(options) + 1)
	elem := newElement(0, uint32(1+len(key)+1))
	elem.value.data = make([]byte, size)

	_, err := elements.Regex.Element(0, elem.value.data, key, pattern, options)
	if err != nil {
		panic(err)
	}

	return elem
}

// DBPointer creates a dbpointer element with the given key and value.
func (Constructor) DBPointer(key string, ns string, oid objectid.ObjectID) *Element {
	size := uint32(1 + len(key) + 1 + 4 + len(ns) + 1 + 12)
	elem := newElement(0, uint32(1+len(key)+1))
	elem.value.data = make([]byte, size)

	_, err := elements.DBPointer.Element(0, elem.value.data, key, ns, oid)
	if err != nil {
		panic(err)
	}

	return elem
}

// JavaScript creates a JavaScript code element with the given key and value.
func (Constructor) JavaScript(key string, code string) *Element {
	size := uint32(1 + len(key) + 1 + 4 + len(code) + 1)
	elem := newElement(0, uint32(1+len(key)+1))
	elem.value.data = make([]byte, size)

	_, err := elements.JavaScript.Element(0, elem.value.data, key, code)
	if err != nil {
		panic(err)
	}

	return elem
}

// Symbol creates a symbol element with the given key and value.
func (Constructor) Symbol(key string, symbol string) *Element {
	size := uint32(1 + len(key) + 1 + 4 + len(symbol) + 1)
	elem := newElement(0, uint32(1+len(key)+1))
	elem.value.data = make([]byte, size)

	_, err := elements.Symbol.Element(0, elem.value.data, key, symbol)
	if err != nil {
		panic(err)
	}

	return elem
}

// CodeWithScope creates a JavaScript code with scope element with the given key and value.
func (Constructor) CodeWithScope(key string, code string, scope *Document) *Element {
	size := uint32(1 + len(key) + 1 + 4 + 4 + len(code) + 1)
	elem := newElement(0, uint32(1+len(key)+1))
	elem.value.data = make([]byte, size)
	elem.value.d = scope

	_, err := elements.Byte.Encode(0, elem.value.data, '\x0F')
	if err != nil {
		panic(err)
	}

	_, err = elements.CString.Encode(1, elem.value.data, key)
	if err != nil {
		panic(err)
	}

	_, err = elements.Int32.Encode(1+uint(len(key))+1, elem.value.data, int32(size))
	if err != nil {
		panic(err)
	}

	_, err = elements.String.Encode(1+uint(len(key))+1+4, elem.value.data, code)
	if err != nil {
		panic(err)
	}

	return elem
}

// Int32 creates a int32 element with the given key and value.
func (Constructor) Int32(key string, i int32) *Element {
	size := uint32(1 + len(key) + 1 + 4)
	elem := newElement(0, uint32(1+len(key)+1))
	elem.value.data = make([]byte, size)

	_, err := elements.Int32.Element(0, elem.value.data, key, i)
	if err != nil {
		panic(err)
	}

	return elem
}

// Timestamp creates a timestamp element with the given key and value.
func (Constructor) Timestamp(key string, t uint32, i uint32) *Element {
	size := uint32(1 + len(key) + 1 + 8)
	elem := newElement(0, uint32(1+len(key)+1))
	elem.value.data = make([]byte, size)

	_, err := elements.Timestamp.Element(0, elem.value.data, key, t, i)
	if err != nil {
		panic(err)
	}

	return elem
}

// Int64 creates a int64 element with the given key and value.
func (Constructor) Int64(key string, i int64) *Element {
	size := uint32(1 + len(key) + 1 + 8)
	elem := newElement(0, 1+uint32(len(key))+1)
	elem.value.data = make([]byte, size)

	_, err := elements.Int64.Element(0, elem.value.data, key, i)
	if err != nil {
		panic(err)
	}

	return elem
}

// Decimal128 creates a decimal element with the given key and value.
func (Constructor) Decimal128(key string, d decimal.Decimal128) *Element {
	size := uint32(1 + len(key) + 1 + 16)
	elem := newElement(0, uint32(1+len(key)+1))
	elem.value.data = make([]byte, size)

	_, err := elements.Decimal128.Element(0, elem.value.data, key, d)
	if err != nil {
		panic(err)
	}

	return elem
}

// MinKey creates a minkey element with the given key and value.
func (Constructor) MinKey(key string) *Element {
	size := uint32(1 + len(key) + 1)
	elem := newElement(0, uint32(1+len(key)+1))
	elem.value.data = make([]byte, size)

	_, err := elements.Byte.Encode(0, elem.value.data, '\xFF')
	if err != nil {
		panic(err)
	}

	_, err = elements.CString.Encode(1, elem.value.data, key)
	if err != nil {
		panic(err)
	}

	return elem
}

// MaxKey creates a maxkey element with the given key and value.
func (Constructor) MaxKey(key string) *Element {
	size := uint32(1 + len(key) + 1)
	elem := newElement(0, uint32(1+len(key)+1))
	elem.value.data = make([]byte, size)

	_, err := elements.Byte.Encode(0, elem.value.data, '\x7F')
	if err != nil {
		panic(err)
	}

	_, err = elements.CString.Encode(1, elem.value.data, key)
	if err != nil {
		panic(err)
	}

	return elem
}

// Double creates a double element with the given value.
func (ArrayConstructor) Double(f float64) *Value {
	return C.Double("", f).value
}

// String creates a string element with the given value.
func (ArrayConstructor) String(val string) *Value {
	return C.String("", val).value
}

// Document creates a subdocument value from the argument.
func (ArrayConstructor) Document(d *Document) *Value {
	return C.SubDocument("", d).value
}

// DocumentFromReader creates a subdocument element from the given value.
func (ArrayConstructor) DocumentFromReader(r Reader) *Value {
	return C.SubDocumentFromReader("", r).value
}

// DocumentFromElements creates a subdocument element from the given elements.
func (ArrayConstructor) DocumentFromElements(elems ...*Element) *Value {
	return C.SubDocumentFromElements("", elems...).value
}

// Array creates an array value from the argument.
func (ArrayConstructor) Array(a *Array) *Value {
	return C.Array("", a).value
}

// ArrayFromValues creates an array element from the given the elements.
func (ArrayConstructor) ArrayFromValues(values ...*Value) *Value {
	return C.ArrayFromElements("", values...).value
}

// Binary creates a binary value from the argument.
func (ac ArrayConstructor) Binary(b []byte) *Value {
	return ac.BinaryWithSubtype(b, 0)
}

// BinaryWithSubtype creates a new binary element with the given data and subtype.
func (ArrayConstructor) BinaryWithSubtype(b []byte, btype byte) *Value {
	return C.BinaryWithSubtype("", b, btype).value
}

// Undefined creates a undefined element.
func (ArrayConstructor) Undefined() *Value {
	return C.Undefined("").value
}

// ObjectID creates a objectid value from the argument.
func (ArrayConstructor) ObjectID(oid objectid.ObjectID) *Value {
	return C.ObjectID("", oid).value
}

// Boolean creates a boolean value from the argument.
func (ArrayConstructor) Boolean(b bool) *Value {
	return C.Boolean("", b).value
}

// DateTime creates a datetime value from the argument.
func (ArrayConstructor) DateTime(dt int64) *Value {
	return C.DateTime("", dt).value
}

// Null creates a null value from the argument.
func (ArrayConstructor) Null() *Value {
	return C.Null("").value
}

// Regex creates a regex value from the arguments.
func (ArrayConstructor) Regex(pattern, options string) *Value {
	return C.Regex("", pattern, options).value
}

// DBPointer creates a dbpointer value from the arguments.
func (ArrayConstructor) DBPointer(ns string, oid objectid.ObjectID) *Value {
	return C.DBPointer("", ns, oid).value
}

// JavaScript creates a JavaScript code value from the argument.
func (ArrayConstructor) JavaScript(code string) *Value {
	return C.JavaScript("", code).value
}

// Symbol creates a symbol value from the argument.
func (ArrayConstructor) Symbol(symbol string) *Value {
	return C.Symbol("", symbol).value
}

// CodeWithScope creates a JavaScript code with scope value from the arguments.
func (ArrayConstructor) CodeWithScope(code string, scope *Document) *Value {
	return C.CodeWithScope("", code, scope).value
}

// Int32 creates a int32 value from the argument.
func (ArrayConstructor) Int32(i int32) *Value {
	return C.Int32("", i).value
}

// Timestamp creates a timestamp value from the arguments.
func (ArrayConstructor) Timestamp(t uint32, i uint32) *Value {
	return C.Timestamp("", t, i).value
}

// Int64 creates a int64 value from the argument.
func (ArrayConstructor) Int64(i int64) *Value {
	return C.Int64("", i).value
}

// Decimal128 creates a decimal value from the argument.
func (ArrayConstructor) Decimal128(d decimal.Decimal128) *Value {
	return C.Decimal128("", d).value
}

// MinKey creates a minkey value from the argument.
func (ArrayConstructor) MinKey() *Value {
	return C.MinKey("").value
}

// MaxKey creates a maxkey value from the argument.
func (ArrayConstructor) MaxKey() *Value {
	return C.MaxKey("").value
}
