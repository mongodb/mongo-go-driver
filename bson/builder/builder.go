// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package builder

import (
	"errors"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/elements"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// ErrTooShort indicates that the slice provided to encode into is not large enough to fit the data.
var ErrTooShort = errors.New("builder: The provided slice's length is too short")

// C is a convenience variable provided for access to the Constructor methods.
var C Constructor

// AC is a convenience variable provided for access to the ArrayConstructor methods.
var AC ArrayConstructor

// Constructor is used as a namespace for document element constructor functions.
type Constructor struct{}

// ArrayConstructor is used as a namespace for array element constructor functions.
type ArrayConstructor struct{}

// Elementer is the interface implemented by types that can serialize
// themselves into a BSON element.
type Elementer interface {
	Element() (ElementSizer, ElementWriter)
}

// ElementFunc is a function type used to insert BSON element values into a document using a
// DocumentBuilder.
type ElementFunc func() (ElementSizer, ElementWriter)

// Element implements the Elementer interface.
func (ef ElementFunc) Element() (ElementSizer, ElementWriter) {
	return ef()
}

// ElementWriter handles writing an element's BSON representation to a writer.
type ElementWriter func(start uint, writer []byte) (n int, err error)

// ElementSizer handles retrieving the size of an element's BSON representation.
type ElementSizer func() (size uint)

// DocumentBuilder allows the creation of a BSON document by appending elements
// and then writing the document. The document can be written multiple times so
// appending then writing and then appending and writing again is a valid usage
// pattern.
type DocumentBuilder struct {
	Key         string
	funcs       []ElementWriter
	sizers      []ElementSizer
	required    uint // number of required bytes. Should start at 4
	initialized bool
}

// NewDocumentBuilder constructs a new DocumentBuilder.
func NewDocumentBuilder() *DocumentBuilder {
	var b DocumentBuilder
	b.init()

	return &b
}

func (db *DocumentBuilder) init() {
	if db.initialized {
		return
	}
	db.funcs = make([]ElementWriter, 0, 5)
	db.sizers = make([]ElementSizer, 0, 5)
	sizer, f := db.documentHeader()
	db.funcs = append(db.funcs, f)
	db.sizers = append(db.sizers, sizer)
	db.initialized = true
}

// Append adds the given elements to the BSON document.
func (db *DocumentBuilder) Append(elems ...Elementer) *DocumentBuilder {
	db.init()
	for _, elem := range elems {
		sizer, f := elem.Element()
		db.funcs = append(db.funcs, f)
		db.sizers = append(db.sizers, sizer)
	}
	return db
}

func (db *DocumentBuilder) documentHeader() (ElementSizer, ElementWriter) {
	return func() uint { return 5 },
		func(start uint, writer []byte) (n int, err error) {
			return elements.Int32.Encode(start, writer, int32(db.RequiredBytes()))
		}
}

// RequiredBytes returns the number of bytes required to write the entire BSON
// document.
func (db *DocumentBuilder) RequiredBytes() uint {
	return db.requiredSize(false)
}

func (db *DocumentBuilder) embeddedSize() uint {
	return db.requiredSize(true)
}

func (db *DocumentBuilder) requiredSize(embedded bool) uint {
	db.required = 0
	for _, sizer := range db.sizers {
		db.required += sizer()
	}
	if db.required < 5 {
		return 5
	}
	if embedded {
		return db.required + 2 + uint(len(db.Key))
	}
	return db.required //+ 1 // We add 1 because we don't include the ending null byte for the document
}

// Element implements the Elementer interface.
func (db *DocumentBuilder) Element() (ElementSizer, ElementWriter) {
	return db.embeddedSize, func(start uint, writer []byte) (n int, err error) {
		return db.writeDocument(start, writer, true)
	}
}

// WriteDocument writes out the document as BSON to the byte slice.
func (db *DocumentBuilder) WriteDocument(writer []byte) (int64, error) {
	n, err := db.writeDocument(0, writer, false)
	return int64(n), err
}

func (db *DocumentBuilder) writeDocument(start uint, writer []byte, embedded bool) (int, error) {
	db.init()
	// This calculates db.required
	db.requiredSize(embedded)

	var total, n int
	var err error

	if uint(len(writer)) < start+db.required {
		return 0, ErrTooShort
	}
	if embedded {
		n, err = elements.Byte.Encode(start, writer, '\x03')
		start += uint(n)
		total += n
		if err != nil {
			return total, err
		}
		n, err = elements.CString.Encode(start, writer, db.Key)
		start += uint(n)
		total += n
		if err != nil {
			return total, err
		}
	}

	n, err = db.writeElements(start, writer)
	start += uint(n)
	total += n
	if err != nil {
		return n, err
	}

	n, err = elements.Byte.Encode(start, writer, '\x00')
	total += n
	return total, err
}

func (db *DocumentBuilder) writeElements(start uint, writer []byte) (total int, err error) {
	for idx := range db.funcs {
		n, err := db.funcs[idx](start, writer)
		total += n
		start += uint(n)
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// SubDocument creates a subdocument element with the given key and value.
func (Constructor) SubDocument(key string, subdoc *DocumentBuilder) Elementer {
	subdoc.Key = key
	return subdoc
}

// SubDocumentWithElements creates a subdocument element with the given key. The elements passed as
// arguments will be used to create a new document as the value.
func (c Constructor) SubDocumentWithElements(key string, elems ...Elementer) Elementer {
	return (&DocumentBuilder{Key: key}).Append(elems...)
}

// Array creates an array element with the given key and value.
func (c Constructor) Array(key string, array *ArrayBuilder) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// A subdocument will always take (1 + key length + 1) + len(subdoc) bytes
		return func() uint {
				return 2 + uint(len(key)) + array.RequiredBytes()
			},
			func(start uint, writer []byte) (int, error) {
				arrayBytes := make([]byte, array.RequiredBytes())
				_, err := array.WriteDocument(arrayBytes)
				if err != nil {
					return 0, err
				}

				return elements.Array.Element(start, writer, key, arrayBytes)
			}
	}
}

// ArrayWithElements creates an element with the given key. The elements passed as
// arguments will be used to create a new array as the value.
func (c Constructor) ArrayWithElements(key string, elems ...ArrayElementer) ElementFunc {
	var b ArrayBuilder
	b.init()
	b.Append(elems...)

	return C.Array(key, &b)
}

// Double creates a double element with the given key and value.
func (Constructor) Double(key string, f float64) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// A double will always take (1 + key length + 1) + 8 bytes
		return func() uint {
				return uint(10 + len(key))
			},
			func(start uint, writer []byte) (int, error) {
				return elements.Double.Element(start, writer, key, f)
			}
	}
}

