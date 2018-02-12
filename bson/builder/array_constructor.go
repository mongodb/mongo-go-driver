// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package builder

import (
	"strconv"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// ArrayElementer is the interface implemented by types that can serialize
// themselves into a BSON array element.
type ArrayElementer interface {
	ArrayElement(pos uint) Elementer
}

// ArrayElementFunc is a function type used to insert BSON element values into an array using an
// ArrayBuilder.
type ArrayElementFunc func(pos uint) Elementer

// ArrayElement implements the ArrayElementer interface.
func (aef ArrayElementFunc) ArrayElement(pos uint) Elementer {
	return aef(pos)
}

// ArrayBuilder allows the creation of a BSON document by appending elements
// and then writing the document. The array can be written multiple times so
// appending then writing and then appending and writing again is a valid usage
// pattern.
type ArrayBuilder struct {
	DocumentBuilder
	current uint
}

// Append adds the given elements to the BSON array.
func (ab *ArrayBuilder) Append(elems ...ArrayElementer) *ArrayBuilder {
	ab.init()
	for _, arrelem := range elems {
		sizer, f := arrelem.ArrayElement(ab.current).Element()
		ab.current++
		ab.funcs = append(ab.funcs, f)
		ab.sizers = append(ab.sizers, sizer)
	}
	return ab
}

// SubDocument creates a subdocument element with the given value.
func (ArrayConstructor) SubDocument(db *DocumentBuilder) ArrayElementFunc {
	return func(pos uint) Elementer {
		key := strconv.FormatUint(uint64(pos), 10)
		return C.SubDocument(key, db)
	}
}

// SubDocumentWithElements creates a subdocument element from the given elements.
func (ArrayConstructor) SubDocumentWithElements(elems ...Elementer) ArrayElementFunc {
	return func(pos uint) Elementer {
		key := strconv.FormatUint(uint64(pos), 10)
		return C.SubDocumentWithElements(key, elems...)
	}
}

// Array creates an array element with the given value.
func (ArrayConstructor) Array(arr *ArrayBuilder) ArrayElementFunc {
	return func(pos uint) Elementer {
		key := strconv.FormatUint(uint64(pos), 10)
		return C.Array(key, arr)
	}
}

// ArrayWithElements creates an array element from the given the elements.
func (ArrayConstructor) ArrayWithElements(elems ...ArrayElementer) ArrayElementFunc {
	return func(pos uint) Elementer {
		key := strconv.FormatUint(uint64(pos), 10)
		return C.ArrayWithElements(key, elems...)
	}
}

// Double creates a double element with the given value.
func (ArrayConstructor) Double(f float64) ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.Double(strconv.FormatUint(uint64(pos), 10), f)
	}
}

// String creates a string element with the given value.
func (ArrayConstructor) String(s string) ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.String(strconv.FormatUint(uint64(pos), 10), s)
	}
}

// Binary creates a binary element with the given value.
func (ArrayConstructor) Binary(b []byte) ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.Binary(strconv.FormatUint(uint64(pos), 10), b)
	}
}

// BinaryWithSubtype creates a new binary element with the given data and subtype.
func (ArrayConstructor) BinaryWithSubtype(b []byte, btype byte) ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.BinaryWithSubtype(strconv.FormatUint(uint64(pos), 10), b, btype)
	}
}

// Undefined creates a undefined element.
func (ArrayConstructor) Undefined() ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.Undefined(strconv.FormatUint(uint64(pos), 10))
	}
}

// ObjectID creates a objectid element with the given value.
func (ArrayConstructor) ObjectID(oid objectid.ObjectID) ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.ObjectID(strconv.FormatUint(uint64(pos), 10), oid)
	}
}

// Boolean creates a boolean element with the given value.
func (ArrayConstructor) Boolean(b bool) ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.Boolean(strconv.FormatUint(uint64(pos), 10), b)
	}
}

// DateTime creates a datetime element with the given value.
func (ArrayConstructor) DateTime(dt int64) ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.DateTime(strconv.FormatUint(uint64(pos), 10), dt)
	}
}

// Null creates a null element with the given value.
func (ArrayConstructor) Null() ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.Null(strconv.FormatUint(uint64(pos), 10))
	}
}

// Regex creates a regex element with the given value.
func (ArrayConstructor) Regex(pattern string, options string) ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.Regex(strconv.FormatUint(uint64(pos), 10), pattern, options)
	}
}

// DBPointer creates a dbpointer element with the given value.
func (ArrayConstructor) DBPointer(ns string, oid objectid.ObjectID) ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.DBPointer(strconv.FormatUint(uint64(pos), 10), ns, oid)
	}
}

// JavaScriptCode creates a JavaScript code element with the given value.
func (ArrayConstructor) JavaScriptCode(code string) ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.JavaScriptCode(strconv.FormatUint(uint64(pos), 10), code)
	}
}

// Symbol creates a symbol element with the given value.
func (ArrayConstructor) Symbol(symbol string) ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.Symbol(strconv.FormatUint(uint64(pos), 10), symbol)
	}
}

// CodeWithScope creates a JavaScript code with scope element with the given value.
func (ArrayConstructor) CodeWithScope(code string, scope []byte) ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.CodeWithScope(strconv.FormatUint(uint64(pos), 10), code, scope)
	}
}

// Int32 creates a int32 element with the given value.
func (ArrayConstructor) Int32(i int32) ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.Int32(strconv.FormatUint(uint64(pos), 10), i)
	}
}

// Timestamp creates a timestamp element with the given value.
func (ArrayConstructor) Timestamp(t uint32, i uint32) ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.Timestamp(strconv.FormatUint(uint64(pos), 10), t, i)
	}
}

// Int64 creates a int64 element with the given value.
func (ArrayConstructor) Int64(i int64) ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.Int64(strconv.FormatUint(uint64(pos), 10), i)
	}
}

// Decimal creates a decimal element with the given value.
func (ArrayConstructor) Decimal(d decimal.Decimal128) ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.Decimal(strconv.FormatUint(uint64(pos), 10), d)
	}
}

// MinKey creates a minkey element with the given value.
func (ArrayConstructor) MinKey() ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.MinKey(strconv.FormatUint(uint64(pos), 10))
	}
}

// MaxKey creates a maxkey element with the given value.
func (ArrayConstructor) MaxKey() ArrayElementFunc {
	return func(pos uint) Elementer {
		return C.MaxKey(strconv.FormatUint(uint64(pos), 10))
	}
}
