// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

// Package primitive contains types similar to Go primitives for BSON types can do not have direct
// Go primitive representations.
package primitive

import (
	"bytes"
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// Binary represents a BSON binary value.
type Binary struct {
	Subtype byte
	Data    []byte
}

// Equal compaes bp to bp2 and returns true is the are equal.
func (bp Binary) Equal(bp2 Binary) bool {
	if bp.Subtype != bp2.Subtype {
		return false
	}
	return bytes.Equal(bp.Data, bp2.Data)
}

// Undefined represents the BSON undefined value type.
type Undefined struct{}

// DateTime represents the BSON datetime value.
type DateTime int64

// Null repreesnts the BSON null value.
type Null struct{}

// Regex represents a BSON regex value.
type Regex struct {
	Pattern string
	Options string
}

func (rp Regex) String() string {
	return fmt.Sprintf(`{"pattern": "%s", "options": "%s"}`, rp.Pattern, rp.Options)
}

// Equal compaes rp to rp2 and returns true is the are equal.
func (rp Regex) Equal(rp2 Regex) bool {
	return rp.Pattern == rp2.Pattern && rp.Options == rp.Options
}

// DBPointer represents a BSON dbpointer value.
type DBPointer struct {
	DB      string
	Pointer objectid.ObjectID
}

func (d DBPointer) String() string {
	return fmt.Sprintf(`{"db": "%s", "pointer": "%s"}`, d.DB, d.Pointer)
}

// Equal compaes d to d2 and returns true is the are equal.
func (d DBPointer) Equal(d2 DBPointer) bool {
	return d.DB == d2.DB && bytes.Equal(d.Pointer[:], d2.Pointer[:])
}

// JavaScript represents a BSON JavaScript code value.
type JavaScript string

// Symbol represents a BSON symbol value.
type Symbol string

// CodeWithScope represents a BSON JavaScript code with scope value.
type CodeWithScope struct {
	Code  JavaScript
	Scope interface{}
}

func (cws CodeWithScope) String() string {
	return fmt.Sprintf(`{"code": "%s", "scope": %v}`, cws.Code, cws.Scope)
}

// Timestamp represents a BSON timestamp value.
type Timestamp struct {
	T uint32
	I uint32
}

// Equal compaes tp to tp2 and returns true is the are equal.
func (tp Timestamp) Equal(tp2 Timestamp) bool {
	return tp.T == tp2.T && tp.I == tp2.I
}

// MinKey represents the BSON minkey value.
type MinKey struct{}

// MaxKey represents the BSON maxkey value.
type MaxKey struct{}
