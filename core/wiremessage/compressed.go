// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package wiremessage

// Compressed represents the OP_COMPRESSED message of the MongoDB wire protocol.
type Compressed struct {
	MsgHeader         Header
	OriginalOpCode    OpCode
	UncompressedSize  int32
	CompressorID      CompressorID
	CompressedMessage []byte
}

// MarshalWireMessage implements the Marshaler and WireMessage interfaces.
func (c Compressed) MarshalWireMessage() ([]byte, error) {
	panic("not implemented")
}

// ValidateWireMessage implements the Validator and WireMessage interfaces.
func (c Compressed) ValidateWireMessage() error {
	panic("not implemented")
}

// AppendWireMessage implements the Appender and WireMessage interfaces.
func (c Compressed) AppendWireMessage([]byte) ([]byte, error) {
	panic("not implemented")
}

// String implements the fmt.Stringer interface.
func (c Compressed) String() string {
	panic("not implemented")
}

// Len implements the WireMessage interface.
func (c Compressed) Len() int {
	panic("not implemented")
}

// UnmarshalWireMessage implements the Unmarshaler interface.
func (c *Compressed) UnmarshalWireMessage([]byte) error {
	panic("not implemented")
}

// CompressorID is the ID for each type of Compressor.
type CompressorID uint8

// These constants represent the individual compressor IDs for an OP_COMPRESSED.
const (
	CompressorNoOp CompressorID = iota
	CompressorSnappy
	CompressorZLip
)