// String creates a string element with the given key and value.
func (Constructor) String(key string, value string) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// A string's length is (1 + key length + 1) + (4 + value length + 1)
		return func() uint {
				return uint(7 + len(key) + len(value))
			},
			func(start uint, writer []byte) (int, error) {
				return elements.String.Element(start, writer, key, value)
			}
	}
}

// Binary creates a binary element with the given key and value.
func (c Constructor) Binary(key string, b []byte) ElementFunc {
	return c.BinaryWithSubtype(key, b, 0)
}

// BinaryWithSubtype creates a binary element with the given key. It will create a new BSON binary value
// with the given data and subtype.
func (Constructor) BinaryWithSubtype(key string, b []byte, btype byte) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// A binary of subtype 2 has length (1 + key length + 1) + (4 + 1 + 4 + b length)
		// All other binary subtypes have length  (1 + key length + 1) + (4 + 1 + b length)
		return func() uint {
				//
				if btype == 2 {
					return uint(11 + len(key) + len(b))
				}

				return uint(7 + len(key) + len(b))
			},
			func(start uint, writer []byte) (int, error) {
				return elements.Binary.Element(start, writer, key, b, btype)
			}
	}
}

// Undefined creates a undefined element with the given key.
func (Constructor) Undefined(key string) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// Undefined's length is 1 + key length + 1
		return func() uint {
				return uint(2 + len(key))
			},
			func(start uint, writer []byte) (int, error) {
				var total int

				n, err := elements.Byte.Encode(start, writer, '\x06')
				start += uint(n)
				total += n
				if err != nil {
					return total, err
				}

				n, err = elements.CString.Encode(start, writer, key)
				start += uint(n)
				total += n
				if err != nil {
					return total, err
				}

				return total, nil
			}
	}
}

// ObjectID creates a objectid element with the given key and value.
func (Constructor) ObjectID(key string, oid objectid.ObjectID) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// An ObjectID's length is (1 + key length + 1) + 12
		return func() uint {
				return uint(14 + len(key))
			},
			func(start uint, writer []byte) (int, error) {
				return elements.ObjectID.Element(start, writer, key, oid)
			}
	}
}

// Boolean creates a boolean element with the given key and value.
func (Constructor) Boolean(key string, b bool) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// An ObjectID's length is (1 + key length + 1) + 1
		return func() uint {
				return uint(3 + len(key))
			},
			func(start uint, writer []byte) (int, error) {
				return elements.Boolean.Element(start, writer, key, b)
			}
	}
}

// DateTime creates a datetime element with the given key and value.
func (Constructor) DateTime(key string, dt int64) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// Datetime's length is (1 + key length + 1) + 8
		return func() uint {
				return uint(10 + len(key))
			},
			func(start uint, writer []byte) (int, error) {
				return elements.DateTime.Element(start, writer, key, dt)
			}
	}
}

// Null creates a null element with the given key.
func (Constructor) Null(key string) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// Null's length is 1 + key length + 1
		return func() uint {
				return uint(2 + len(key))
			},
			func(start uint, writer []byte) (int, error) {
				var total int

				n, err := elements.Byte.Encode(start, writer, '\x0A')
				start += uint(n)
				total += n
				if err != nil {
					return total, err
				}

				n, err = elements.CString.Encode(start, writer, key)
				start += uint(n)
				total += n
				if err != nil {
					return total, err
				}

				return total, nil
			}
	}
}

