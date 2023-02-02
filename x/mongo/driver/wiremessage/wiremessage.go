// Copyright (C) MongoDB, Inc. 2022-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package wiremessage

import (
	"strings"
	"sync/atomic"
)

// WireMessage represents a MongoDB wire message in binary form.
type WireMessage []byte

var globalRequestID int32

// CurrentRequestID returns the current request ID.
func CurrentRequestID() int32 { return atomic.LoadInt32(&globalRequestID) }

// NextRequestID returns the next request ID.
func NextRequestID() int32 { return atomic.AddInt32(&globalRequestID, 1) }

// OpCode represents a MongoDB wire protocol opcode.
type OpCode int32

// These constants are the valid opcodes for the version of the wireprotocol
// supported by this library. The skipped OpCodes are historical OpCodes that
// are no longer used.
const (
	OpReply        OpCode = 1
	_              OpCode = 1001
	OpUpdate       OpCode = 2001
	OpInsert       OpCode = 2002
	_              OpCode = 2003
	OpQuery        OpCode = 2004
	OpGetMore      OpCode = 2005
	OpDelete       OpCode = 2006
	OpKillCursors  OpCode = 2007
	OpCommand      OpCode = 2010
	OpCommandReply OpCode = 2011
	OpCompressed   OpCode = 2012
	OpMsg          OpCode = 2013
)

// String implements the fmt.Stringer interface.
func (oc OpCode) String() string {
	switch oc {
	case OpReply:
		return "OP_REPLY"
	case OpUpdate:
		return "OP_UPDATE"
	case OpInsert:
		return "OP_INSERT"
	case OpQuery:
		return "OP_QUERY"
	case OpGetMore:
		return "OP_GET_MORE"
	case OpDelete:
		return "OP_DELETE"
	case OpKillCursors:
		return "OP_KILL_CURSORS"
	case OpCommand:
		return "OP_COMMAND"
	case OpCommandReply:
		return "OP_COMMANDREPLY"
	case OpCompressed:
		return "OP_COMPRESSED"
	case OpMsg:
		return "OP_MSG"
	default:
		return "<invalid opcode>"
	}
}

// QueryFlag represents the flags on an OP_QUERY message.
type QueryFlag int32

// These constants represent the individual flags on an OP_QUERY message.
const (
	_ QueryFlag = 1 << iota
	TailableCursor
	SecondaryOK
	OplogReplay
	NoCursorTimeout
	AwaitData
	Exhaust
	Partial
)

// String implements the fmt.Stringer interface.
func (qf QueryFlag) String() string {
	strs := make([]string, 0)
	if qf&TailableCursor == TailableCursor {
		strs = append(strs, "TailableCursor")
	}
	if qf&SecondaryOK == SecondaryOK {
		strs = append(strs, "SecondaryOK")
	}
	if qf&OplogReplay == OplogReplay {
		strs = append(strs, "OplogReplay")
	}
	if qf&NoCursorTimeout == NoCursorTimeout {
		strs = append(strs, "NoCursorTimeout")
	}
	if qf&AwaitData == AwaitData {
		strs = append(strs, "AwaitData")
	}
	if qf&Exhaust == Exhaust {
		strs = append(strs, "Exhaust")
	}
	if qf&Partial == Partial {
		strs = append(strs, "Partial")
	}
	str := "["
	str += strings.Join(strs, ", ")
	str += "]"
	return str
}

// MsgFlag represents the flags on an OP_MSG message.
type MsgFlag uint32

// These constants represent the individual flags on an OP_MSG message.
const (
	ChecksumPresent MsgFlag = 1 << iota
	MoreToCome

	ExhaustAllowed MsgFlag = 1 << 16
)

// ReplyFlag represents the flags of an OP_REPLY message.
type ReplyFlag int32

// These constants represent the individual flags of an OP_REPLY message.
const (
	CursorNotFound ReplyFlag = 1 << iota
	QueryFailure
	ShardConfigStale
	AwaitCapable
)

// String implements the fmt.Stringer interface.
func (rf ReplyFlag) String() string {
	strs := make([]string, 0)
	if rf&CursorNotFound == CursorNotFound {
		strs = append(strs, "CursorNotFound")
	}
	if rf&QueryFailure == QueryFailure {
		strs = append(strs, "QueryFailure")
	}
	if rf&ShardConfigStale == ShardConfigStale {
		strs = append(strs, "ShardConfigStale")
	}
	if rf&AwaitCapable == AwaitCapable {
		strs = append(strs, "AwaitCapable")
	}
	str := "["
	str += strings.Join(strs, ", ")
	str += "]"
	return str
}

// SectionType represents the type for 1 section in an OP_MSG
type SectionType uint8

// These constants represent the individual section types for a section in an OP_MSG
const (
	SingleDocument SectionType = iota
	DocumentSequence
)

// OpmsgWireVersion is the minimum wire version needed to use OP_MSG
const OpmsgWireVersion = 6

// CompressorID is the ID for each type of Compressor.
type CompressorID uint8

// These constants represent the individual compressor IDs for an OP_COMPRESSED.
const (
	CompressorNoOp CompressorID = iota
	CompressorSnappy
	CompressorZLib
	CompressorZstd
)

// String implements the fmt.Stringer interface.
func (id CompressorID) String() string {
	switch id {
	case CompressorNoOp:
		return "CompressorNoOp"
	case CompressorSnappy:
		return "CompressorSnappy"
	case CompressorZLib:
		return "CompressorZLib"
	case CompressorZstd:
		return "CompressorZstd"
	default:
		return "CompressorInvalid"
	}
}

const (
	// DefaultZlibLevel is the default level for zlib compression
	DefaultZlibLevel = 6
	// DefaultZstdLevel is the default level for zstd compression.
	// Matches https://github.com/wiredtiger/wiredtiger/blob/f08bc4b18612ef95a39b12166abcccf207f91596/ext/compressors/zstd/zstd_compress.c#L299
	DefaultZstdLevel = 6
)

// ParseHeader parses a wire message header.
func ParseHeader(hdr []byte) (length, requestID, responseTo int32, opcode OpCode, ok bool) {
	if len(hdr) < 16 {
		return 0, 0, 0, 0, false
	}
	length = (int32(hdr[0]) | int32(hdr[1])<<8 | int32(hdr[2])<<16 | int32(hdr[3])<<24)
	requestID = (int32(hdr[4]) | int32(hdr[5])<<8 | int32(hdr[6])<<16 | int32(hdr[7])<<24)
	responseTo = (int32(hdr[8]) | int32(hdr[9])<<8 | int32(hdr[10])<<16 | int32(hdr[11])<<24)
	opcode = OpCode(int32(hdr[12]) | int32(hdr[13])<<8 | int32(hdr[14])<<16 | int32(hdr[15])<<24)
	return length, requestID, responseTo, opcode, true
}
