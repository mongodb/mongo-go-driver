// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// BinaryPrimitive represents a BSON binary value.
type BinaryPrimitive struct {
	Subtype byte
	Data    []byte
}

// Equal compaes bp to bp2 and returns true is the are equal.
func (bp BinaryPrimitive) Equal(bp2 BinaryPrimitive) bool {
	if bp.Subtype != bp2.Subtype {
		return false
	}
	return bytes.Equal(bp.Data, bp2.Data)
}

// UndefinedPrimitive represents the BSON undefined value type.
type UndefinedPrimitive struct{}

// DateTimePrimitive represents the BSON datetime value.
type DateTimePrimitive int64

// NullPrimitive repreesnts the BSON null value.
type NullPrimitive struct{}

// RegexPrimitive represents a BSON regex value.
type RegexPrimitive struct {
	Pattern string
	Options string
}

func (rp RegexPrimitive) String() string {
	return fmt.Sprintf(`{"pattern": "%s", "options": "%s"}`, rp.Pattern, rp.Options)
}

// Equal compaes rp to rp2 and returns true is the are equal.
func (rp RegexPrimitive) Equal(rp2 RegexPrimitive) bool {
	return rp.Pattern == rp2.Pattern && rp.Options == rp.Options
}

// DBPointerPrimitive represents a BSON dbpointer value.
type DBPointerPrimitive struct {
	DB      string
	Pointer objectid.ObjectID
}

func (d DBPointerPrimitive) String() string {
	return fmt.Sprintf(`{"db": "%s", "pointer": "%s"}`, d.DB, d.Pointer)
}

// Equal compaes d to d2 and returns true is the are equal.
func (d DBPointerPrimitive) Equal(d2 DBPointerPrimitive) bool {
	return d.DB == d2.DB && bytes.Equal(d.Pointer[:], d2.Pointer[:])
}

// JavaScriptCodePrimitive represents a BSON JavaScript code value.
type JavaScriptCodePrimitive string

// SymbolPrimitive represents a BSON symbol value.
type SymbolPrimitive string

// CodeWithScopePrimitive represents a BSON JavaScript code with scope value.
type CodeWithScopePrimitive struct {
	Code    JavaScriptCodePrimitive
	Scope   *Document
	Scopev2 *Documentv2
}

func (cws CodeWithScopePrimitive) String() string {
	return fmt.Sprintf(`{"code": "%s", "scope": %s}`, cws.Code, cws.Scope)
}

// Equal compaes cws to cws2 and returns true is the are equal.
func (cws CodeWithScopePrimitive) Equal(cws2 CodeWithScopePrimitive) bool {
	return cws.Code == cws2.Code && cws.Scope.Equal(cws2.Scope)
}

// TimestampPrimitive represents a BSON timestamp value.
type TimestampPrimitive struct {
	T uint32
	I uint32
}

// Equal compaes tp to tp2 and returns true is the are equal.
func (tp TimestampPrimitive) Equal(tp2 TimestampPrimitive) bool {
	return tp.T == tp2.T && tp.I == tp2.I
}

// MinKeyPrimitive represents the BSON minkey value.
type MinKeyPrimitive struct{}

// MaxKeyPrimitive represents the BSON maxkey value.
type MaxKeyPrimitive struct{}
