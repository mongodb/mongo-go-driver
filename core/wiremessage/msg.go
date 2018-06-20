// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package wiremessage

import (
	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
)

// Msg represents the OP_MSG message of the MongoDB wire protocol.
type Msg struct {
	MsgHeader Header
	FlagBits  MsgFlag
	Sections  []Section
	Checksum  uint32
}

// MarshalWireMessage implements the Marshaler and WireMessage interfaces.
func (m Msg) MarshalWireMessage() ([]byte, error) {
	b := make([]byte, 0, m.Len())
	return m.AppendWireMessage(b)
}

// ValidateWireMessage implements the Validator and WireMessage interfaces.
func (m Msg) ValidateWireMessage() error {
	if int(m.MsgHeader.MessageLength) != m.Len() {
		return errors.New("incorrect header: message length is not correct")
	}
	if m.MsgHeader.OpCode != OpMsg {
		return errors.New("incorrect header: opcode is not OpMsg")
	}

	return nil
}

// AppendWireMessage implements the Appender and WireMessage interfaces.
//
// AppendWireMesssage will set the MessageLength property of the MsgHeader if it is zero. It will also set the Opcode
// to OP_MSG if it is zero. If either of these properties are non-zero and not correct, this method will return both the
// []byte with the wire message appended to it and an invalid header error.
func (m Msg) AppendWireMessage(b []byte) ([]byte, error) {
	var err error
	err = m.MsgHeader.SetDefaults(m.Len(), OpMsg)

	b = m.MsgHeader.AppendHeader(b)
	b = appendInt32(b, int32(m.FlagBits))

	for _, section := range m.Sections {
		b = section.AppendSection(b)
	}

	return b, err
}

// String implements the fmt.Stringer interface.
func (m Msg) String() string {
	panic("not implemented")
}

// Len implements the WireMessage interface.
func (m Msg) Len() int {
	// Header + FlagBits + len of each section + optional checksum
	totalLen := 16 + 32 // header and flag bits

	for _, section := range m.Sections {
		totalLen += section.Len()
	}

	if m.FlagBits&ChecksumPresent > 0 {
		totalLen += 4
	}

	return totalLen
}

// UnmarshalWireMessage implements the Unmarshaler interface.
func (m *Msg) UnmarshalWireMessage(b []byte) error {
	var err error

	m.MsgHeader, err = ReadHeader(b, 0)
	if err != nil {
		return err
	}
	if len(b) < int(m.MsgHeader.MessageLength) {
		return Error{
			Type:    ErrOpMsg,
			Message: "[]byte too small",
		}
	}

	m.FlagBits = MsgFlag(readInt32(b, 16))

	// read each section
	sectionBytes := m.MsgHeader.MessageLength - 16 - 4 // number of bytes taken up by sections
	hasChecksum := m.FlagBits&ChecksumPresent > 0
	if hasChecksum {
		sectionBytes -= 4 // 4 bytes at end for checksum
	}

	m.Sections = make([]Section, 0)
	position := 20 // position to read from
	for sectionBytes > 0 {
		sectionType := SectionType(b[position])
		position++

		switch sectionType {
		case SingleDocument:
			rdr, size, err := readDocument(b, int32(position))
			if err.Message != "" {
				err.Type = ErrOpMsg
				return err
			}

			position += size
			sb := SectionBody{
				Document: rdr,
			}
			sectionBytes -= int32(sb.Len())
			m.Sections = append(m.Sections, sb)
		case DocumentSequence:
			sds := SectionDocumentSequence{}
			sds.Size = readInt32(b, int32(position))
			position += 4

			identifier, err := readCString(b, int32(position))
			if err != nil {
				return err
			}

			sds.Identifier = identifier
			position += len(identifier) + 1 // +1 for \0

			// length of documents to read
			// sequenceLen - 4 (size) - identifierLength (including \0)
			docsLen := int(sds.Size) - 4 - len(identifier) - 1
			for docsLen > 0 {
				rdr, size, err := readDocument(b, int32(position))
				if err.Message != "" {
					err.Type = ErrOpMsg
					return err
				}

				position += size
				sds.Documents = append(sds.Documents, rdr)
			}

			sectionBytes -= int32(sds.Len())
			m.Sections = append(m.Sections, sds)
		}
	}

	if hasChecksum {
		m.Checksum = uint32(readInt32(b, int32(position)))
	}

	return nil
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
	Kind() SectionType
	Len() int
	AppendSection([]byte) []byte
}

// SectionBody represents the kind body of an OP_MSG message.
type SectionBody struct {
	Document bson.Reader
}

// Kind implements the Section interface.
func (sb SectionBody) Kind() SectionType {
	return SingleDocument
}

// Len implements the Section interface
func (sb SectionBody) Len() int {
	return len(sb.Document)
}

// AppendSection implements the Section interface.
func (sb SectionBody) AppendSection(dest []byte) []byte {
	dest = append(dest, sb.Document...)
	return dest
}

// SectionDocumentSequence represents the kind document sequence of an OP_MSG message.
type SectionDocumentSequence struct {
	Size       int32
	Identifier string
	Documents  []bson.Reader
}

// Kind implements the Section interface.
func (sds SectionDocumentSequence) Kind() SectionType {
	return DocumentSequence
}

// Len implements the Section interface
func (sds SectionDocumentSequence) Len() int {
	// Size + Identifier + 1 (null terminator) + totalDocLen
	totalDocLen := 0
	for _, doc := range sds.Documents {
		totalDocLen += len(doc)
	}

	return 4 + len(sds.Identifier) + 1 + totalDocLen
}

// AppendSection implements the Section interface
func (sds SectionDocumentSequence) AppendSection(dest []byte) []byte {
	dest = appendInt32(dest, sds.Size)
	dest = appendCString(dest, sds.Identifier)

	for _, doc := range sds.Documents {
		dest = append(dest, doc...)
	}

	return dest
}

// SectionType represents the type for 1 section in an OP_MSG
type SectionType uint8

// These constants represent the individual section types for a section in an OP_MSG
const (
	SingleDocument SectionType = iota
	DocumentSequence
)

// OPMSG_WIRE_VERSION is the minimum wire version needed to use OP_MSG
const OPMSG_WIRE_VERSION = 6
