// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package wiremessage

// GetMore represents the OP_GET_MORE message of the MongoDB wire protocol.
type GetMore struct {
	MsgHeader          Header
	FullCollectionName string
	NumberToReturn     int32
	CursorID           int64
}

// MarshalWireMessage implements the Marshaler and WireMessage interfaces.
func (gm GetMore) MarshalWireMessage() ([]byte, error) {
	panic("not implemented")
}

// ValidateWireMessage implements the Validator and WireMessage interfaces.
func (gm GetMore) ValidateWireMessage() error {
	panic("not implemented")
}

// AppendWireMessage implements the Appender and WireMessage interfaces.
func (gm GetMore) AppendWireMessage([]byte) ([]byte, error) {
	panic("not implemented")
}

// String implements the fmt.Stringer interface.
func (gm GetMore) String() string {
	panic("not implemented")
}

// Len implements the WireMessage interface.
func (gm GetMore) Len() int {
	panic("not implemented")
}

// UnmarshalWireMessage implements the Unmarshaler interface.
func (gm *GetMore) UnmarshalWireMessage([]byte) error {
	panic("not implemented")
}
