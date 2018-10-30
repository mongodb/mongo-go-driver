// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"fmt"

	"github.com/mongodb/mongo-go-driver/bson/objectid"
)

// BinaryPrimitive represents a BSON binary value.
type BinaryPrimitive struct {
	Subtype byte
	Data    []byte
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

// DBPointerPrimitive represents a BSON dbpointer value.
type DBPointerPrimitive struct {
	DB      string
	Pointer objectid.ObjectID
}

func (d DBPointerPrimitive) String() string {
	return fmt.Sprintf(`{"db": "%s", "pointer": "%s"}`, d.DB, d.Pointer)
}

// JavaScriptCodePrimitive represents a BSON JavaScript code value.
type JavaScriptCodePrimitive string

// SymbolPrimitive represents a BSON symbol value.
type SymbolPrimitive string

// CodeWithScopePrimitive represents a BSON JavaScript code with scope value.
type CodeWithScopePrimitive struct {
	Code  string
	Scope *Document
}

func (cws CodeWithScopePrimitive) String() string {
	return fmt.Sprintf(`{"code": "%s", "scope": %s}`, cws.Code, cws.Scope)
}

// TimestampPrimitive represents a BSON timestamp value.
type TimestampPrimitive struct {
	T uint32
	I uint32
}

// MinKeyPrimitive represents the BSON minkey value.
type MinKeyPrimitive struct{}

// MaxKeyPrimitive represents the BSON maxkey value.
type MaxKeyPrimitive struct{}
