// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package msg

import "sync/atomic"

var globalRequestID int32

// CurrentRequestID gets the current request id.
func CurrentRequestID() int32 {
	return atomic.LoadInt32(&globalRequestID)
}

// NextRequestID gets the next request id.
func NextRequestID() int32 {
	return atomic.AddInt32(&globalRequestID, 1)
}

type opcode uint32

const (
	replyOpcode opcode = 1
	queryOpcode opcode = 2004
)

// Message represents a MongoDB message.
type Message interface {
	msg()
}

// Request is a message sent to the server.
type Request interface {
	Message
	RequestID() int32
}

// Response is a message received from the server.
type Response interface {
	Message
	ResponseTo() int32
}

func (m *Query) msg() {}
func (m *Reply) msg() {}
