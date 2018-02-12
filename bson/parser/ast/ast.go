// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package ast

import (
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// Document represents a BSON Document.
type Document struct {
	Length int32
	EList  []Element
}

// Element represents an individual element of a BSON document. All element
// types implement the Element interface.
type Element interface {
	elementNode()
}

func (*FloatElement) elementNode()         {}
func (*StringElement) elementNode()        {}
func (*DocumentElement) elementNode()      {}
func (*ArrayElement) elementNode()         {}
func (*BinaryElement) elementNode()        {}
func (*UndefinedElement) elementNode()     {}
func (*ObjectIDElement) elementNode()      {}
func (*BoolElement) elementNode()          {}
func (*DateTimeElement) elementNode()      {}
func (*NullElement) elementNode()          {}
func (*RegexElement) elementNode()         {}
func (*DBPointerElement) elementNode()     {}
func (*JavaScriptElement) elementNode()    {}
func (*SymbolElement) elementNode()        {}
func (*CodeWithScopeElement) elementNode() {}
func (*Int32Element) elementNode()         {}
func (*TimestampElement) elementNode()     {}
func (*Int64Element) elementNode()         {}
func (*DecimalElement) elementNode()       {}
func (*MinKeyElement) elementNode()        {}
func (*MaxKeyElement) elementNode()        {}

// FloatElement represents a BSON double element.
type FloatElement struct {
	Name   *ElementKeyName
	Double float64
}

// StringElement represents a BSON string element.
type StringElement struct {
	Name   *ElementKeyName
	String string
}

// DocumentElement represents a BSON subdocument element.
type DocumentElement struct {
	Name     *ElementKeyName
	Document *Document
}

// ArrayElement represents a BSON array element.
type ArrayElement struct {
	Name  *ElementKeyName
	Array *Document
}

// BinaryElement represents a BSON binary element.
type BinaryElement struct {
	Name   *ElementKeyName
	Binary *Binary
}

// UndefinedElement represents a BSON undefined element.
type UndefinedElement struct {
	Name *ElementKeyName
}

// ObjectIDElement represents a BSON objectID element.
type ObjectIDElement struct {
	Name *ElementKeyName
	ID   objectid.ObjectID
}

// BoolElement represents a BSON boolean element.
type BoolElement struct {
	Name *ElementKeyName
	Bool bool
}

// DateTimeElement represents a BSON datetime element.
type DateTimeElement struct {
	Name *ElementKeyName
	// TODO(skriptble): This should be an actual time.Time value
	DateTime int64
}

// NullElement represents a BSON null element.
type NullElement struct {
	Name *ElementKeyName
}

// RegexElement represents a BSON regex element.
type RegexElement struct {
	Name         *ElementKeyName
	RegexPattern *CString
	RegexOptions *CString
}

// DBPointerElement represents a BSON db pointer element.
type DBPointerElement struct {
	Name    *ElementKeyName
	String  string
	Pointer objectid.ObjectID
}

// JavaScriptElement represents a BSON JavaScript element.
type JavaScriptElement struct {
	Name   *ElementKeyName
	String string
}

// SymbolElement represents a BSON symbol element.
type SymbolElement struct {
	Name   *ElementKeyName
	String string
}

// CodeWithScopeElement represents a BSON JavaScript with scope element.
type CodeWithScopeElement struct {
	Name          *ElementKeyName
	CodeWithScope *CodeWithScope
}

// Int32Element represents a BSON int32 element.
type Int32Element struct {
	Name  *ElementKeyName
	Int32 int32
}

// TimestampElement represents a BSON timestamp element.
type TimestampElement struct {
	Name      *ElementKeyName
	Timestamp uint64
}

// Int64Element represents a BSON int64 element.
type Int64Element struct {
	Name  *ElementKeyName
	Int64 int64
}

// DecimalElement represents a BSON Decimal128 element.
//
// TODO(skriptble): Borrowing the Decimal128 implementation from mgo/bson
// for now until we write a new implementation, preferably using the math/big
// package and providing a way to return a big.Float.
type DecimalElement struct {
	Name       *ElementKeyName
	Decimal128 decimal.Decimal128
}

// MinKeyElement represents a BSON min key element.
type MinKeyElement struct {
	Name *ElementKeyName
}

// MaxKeyElement represents a BSON max key element.
type MaxKeyElement struct {
	Name *ElementKeyName
}

// ElementKeyName represents the key for a BSON Document element.
type ElementKeyName struct {
	Key string
}

// String implements the fmt.Stringer interface.
func (ekn *ElementKeyName) String() string {
	if ekn == nil {
		return "<nil>"
	}
	return fmt.Sprintf("&ElementKeyName{Key:%s}", ekn.Key)
}

// GoString implements the fmt.GoStringer interface.
func (ekn *ElementKeyName) GoString() string {
	if ekn == nil {
		return "<nil>"
	}
	return fmt.Sprintf("&ElementKeyName{Key:%s}", ekn.Key)
}

// CString represents a BSON cstring.
type CString struct {
	String string
}

// Binary represents a BSON binary node.
type Binary struct {
	Subtype BinarySubtype
	Data    []byte
}

// String implements the fmt.Stringer interface.
func (b *Binary) String() string {
	if b == nil {
		return "<nil>"
	}
	return fmt.Sprintf("&Binary{Subtype:%d, Data:%#v}", b.Subtype, b.Data)
}

// GoString implements the fmt.GoStringer interface.
func (b *Binary) GoString() string {
	if b == nil {
		return "<nil>"
	}
	return fmt.Sprintf("&Binary{Subtype:%d, Data:%#v}", b.Subtype, b.Data)
}

// BinarySubtype describes the subtype of a Binary node.
type BinarySubtype byte

// The possible Binary Subtypes.
const (
	SubtypeGeneric     BinarySubtype = '\x00' // Generic, default
	SubtypeFunction    BinarySubtype = '\x01' // Function
	SubtypeBinaryOld   BinarySubtype = '\x02' // Old Binary, prefixed with length
	SubtypeUUIDOld     BinarySubtype = '\x03' // Old UUID
	SubtypeUUID        BinarySubtype = '\x04' // UUID
	SubtypeMD5         BinarySubtype = '\x05' // MD5
	SubtypeUserDefined BinarySubtype = '\x80' // User defined types, anything greater than this
)

// CodeWithScope represents a BSON JavaScript with scope node.
type CodeWithScope struct {
	String   string
	Document *Document
}
