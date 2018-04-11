// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package wiremessage

import "github.com/mongodb/mongo-go-driver/bson"

// Msg represents the OP_MSG message of the MongoDB wire protocol.
type Msg struct {
	MsgHeader Header
	FlagBits  MsgFlag
	Sections  []Section
	Checksum  uint32
}

// MarshalWireMessage implements the Marshaler and WireMessage interfaces.
func (m Msg) MarshalWireMessage() ([]byte, error) {
	panic("not implemented")
}

// ValidateWireMessage implements the Validator and WireMessage interfaces.
func (m Msg) ValidateWireMessage() error {
	panic("not implemented")
}

// AppendWireMessage implements the Appender and WireMessage interfaces.
func (m Msg) AppendWireMessage([]byte) ([]byte, error) {
	panic("not implemented")
}

// String implements the fmt.Stringer interface.
func (m Msg) String() string {
	panic("not implemented")
}

// Len implements the WireMessage interface.
func (m Msg) Len() int {
	panic("not implemented")
}

// UnmarshalWireMessage implements the Unmarshaler interface.
func (m *Msg) UnmarshalWireMessage([]byte) error {
	panic("not implemented")
}

// MsgFlag represents the flags on an OP_MSG message.
type MsgFlag uint32

// These constants represent the individual flags on an OP_MSG message.
const (
	ChecksumPresent MsgFlag = 1 << iota
	MoreToCome

	ExhaustAllowed MsgFlag = 1 << 16
)

// Section represents a section on an OP_MSG message.
type Section interface {
	Kind() uint8
}

// SectionBody represents the kind body of an OP_MSG message.
type SectionBody struct {
	Document bson.Reader
}

// Kind implements the Section interface.
func (sb SectionBody) Kind() uint8 {
	panic("not implemented")
}

// SectionDocumentSequence represents the kind document sequence of an OP_MSG message.
type SectionDocumentSequence struct {
	Size       int32
	Identifier string
	Documents  []bson.Reader
}

// Kind implements the Section interface.
func (sds SectionDocumentSequence) Kind() uint8 {
	panic("not implemented")
}
