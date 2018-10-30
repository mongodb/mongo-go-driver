// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"encoding/binary"
	"math"
	"time"

	"github.com/mongodb/mongo-go-driver/bson/bsontype"
	"github.com/mongodb/mongo-go-driver/bson/decimal"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/mongodb/mongo-go-driver/bson/primitive"
)

var _ Embeddable = (*Documentv2)(nil)
var _ Embeddable = (*Arrayv2)(nil)

// Embeddable is the interface implemented by types that can be embedded into a Value. There are
// only two implementors of this type Document and Array.
type Embeddable interface {
	embed()
}

// EC constructs an element. It is a helper function to enable concise creation of an Element.
func EC(key string, value Valuev2) Elementv2 { return Elementv2{Key: key, Value: value} }

// Double constructs a BSON double Value.
func Double(f64 float64) Valuev2 {
	v := Valuev2{t: bsontype.Double}
	binary.LittleEndian.PutUint64(v.bootstrap[0:8], math.Float64bits(f64))
	return v
}

// String constructs a BSON string Value.
func String(str string) Valuev2 { return Valuev2{t: bsontype.String}.writestring(str) }

// Embed constructs a Value from the given Embeddable. The type will be a BSON embedded document for
// *Document, a BSON array for *Array, and a BSON null for either a nil pointer to *Document,
// *Array, or the value nil.
func Embed(embed Embeddable) Valuev2 {
	var v Valuev2
	switch tt := embed.(type) {
	case *Documentv2:
		if tt == nil {
			v.t = bsontype.Null
			break
		}
		v.t = bsontype.EmbeddedDocument
		v.primitive = tt
	case *Arrayv2:
		if tt == nil {
			v.t = bsontype.Null
			break
		}
		v.t = bsontype.Array
		v.primitive = tt
	default:
		v.t = bsontype.Null
	}
	return v
}

// EmbedElements constructs a BSON embedded document from elems. If elems is nil, an empty embedded
// document value is returned.
func EmbedElements(elems []Elementv2) Valuev2 { return Embed(NewDocumentv2(elems...)) }

// EmbedElement constructs a BSON embedded document containing one element constructed from key and
// value.
func EmbedElement(key string, value Valuev2) Valuev2 {
	return EmbedElements([]Elementv2{{Key: key, Value: value}})
}

// EmbedValues constructs a BSON array from values. If values is nil or empty, an empty array value
// is returned.
func EmbedValues(values ...Valuev2) Valuev2 { return Embed(NewArrayv2(values...)) }

// Binary constructs a BSON binary Value.
func Binary(subtype byte, data []byte) Valuev2 {
	return Valuev2{t: bsontype.Binary, primitive: primitive.Binary{Subtype: subtype, Data: data}}
}

// Undefined constructs a BSON binary Value.
func Undefined() Valuev2 { return Valuev2{t: bsontype.Undefined} }

// ObjectID constructs a BSON objectid Value.
func ObjectID(oid objectid.ObjectID) Valuev2 {
	v := Valuev2{t: bsontype.ObjectID}
	copy(v.bootstrap[0:12], oid[:])
	return v
}

// Boolean constructs a BSON boolean Value.
func Boolean(b bool) Valuev2 {
	v := Valuev2{t: bsontype.Boolean}
	if b {
		v.bootstrap[0] = 0x01
	}
	return v
}

// DateTime constructs a BSON datetime Value.
func DateTime(dt int64) Valuev2 { return Valuev2{t: bsontype.DateTime}.writei64(dt) }

// Time constructs a BSON datetime Value.
func Time(t time.Time) Valuev2 {
	return Valuev2{t: bsontype.DateTime}.writei64(t.Unix()*1e3 + int64(t.Nanosecond()/1e6))
}

// Null constructs a BSON binary Value.
func Null() Valuev2 { return Valuev2{t: bsontype.Null} }

// Regex constructs a BSON regex Value.
func Regex(pattern, options string) Valuev2 {
	regex := primitive.Regex{Pattern: pattern, Options: options}
	return Valuev2{t: bsontype.Regex, primitive: regex}
}

// DBPointer constructs a BSON dbpointer Value.
func DBPointer(ns string, ptr objectid.ObjectID) Valuev2 {
	dbptr := primitive.DBPointer{DB: ns, Pointer: ptr}
	return Valuev2{t: bsontype.DBPointer, primitive: dbptr}
}

// JavaScript constructs a BSON javascript Value.
func JavaScript(js string) Valuev2 {
	return Valuev2{t: bsontype.JavaScript}.writestring(js)
}

// Symbol constructs a BSON symbol Value.
func Symbol(symbol string) Valuev2 {
	return Valuev2{t: bsontype.Symbol}.writestring(symbol)
}

// CodeWithScope constructs a BSON code with scope Value.
func CodeWithScope(code string, scope *Documentv2) Valuev2 {
	cws := primitive.CodeWithScope{Code: primitive.JavaScript(code), Scope: scope}
	return Valuev2{t: bsontype.CodeWithScope, primitive: cws}
}

// Int32 constructs a BSON int32 Value.
func Int32(i32 int32) Valuev2 {
	v := Valuev2{t: bsontype.Int32}
	v.bootstrap[0] = byte(i32)
	v.bootstrap[1] = byte(i32 >> 8)
	v.bootstrap[2] = byte(i32 >> 16)
	v.bootstrap[3] = byte(i32 >> 24)
	return v
}

// Timestamp constructs a BSON timestamp Value.
func Timestamp(t, i uint32) Valuev2 {
	v := Valuev2{t: bsontype.Timestamp}
	v.bootstrap[0] = byte(i)
	v.bootstrap[1] = byte(i >> 8)
	v.bootstrap[2] = byte(i >> 16)
	v.bootstrap[3] = byte(i >> 24)
	v.bootstrap[4] = byte(t)
	v.bootstrap[5] = byte(t >> 8)
	v.bootstrap[6] = byte(t >> 16)
	v.bootstrap[7] = byte(t >> 24)
	return v
}

// Int64 constructs a BSON int64 Value.
func Int64(i64 int64) Valuev2 { return Valuev2{t: bsontype.Int64}.writei64(i64) }

// Decimal128 constructs a BSON decimal128 Value.
func Decimal128(d128 decimal.Decimal128) Valuev2 {
	return Valuev2{t: bsontype.Decimal128, primitive: d128}
}

// MinKey constructs a BSON minkey Value.
func MinKey() Valuev2 { return Valuev2{t: bsontype.MinKey} }

// MaxKey constructs a BSON maxkey Value.
func MaxKey() Valuev2 { return Valuev2{t: bsontype.MaxKey} }