// Regex creates a regex element with the given key and value.
func (Constructor) Regex(key string, pattern, options string) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// Null's length is (1 + key length + 1) + (pattern length + 1) + (options length + 1)
		return func() uint {
				return uint(4 + len(key) + len(pattern) + len(options))
			},
			func(start uint, writer []byte) (int, error) {
				return elements.Regex.Element(start, writer, key, pattern, options)
			}
	}
}

// DBPointer creates a dbpointer element with the given key and value.
func (Constructor) DBPointer(key string, ns string, oid objectid.ObjectID) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// An dbpointer's length is (1 + key length + 1) + (4 + ns length + 1) + 12
		return func() uint {
				return uint(19 + len(key) + len(ns))
			},
			func(start uint, writer []byte) (int, error) {
				return elements.DBPointer.Element(start, writer, key, ns, oid)
			}
	}
}

// JavaScriptCode creates a JavaScript code element with the given key and value.
func (Constructor) JavaScriptCode(key string, code string) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// JavaScript code's length is (1 + key length + 1) + (4 + code length + 1)
		return func() uint {
				return uint(7 + len(key) + len(code))
			},
			func(start uint, writer []byte) (int, error) {
				return elements.JavaScript.Element(start, writer, key, code)
			}
	}
}

// Symbol creates a symbol element with the given key and value.
func (Constructor) Symbol(key string, symbol string) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// A symbol's length is (1 + key length + 1) + (4 + symbol length + 1)
		return func() uint {
				return uint(7 + len(key) + len(symbol))
			},
			func(start uint, writer []byte) (int, error) {
				return elements.Symbol.Element(start, writer, key, symbol)
			}
	}
}

// CodeWithScope creates a JavaScript code with scope element with the given key and value.
func (Constructor) CodeWithScope(key string, code string, scope []byte) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// JavaScript code with scope's length is (1 + key length + 1) + 4 +  (4 + len key + 1) + len(scope)
		return func() uint {
				return uint(11 + len(key) + len(code) + len(scope))
			},
			func(start uint, writer []byte) (int, error) {
				return elements.CodeWithScope.Element(start, writer, key, code, scope)
			}
	}
}

// Int32 creates a int32 element with the given key and value.
func (Constructor) Int32(key string, i int32) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// An int32's length is (1 + key length + 1) + 4 bytes
		return func() uint {
				return uint(6 + len(key))
			},
			func(start uint, writer []byte) (int, error) {
				return elements.Int32.Element(start, writer, key, i)
			}
	}
}

// Timestamp creates a timestamp element with the given key and value.
func (Constructor) Timestamp(key string, t uint32, i uint32) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// An decimal's length is (1 + key length + 1) + 8 bytes
		return func() uint {
				return uint(10 + len(key))
			},
			func(start uint, writer []byte) (int, error) {
				return elements.Timestamp.Element(start, writer, key, t, i)
			}
	}
}

// Int64 creates a int64 element with the given key and value.
func (Constructor) Int64(key string, i int64) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// An int64's length is (1 + key length + 1) + 8 bytes
		return func() uint {
				return uint(10 + len(key))
			},
			func(start uint, writer []byte) (int, error) {
				return elements.Int64.Element(start, writer, key, i)
			}
	}
}

// Decimal creates a decimal element with the given key and value.
func (Constructor) Decimal(key string, d decimal.Decimal128) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// An decimal's length is (1 + key length + 1) + 16 bytes
		return func() uint {
				return uint(18 + len(key))
			},
			func(start uint, writer []byte) (int, error) {
				return elements.Decimal128.Element(start, writer, key, d)
			}
	}
}

// MinKey creates a minkey element with the given key and value.
func (Constructor) MinKey(key string) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// An min key's length is (1 + key length + 1)
		return func() uint {
				return uint(2 + len(key))
			},
			func(start uint, writer []byte) (int, error) {
				var total int

				n, err := elements.Byte.Encode(start, writer, '\xFF')
				start += uint(n)
				total += n
				if err != nil {
					return total, err
				}

				n, err = elements.CString.Encode(start, writer, key)
				start += uint(n)
				total += n
				if err != nil {
					return total, err
				}

				return total, nil
			}
	}
}

// MaxKey creates a maxkey element with the given key and value.
func (Constructor) MaxKey(key string) ElementFunc {
	return func() (ElementSizer, ElementWriter) {
		// An max key's length is (1 + key length + 1)
		return func() uint {
				return uint(2 + len(key))
			},
			func(start uint, writer []byte) (int, error) {
				var total int

				n, err := elements.Byte.Encode(start, writer, '\x7F')
				start += uint(n)
				total += n
				if err != nil {
					return total, err
				}

				n, err = elements.CString.Encode(start, writer, key)
				start += uint(n)
				total += n
				if err != nil {
					return total, err
				}

				return total, nil
			}
	}
}
